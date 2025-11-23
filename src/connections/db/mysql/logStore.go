package mysql

import (
	"com/connections/db"
	"com/data"
	"com/utils"
	"database/sql"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/samborkent/uuidv7"
)

var _ db.LogStore = (*MySQLLogStore)(nil)

type MySQLLogStore struct {
	DB *sql.DB
}

func (store *MySQLLogStore) Add(item data.Log) error {
	context, cancel := utils.TimeoutContext(requestTimeout)
	defer cancel()
	_, err := store.DB.ExecContext(context,
		`
			INSERT INTO logs (
				log_id,
				job_id,
				log_level,
				log_stack_trace,
				log_description,
				log_timestamp
			) VALUES (?, ?, ?, ?, ?, ?)
		`,
		uuidv7.New().String(), // MySQL does not support uuidv7 and is notably slower
		item.JobID,
		item.Level,
		item.StackTrace,
		item.Description,
		item.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("error while adding %v to log store: %w", item, err)
	}
	return nil
}
func (store *MySQLLogStore) Delete(storeItem data.StoreLog) error {
	context, cancel := utils.TimeoutContext(requestTimeout)
	defer cancel()
	response, err := store.DB.ExecContext(
		context,
		`DELETE FROM logs WHERE (log_id = ?)`,
		storeItem.ID,
	)
	if err != nil {
		return fmt.Errorf("error while deleting %v from log store: %w", storeItem, err)
	}
	rows, err := response.RowsAffected()
	if err != nil {
		return fmt.Errorf("error while checking rows affected for deleting of %v from log store. query response: %v, error: %w", storeItem, response, err)
	}
	if rows == 0 {
		return fmt.Errorf("no rows deleted while deleting %v from log store", storeItem)
	}
	return nil
}
func (store *MySQLLogStore) Get(filter data.LogFilter) (*data.IterablePaginatedData[data.StoreLog], error) {
	return store.GetInTimeRange(filter, nil, nil)
}
func (store *MySQLLogStore) GetInTimeRange(filter data.LogFilter, startTime *int64, endTime *int64) (*data.IterablePaginatedData[data.StoreLog], error) {
	// Build conditions
	args := []any{}
	conditions := []string{}
	if filter.ID != nil {
		conditions = append(conditions, "log_id = ?")
		args = append(args, *filter.ID)
	}
	if filter.JobID != nil {
		conditions = append(conditions, "job_id = ?")
		args = append(args, *filter.JobID)
	}
	if filter.Level != nil {
		conditions = append(conditions, "log_level = ?")
		args = append(args, *filter.Level)
	}
	if filter.StackTrace != nil {
		conditions = append(conditions, "log_stack_trace = ?")
		args = append(args, *filter.StackTrace)
	}
	if filter.Description != nil {
		conditions = append(conditions, "log_description = ?")
		args = append(args, *filter.Description)
	}
	if filter.Timestamp != nil {
		conditions = append(conditions, "log_timestamp = ?")
		args = append(args, *filter.Timestamp)
	}
	if startTime != nil {
		conditions = append(conditions, "log_timestamp > ?")
		args = append(args, *startTime)
	}
	if endTime != nil {
		conditions = append(conditions, "log_timestamp < ?")
		args = append(args, *endTime)
	}
	query := "SELECT * FROM logs"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Add pagination condition
	if len(conditions) > 0 {
		query += " AND "
	} else {
		query += " WHERE "
	}
	query += "log_id > ? ORDER BY log_id LIMIT ?"

	getPage := func(lastID *string) ([]data.StoreLog, *string, error) {
		var filterID string
		if lastID != nil {
			filterID = *lastID
		}
		context, cancel := utils.TimeoutContext(requestTimeout)
		defer cancel()
		rows, err := store.DB.QueryContext(context, query, append(args, filterID, data.PAGE_SIZE)...)
		if err != nil {
			return nil, nil, fmt.Errorf("error querying logs with filter %v: %w", filter, err)
		}
		defer utils.LogErrors(rows.Close, fmt.Sprintf("error closing rows for query %v and lastID %v", query, lastID))

		var logs []data.StoreLog
		for rows.Next() {
			var log data.StoreLog
			err := rows.Scan(
				&log.ID,
				&log.JobID,
				&log.Level,
				&log.StackTrace,
				&log.Description,
				&log.Timestamp,
			)
			if err != nil {
				return nil, nil, fmt.Errorf("error scanning log: %w", err)
			}
			logs = append(logs, log)
		}
		err = rows.Err()
		if err != nil {
			return nil, nil, fmt.Errorf("error in rows: %w", err)
		}
		if len(logs) == 0 {
			return []data.StoreLog{}, nil, nil
		}
		lastLog := logs[len(logs)-1]
		return logs, &lastLog.ID, nil
	}

	return &data.IterablePaginatedData[data.StoreLog]{GetPage: getPage}, nil
}
func (store *MySQLLogStore) Setup(isDestructive bool) error {
	if isDestructive {
		context, cancel := utils.TimeoutContext(requestTimeout)
		defer cancel()
		_, err := store.DB.ExecContext(context, `SET FOREIGN_KEY_CHECKS = 0`)
		if err != nil {
			return fmt.Errorf("error disabling FK checks: %w", err)
		}
		context, cancel = utils.TimeoutContext(requestTimeout)
		defer cancel()
		_, err = store.DB.ExecContext(context, `DROP TABLE IF EXISTS logs`)
		if err != nil {
			return fmt.Errorf("error dropping logs table: %w", err)
		}
		context, cancel = utils.TimeoutContext(requestTimeout)
		defer cancel()
		_, err = store.DB.ExecContext(context, `SET FOREIGN_KEY_CHECKS = 1`)
		if err != nil {
			return fmt.Errorf("error enabling FK checks: %w", err)
		}
	}
	context, cancel := utils.TimeoutContext(requestTimeout)
	defer cancel()
	_, err := store.DB.ExecContext(context, `		
		CREATE TABLE IF NOT EXISTS logs (
			log_id 			VARCHAR(36) NOT NULL,
			job_id 			VARCHAR(36) NOT NULL,
			log_level 		INT			NOT NULL,
			log_stack_trace TEXT 		NOT NULL,
			log_description TINYTEXT	NOT NULL
		) ENGINE = InnoDB;
	`)
	if err != nil {
		return fmt.Errorf("error creating log table: %w", err)
	}
	return nil
}
func (store *MySQLLogStore) Export(filter data.LogFilter) error {
	return store.ExportInTimeRange(filter, nil, nil)
}
func (store *MySQLLogStore) ExportInTimeRange(filter data.LogFilter, startTime *int64, endTime *int64) error {
	// Ensure exports directory exists
	var OwnerReadWriteExecuteAndOthersReadExecute = 0755
	err := os.MkdirAll(db.EXPORT_DIR, os.FileMode(OwnerReadWriteExecuteAndOthersReadExecute))
	if err != nil {
		return fmt.Errorf("error creating export directory: %w", err)
	}

	// Generate filename
	now := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("%s/%s_logs.csv", db.EXPORT_DIR, now)

	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating export file: %w", err)
	}
	defer utils.LogErrors(f.Close, fmt.Sprintf("error closing file %v", filename))

	w := csv.NewWriter(f)
	defer w.Flush()

	// Write CSV header
	err = w.Write([]string{
		"log_id",
		"job_id",
		"log_level",
		"log_stack_trace",
		"log_description",
		"log_timestamp",
	})
	if err != nil {
		return fmt.Errorf("error writing CSV header: %w", err)
	}

	// Get data
	logs, err := store.GetInTimeRange(filter, startTime, endTime)
	if err != nil {
		return fmt.Errorf("error getting logs for export with filter %v: %w", filter, err)
	}

	// Write each row
	for {
		log, err := logs.Next()
		if err != nil {
			utils.DefaultSafeLog(fmt.Sprintf("Error while fetching log while exporting: %v", err))
		}
		if log == nil {
			break
		}

		err = w.Write([]string{
			log.ID,
			log.JobID,
			fmt.Sprint(log.Level),
			log.StackTrace,
			log.Description,
			strconv.FormatInt(log.Timestamp, 10),
		})
		if err != nil {
			utils.DefaultSafeLog(fmt.Sprintf("Error while writing csv row with data %v: %v", log, err))
		}
	}

	return nil
}
