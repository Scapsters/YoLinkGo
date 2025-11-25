package mysql

import (
	"com/connections/db"
	"com/data"
	"com/logs"
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/samborkent/uuidv7"
)

func dropTable(ctx context.Context, db *sql.DB, tableName string) error {
	sqlctx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()
	_, err := db.ExecContext(sqlctx, `SET FOREIGN_KEY_CHECKS = 0`)
	if err != nil {
		return fmt.Errorf("error disabling FK checks: %w", err)
	}
	sqlctx, cancel = context.WithTimeout(ctx, RequestTimeout)
	defer cancel()
	_, err = db.ExecContext(sqlctx, `DROP TABLE IF EXISTS ` + tableName)
	if err != nil {
		return fmt.Errorf("error dropping logs table: %w", err)
	}
	sqlctx, cancel = context.WithTimeout(ctx, RequestTimeout)
	defer cancel()
	_, err = db.ExecContext(sqlctx, `SET FOREIGN_KEY_CHECKS = 1`)
	if err != nil {
		return fmt.Errorf("error enabling FK checks: %w", err)
	}
	return nil
}

func sqlCreateTableHelper(ctx context.Context, db *sql.DB, tableSQL string, tableName string) error {
	sqlctx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()

	_, err := db.ExecContext(sqlctx, tableSQL)
	if err != nil {
		return fmt.Errorf("error creating table %s: %w", tableName, err)
	}
	return nil
}

func sqlInsertHelper[T any](ctx context.Context, db *sql.DB, table string, columns []string, values []any) (string, error) {
	sqlctx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()

	id := uuidv7.New().String()
	args := append([]any{id}, values...) // assume first column is ID
	colList := strings.Join(columns, ", ")
	placeholders := strings.Repeat("?, ", len(columns)-1) + "?"

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, colList, placeholders)

	_, err := db.ExecContext(sqlctx, query, args...)
	if err != nil {
		return "", fmt.Errorf("error inserting into %s with values %v: %w", table, values, err)
	}

	return id, nil
}

func sqlGetHelper[T data.HasIDGetter](
	db *sql.DB,
	fieldNames []string,
	fieldValues []any,
	tableName string,
	primaryKey string,
	scanIntoItem func(rows *sql.Rows) (*T, error),
) *data.IterablePaginatedData[T] {
	// Build conditions
	args := []any{}
	conditions := []string{}
	for index, value := range fieldValues {
		if value == nil {
			continue
		}
		conditions = append(conditions, fieldNames[index])
		args = append(args, value)
	}
	
	// Combine into query
	query := "SELECT * FROM " + tableName
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Add pagination condition
	if len(conditions) > 0 {
		query += " AND "
	} else {
		query += " WHERE "
	}
	query += fmt.Sprintf("%v > ? ORDER BY %v LIMIT ?", primaryKey, primaryKey)
	
	// Define pagination function
	getPage := func(ctx context.Context, lastID *string) ([]T, *string, error) {
		var filterID string
		if lastID != nil {
			filterID = *lastID
		}
		context, cancel := context.WithTimeout(ctx, RequestTimeout)
		defer cancel()
		rows, err := db.QueryContext(context, query, append(args, filterID, data.PAGE_SIZE)...)
		if err != nil {
			return nil, nil, fmt.Errorf("error querying %v with filter %v: %w", tableName, fieldValues, err)
		}
		defer logs.LogErrorsWithContext(ctx, rows.Close, fmt.Sprintf("error closing rows for query %v and lastID %v", query, lastID))

		var items []T
		for rows.Next() {
			item, err := scanIntoItem(rows)
			if err != nil {
				return nil, nil, err
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
	}

	return &data.IterablePaginatedData[T]{GetPage: getPage}
}

func sqlDeleteHelper[T any](ctx context.Context, db *sql.DB, table string, id string) error {
	sqlctx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()

	res, err := db.ExecContext(sqlctx, fmt.Sprintf("DELETE FROM %s WHERE id = ?", table), id)
	if err != nil {
		return fmt.Errorf("error deleting id %v from table %s: %w", id, table, err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking rows affected for id %v in table %s: %w", id, table, err)
	}
	if rows == 0 {
		return fmt.Errorf("no rows deleted for id %v in table %s", id, table)
	}

	return nil
}

func sqlExportHelper[S any](
	ctx context.Context,
	items *data.IterablePaginatedData[S],
	tableName string,
	headerNames []string,
	write func(wrtier *csv.Writer, storeItem S) error,
) error {
	// Ensure exports directory exists
	var OwnerReadWriteExecuteAndOthersReadExecute = 0755
	err := os.MkdirAll(db.EXPORT_DIR, os.FileMode(OwnerReadWriteExecuteAndOthersReadExecute))
	if err != nil {
		return fmt.Errorf("error creating export directory: %w", err)
	}

	// Generate filename
	now := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("%s/%s_%v.csv", db.EXPORT_DIR, tableName, now)

	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating export file: %w", err)
	}
	defer logs.LogErrorsWithContext(ctx, f.Close, fmt.Sprintf("error closing file %v", filename))

	writer := csv.NewWriter(f)
	defer writer.Flush()

	// Write CSV header
	err = writer.Write(headerNames)
	if err != nil {
		return fmt.Errorf("error writing CSV header: %w", err)
	}

	// Write each row
	for {
		item, err := items.Next(ctx)
		if err != nil {
			logs.ErrorWithContext(ctx, "Error while fetching item from %v while exporting: %v", tableName, err)
		}
		if item == nil {
			break
		}

		err = write(writer, *item)
		if err != nil {
			logs.ErrorWithContext(ctx, "Error while writing csv row with data %v: %v", item, err)
		}
	}

	return nil
}