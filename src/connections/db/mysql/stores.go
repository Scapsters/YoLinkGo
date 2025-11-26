package mysql

import (
	"com/connections/db"
	"com/data"
	"com/logs"
	"com/utils"
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/samborkent/uuidv7"
)

// Generic MySQL Store. Instanatiations require a couple assertions:
// When returning properties in a list, or doing anything, it must always be in the same order.
// SQL Queries, anything. All in the same order every time.
// The ID comes first in this order.
// The main way this order is coordinated is via the data structs "Spread" and related functions. These methods only respect that order.
type MySQLStore[T data.Spreadable, S data.HasIDGetterAndSpreadable[S], F data.Spreadable] struct {
	db               *sql.DB
	tableName        string
	tableCreationSQL string
	tableColumns     []string
	primaryKey       string
}

func (s *MySQLStore[T, S, F]) Add(ctx context.Context, item T) (string, error) {
	// Build query
	id := uuidv7.New().String()
	sqlArgs := append([]any{id}, item.Spread()...)
	sqlColumns := strings.Join(s.tableColumns, ", ")
	sqlPlaceholders := strings.Repeat("?, ", len(s.tableColumns)-1) + "?"
	sqlQuery := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", s.tableName, sqlColumns, sqlPlaceholders)

	// Execute query
	sqlctx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()
	_, err := s.db.ExecContext(sqlctx, sqlQuery, sqlArgs...)
	if err != nil {
		return "", fmt.Errorf("error inserting into %s with values %v: %w", s.tableName, item.Spread(), err)
	}
	return id, nil
}
func (s *MySQLStore[T, S, F]) Get(ctx context.Context, filter F) *data.IterablePaginatedData[S] {
	// Build conditions
	args := []any{}
	conditions := []string{}
	for index, columnName := range s.tableColumns {
		filterInterface := filter.Spread()[index]
		filterValue := reflect.ValueOf(filterInterface)
		if filterValue.IsNil() {
			continue
		}
		conditions = append(conditions, columnName+" = ?")
		args = append(args, filterInterface)
	}

	// Combine into query
	query := "SELECT * FROM " + s.tableName
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Add pagination condition
	if len(conditions) > 0 {
		query += " AND "
	} else {
		query += " WHERE "
	}
	query += fmt.Sprintf("%v > ? ORDER BY %v LIMIT ?", s.primaryKey, s.primaryKey)

	paginator := newSQLIterablePaginatedData[S](s.db, query, args)
	return &paginator
}
func (s *MySQLStore[T, S, F]) Delete(ctx context.Context, storeItem S) error {
	sqlctx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()

	res, err := s.db.ExecContext(sqlctx, fmt.Sprintf("DELETE FROM %s WHERE id = ?", s.tableName), storeItem.GetID())
	if err != nil {
		return fmt.Errorf("error deleting id %v from table %s: %w", storeItem.GetID(), s.tableName, err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking rows affected for id %v in table %s: %w", storeItem.GetID(), s.tableName, err)
	}
	if rows == 0 {
		return fmt.Errorf("no rows deleted for id %v in table %s", storeItem.GetID(), s.tableName)
	}

	return nil
}
func (s *MySQLStore[T, S, F]) Setup(ctx context.Context, isDestructive bool) error {
	sqlctx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()

	if isDestructive {
		sqlctx, cancel := context.WithTimeout(ctx, RequestTimeout)
		defer cancel()
		_, err := s.db.ExecContext(sqlctx, `SET FOREIGN_KEY_CHECKS = 0`)
		if err != nil {
			return fmt.Errorf("error disabling FK checks: %w", err)
		}
		sqlctx, cancel = context.WithTimeout(ctx, RequestTimeout)
		defer cancel()
		_, err = s.db.ExecContext(sqlctx, `DROP TABLE IF EXISTS `+s.tableName)
		if err != nil {
			return fmt.Errorf("error dropping logs table: %w", err)
		}
		sqlctx, cancel = context.WithTimeout(ctx, RequestTimeout)
		defer cancel()
		_, err = s.db.ExecContext(sqlctx, `SET FOREIGN_KEY_CHECKS = 1`)
		if err != nil {
			return fmt.Errorf("error enabling FK checks: %w", err)
		}
	}

	// Create table
	_, err := s.db.ExecContext(sqlctx, s.tableCreationSQL)
	if err != nil {
		return fmt.Errorf("error creating table %s: %w", s.tableName, err)
	}
	return nil
}
func (s *MySQLStore[T, S, F]) Export(ctx context.Context, storeItems *data.IterablePaginatedData[S]) error {
	// Ensure exports directory exists
	var OwnerReadWriteExecuteAndOthersReadExecute = 0755
	err := os.MkdirAll(db.EXPORT_DIR, os.FileMode(OwnerReadWriteExecuteAndOthersReadExecute))
	if err != nil {
		return fmt.Errorf("error creating export directory: %w", err)
	}

	// Generate filename
	now := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("%s/%s_%v.csv", db.EXPORT_DIR, s.tableName, now)

	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating export file: %w", err)
	}
	defer logs.LogErrorsWithContext(ctx, f.Close, fmt.Sprintf("error closing file %v", filename))

	writer := csv.NewWriter(f)
	defer writer.Flush()

	// Write CSV header
	err = writer.Write(s.tableColumns)
	if err != nil {
		return fmt.Errorf("error writing CSV header: %w", err)
	}

	// Write each row
	for {
		item, err := storeItems.Next(ctx)
		if err != nil {
			logs.ErrorWithContext(ctx, "Error while fetching item from %v while exporting: %v", s.tableName, err)
		}
		if item == nil {
			break
		}

		err = writer.Write((*item).SpreadForExport())
		if err != nil {
			logs.ErrorWithContext(ctx, "Error while writing csv row with data %v: %v", item, err)
		}
	}

	return nil
}

type MySQLTimestampedDataStore[T data.Spreadable, S data.HasIDGetterAndSpreadable[S], F data.Spreadable] struct {
	MySQLStore[T, S, F]

	timestampKey string
}

func (s *MySQLTimestampedDataStore[T, S, F]) GetInTimeRange(ctx context.Context, filter F, startTime *int64, endTime *int64) *data.IterablePaginatedData[S] {
	// Build conditions
	args := []any{}
	conditions := []string{}
	for index, columnName := range s.tableColumns {
		filterValue := filter.Spread()[index]
		if filterValue == nil {
			continue
		}
		conditions = append(conditions, columnName+" = ?")
		args = append(args, filterValue)
	}
	if startTime != nil {
		conditions = append(conditions, s.timestampKey+" > ?")
		args = append(args, startTime)
	}
	if endTime != nil {
		conditions = append(conditions, s.timestampKey+" < ?")
		args = append(args, startTime)
	}

	// Combine into query
	query := "SELECT * FROM " + s.tableName
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Add pagination condition
	if len(conditions) > 0 {
		query += " AND "
	} else {
		query += " WHERE "
	}
	query += fmt.Sprintf("%v > ? ORDER BY %v LIMIT ?", s.primaryKey, s.primaryKey)

	paginator := newSQLIterablePaginatedData[S](s.db, query, args)
	return &paginator
}

type MySQLEditableStore[T data.Spreadable, S data.HasIDGetterAndSpreadable[S], F data.Spreadable] struct {
	MySQLStore[T, S, F]
}

func (s *MySQLEditableStore[T, S, F]) Edit(ctx context.Context, storeItem S) error {
	sqlEdits := strings.TrimSuffix(strings.Join(s.tableColumns, " = ?, "), ", ")

	sqlctx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()
	result, err := s.db.ExecContext(
		sqlctx,
		fmt.Sprintf(
			`UPDATE %v SET %v WHERE %v = ?`,
			s.tableName, sqlEdits, s.primaryKey,
		),
		append(storeItem.Spread(), storeItem.GetID())...,
	)
	if err != nil {
		return fmt.Errorf("error editing item %v in table %v: %w", storeItem, s.tableName, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected while editing item %v in table %v: %w", storeItem, s.tableName, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("no rows found while exiting item %v in table %v: %w", storeItem, s.tableName, err)
	}
	return nil
}

type MySQLClosableStore[T data.Spreadable, S data.HasIDGetterAndSpreadable[S], F data.Spreadable] struct {
	MySQLTimestampedDataStore[T, S, F]

	closeKey string
}

func (s *MySQLClosableStore[T, S, F]) Close(ctx context.Context, storeItem S) error {
	sqlctx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()
	result, err := s.db.ExecContext(
		sqlctx,
		fmt.Sprintf(
			`UPDATE %v SET %v = ? WHERE %v = ?`,
			s.tableName, s.closeKey, s.primaryKey,
		),
		utils.TimeSeconds(),
		storeItem.GetID(),
	)
	if err != nil {
		return fmt.Errorf("error editing item %v in table %v: %w", storeItem, s.tableName, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected while editing item %v in table %v: %w", storeItem, s.tableName, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("no rows found while exiting item %v in table %v: %w", storeItem, s.tableName, err)
	}
	return nil
}

// Helper function for get methods.
func newSQLIterablePaginatedData[T data.HasIDGetterAndSpreadable[T]](db *sql.DB, query string, args []any) data.IterablePaginatedData[T] {
	// Define pagination function
	return data.NewIterablePaginatedData(
		func(ctx context.Context, lastID *string) ([]T, *string, error) {
			// Peform query. First query has no pagination filter, subsequent queries use the last id from the previous filter
			var filterID string
			if lastID != nil {
				filterID = *lastID
			}
			context, cancel := context.WithTimeout(ctx, RequestTimeout)
			defer cancel()
			rows, err := db.QueryContext(context, query, append(args, filterID, data.PAGE_SIZE)...)
			if err != nil {
				return nil, nil, fmt.Errorf("error running query %v with args %v: %w", query, append(args, filterID, data.PAGE_SIZE), err)
			}
			defer logs.LogErrorsWithContext(ctx, rows.Close, fmt.Sprintf("error closing rows for query %v and lastID %v", query, lastID))

			// Scan all results into the next "page" of data to store
			var items []T
			for rows.Next() {
				var emptyItem T
				item, addresses := emptyItem.SpreadAddresses()
				err := rows.Scan(addresses...)
				if err != nil {
					return nil, nil, fmt.Errorf("error scanning while paginating: %w", err)
				}
				items = append(items, *item)
			}
			err = rows.Err()
			if err != nil {
				return nil, nil, fmt.Errorf("error in rows: %w", err)
			}
			if len(items) == 0 {
				return []T{}, nil, nil
			}
			lastItem := items[len(items)-1]
			id := lastItem.GetID()
			return items, &id, nil
		},
	)
}
