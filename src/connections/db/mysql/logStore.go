package mysql

import (
	"com/connections/db"
	"com/data"
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"strconv"
)

var _ db.LogStore = (*MySQLLogStore)(nil)

type MySQLLogStore struct {
	DB *sql.DB
}

func (store *MySQLLogStore) Add(ctx context.Context, item data.Log) (string, error) {
	return sqlInsertHelper[data.Event](
		ctx,
		store.DB,
		"logs",
		[]string{"log_id", "job_id", "log_level", "log_stack_trace", "log_description", "log_timestamp"},
		[]any{item.JobID, item.Level, item.StackTrace, item.Description, item.Timestamp},
	)
}
func (store *MySQLLogStore) Delete(ctx context.Context, storeItem data.StoreLog) error {
	return sqlDeleteHelper[data.StoreJob](ctx, store.DB, "logs", storeItem.ID)
}
func (store *MySQLLogStore) Get(ctx context.Context, filter data.LogFilter) *data.IterablePaginatedData[data.StoreLog] {
	return store.GetInTimeRange(ctx, filter, nil, nil)
}
func (store *MySQLLogStore) GetInTimeRange(ctx context.Context, filter data.LogFilter, startTime *int64, endTime *int64) *data.IterablePaginatedData[data.StoreLog] {
	return sqlGetHelper(
		store.DB,
		[]string{"log_id = ?", "job_id = ?", "log_level = ?", "log_stack_trace = ?", "log_description = ?", "log_timestamp = ?", "log_timestamp > ?", "log_timestamp < ?"},
		[]any{filter.ID, filter.JobID, filter.Level, filter.StackTrace, filter.Description, filter.Timestamp, startTime, endTime}, 
		"logs",
		"log_id",
		func(rows *sql.Rows) (*data.StoreLog, error) {
			var log data.StoreLog
			err := rows.Scan(&log.ID, &log.JobID, &log.Level, &log.StackTrace, &log.Description, &log.Timestamp)
			if err != nil {
				return nil, fmt.Errorf("error scanning log: %w", err)
			}
			return &log, nil
		},
	)	
}
func (store *MySQLLogStore) Setup(ctx context.Context, isDestructive bool) error {
	if isDestructive {
		err := dropTable(ctx, store.DB, "logs")
		if err != nil {
			return err
		}
	}
	return sqlCreateTableHelper(ctx, store.DB, `		
		CREATE TABLE IF NOT EXISTS logs (
			log_id 			VARCHAR(36) NOT NULL,
			job_id 			VARCHAR(36) NOT NULL,
			log_level 		INT			NOT NULL,
			log_stack_trace TEXT 		NOT NULL,
			log_description TEXT		NOT NULL,
			log_timestamp   BIGINT		NOT NULL
		) ENGINE = InnoDB;
	`, "logs")
}
func (store *MySQLLogStore) Export(ctx context.Context, filter data.LogFilter) error {
	return store.ExportInTimeRange(ctx, filter, nil, nil)
}
func (store *MySQLLogStore) ExportInTimeRange(ctx context.Context, filter data.LogFilter, startTime *int64, endTime *int64) error {
	// Get data
	logs := store.GetInTimeRange(ctx, filter, startTime, endTime)

	// Export data
	return sqlExportHelper(
		ctx,
		logs,
		"logs",
		[]string{"log_id", "job_id", "log_level", "log_stack_trace", "log_description", "log_timestamp"},
		func(writer *csv.Writer, log data.StoreLog) error {
			return writer.Write([]string{
				log.ID,
				log.JobID,
				strconv.Itoa(log.Level),
				log.StackTrace,
				log.Description,
				strconv.FormatInt(log.Timestamp, 10),
			})
		},
	)
}
