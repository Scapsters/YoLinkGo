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
)

var _ db.JobStore = (*MySQLJobStore)(nil)

type MySQLJobStore struct {
	DB *sql.DB
}

func (store *MySQLJobStore) Add(ctx context.Context, item data.Job) (string, error) {
	return sqlInsertHelper[data.Event](
		ctx,
		store.DB,
		"jobs",
		[]string{"job_id", "parent_job_id", "job_category", "job_start_timestamp", "job_end_timestamp"},
		[]any{item.ParentID, item.Category, item.StartTimestamp, item.EndTimestamp},
	)
}
func (store *MySQLJobStore) Delete(ctx context.Context, storeItem data.StoreJob) error {
	return sqlDeleteHelper[data.StoreJob](ctx, store.DB, "jobs", storeItem.ID)
}
func (store *MySQLJobStore) End(ctx context.Context, job data.StoreJob) error {
	sqlctx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()
	res, err := store.DB.ExecContext(
		sqlctx, 
		`UPDATE devices SET job_end_timestamp = ? WHERE job_id = ?`,
		utils.TimeSeconds(),
		job.ID,
	)
	if err != nil {
		return fmt.Errorf("error ending job %v: %w", job, err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking rows affected in edit for job %v: %w", job, err)
	}
	if rows == 0 {
		logs.WarnWithContext(ctx, "job %v was not found when attempting to close it", job)
		return fmt.Errorf("no job ended with ID %v", job.ID)
	}

	return nil
}
func (store *MySQLJobStore) Get(ctx context.Context, filter data.JobFilter) *data.IterablePaginatedData[data.StoreJob] {
	return store.GetInTimeRange(ctx, filter, nil, nil)
}
func (store *MySQLJobStore) GetInTimeRange(ctx context.Context, filter data.JobFilter, startTime *int64, endTime *int64) *data.IterablePaginatedData[data.StoreJob] {
	return sqlGetHelper(
		store.DB,
		[]string{"job_id = ?", "parent_job_id = ?", "job_category = ?", "job_start_timestamp = ?", "job_end_timestamp = ?", "job_start_timestamp > ?", "job_start_timestamp < ?"},
		[]any{filter.ID, filter.ParentID, filter.Category, filter.StartTimestamp, filter.EndTimestamp, startTime, endTime}, 
		"jobs",
		"job_id",
		func(rows *sql.Rows) (*data.StoreJob, error) {
			var job data.StoreJob
			err := rows.Scan(
				&job.ID, &job.ParentID, &job.Category, &job.StartTimestamp, &job.EndTimestamp,
			)
			if err != nil {
				return nil, fmt.Errorf("error scanning log: %w", err)
			}
			return &job, nil
		},
	)
}
func (store *MySQLJobStore) Setup(ctx context.Context, isDestructive bool) error {
	if isDestructive {
		err := dropTable(ctx, store.DB, "jobs")
		if err != nil {
			return err
		}
	}
	return sqlCreateTableHelper(ctx, store.DB, `		
		CREATE TABLE IF NOT EXISTS jobs (
			job_id 				VARCHAR(36) NOT NULL,
			parent_job_id		VARCHAR(36) NOT NULL,
			job_category 		VARCHAR(36) NOT NULL,
			job_start_timestamp BIGINT		NOT NULL,
			job_end_timestamp 	BIGINT		NOT NULL
		) ENGINE = InnoDB;
	`, "jobs")
}
func (store *MySQLJobStore) Export(ctx context.Context, filter data.JobFilter) error {
	return store.ExportInTimeRange(ctx, filter, nil, nil)
}
func (store *MySQLJobStore) ExportInTimeRange(ctx context.Context, filter data.JobFilter, startTime *int64, endTime *int64) error {
	// Get data
	jobs := store.GetInTimeRange(ctx, filter, startTime, endTime)

	return sqlExportHelper(
		ctx,
		jobs,
		"jobs",
		[]string{"job_id", "parent_job_id", "job_category", "job_start_timestamp", "job_end_timestamp"},
		func(writer *csv.Writer, job data.StoreJob) error {
			return writer.Write([]string{
				job.ID,
				job.ParentID,
				job.Category,
				utils.EpochSecondsToExcelDate(job.StartTimestamp),
				utils.EpochSecondsToExcelDate(job.EndTimestamp),
			})
		},
	)
}
