package mysql

import (
	"com/connections/db"
	"com/data"
	"com/utils"
	"database/sql"
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/samborkent/uuidv7"
)

var _ db.JobStore = (*MySQLJobStore)(nil)

type MySQLJobStore struct {
	DB *sql.DB
}

func (store *MySQLJobStore) Add(item data.Job) error {
	context, cancel := utils.TimeoutContext(requestTimeout)
	defer cancel()
	_, err := store.DB.ExecContext(context,
		`
			INSERT INTO jobs (
				job_id,
				job_category,
				job_start_timestamp,
				job_end_timestamp
			) VALUES (?, ?, ?, ?)
		`,
		uuidv7.New().String(), // MySQL does not support uuidv7 and is notably slower
		item.Category,
		item.StartTimestamp,
		item.EndTimestamp,
	)
	if err != nil {
		return fmt.Errorf("error while adding %v to job store: %w", item, err)
	}
	return nil
}
func (store *MySQLJobStore) Delete(storeItem data.StoreJob) error {
	context, cancel := utils.TimeoutContext(requestTimeout)
	defer cancel()
	response, err := store.DB.ExecContext(
		context,
		`DELETE FROM jobs WHERE (job_id = ?)`,
		storeItem.ID,
	)
	if err != nil {
		return fmt.Errorf("error while deleting %v from job store: %w", storeItem, err)
	}
	rows, err := response.RowsAffected()
	if err != nil {
		return fmt.Errorf("error while checking rows affected for deleting of %v from job store. query response: %v, error: %w", storeItem, response, err)
	}
	if rows == 0 {
		return fmt.Errorf("no rows deleted while deleting %v from job store", storeItem)
	}
	return nil
}
func (store *MySQLJobStore) Get(filter data.JobFilter) (*data.IterablePaginatedData[data.StoreJob], error) {
	return store.GetInTimeRange(filter, nil, nil)
}
func (store *MySQLJobStore) GetInTimeRange(filter data.JobFilter, startTime *int64, endTime *int64) (*data.IterablePaginatedData[data.StoreJob], error) {
	// Build conditions
	args := []any{}
	conditions := []string{}
	if filter.ID != nil {
		conditions = append(conditions, "job_id = ?")
		args = append(args, *filter.ID)
	}
	if filter.Category != nil {
		conditions = append(conditions, "job_category = ?")
		args = append(args, *filter.Category)
	}
	if filter.StartTimestamp != nil {
		conditions = append(conditions, "job_start_timestamp = ?")
		args = append(args, *filter.StartTimestamp)
	}
	if filter.EndTimestamp != nil {
		conditions = append(conditions, "job_end_timestamp = ?")
		args = append(args, *filter.EndTimestamp)
	}
	if startTime != nil {
		conditions = append(conditions, "job_start_timestamp > ?")
		args = append(args, *startTime)
	}
	if endTime != nil {
		conditions = append(conditions, "job_start_timestamp < ?")
		args = append(args, *endTime)
	}
	query := "SELECT * FROM jobs"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Add pagination condition
	if len(conditions) > 0 {
		query += " AND "
	} else {
		query += " WHERE "
	}
	query += "job_id > ? ORDER BY job_id LIMIT ?"

	getPage := func(lastID *string) ([]data.StoreJob, *string, error) {
		var filterID string
		if lastID != nil {
			filterID = *lastID
		}
		context, cancel := utils.TimeoutContext(requestTimeout)
		defer cancel()
		rows, err := store.DB.QueryContext(context, query, append(args, filterID, data.PAGE_SIZE)...)
		if err != nil {
			return nil, nil, fmt.Errorf("error querying jobs with filter %v: %w", filter, err)
		}
		defer utils.LogErrors(rows.Close, fmt.Sprintf("error closing rows for query %v and lastID %v", query, lastID))

		var jobs []data.StoreJob
		for rows.Next() {
			var job data.StoreJob
			err := rows.Scan(
				&job.ID,
				&job.Category,
				&job.StartTimestamp,
				&job.EndTimestamp,
			)
			if err != nil {
				return nil, nil, fmt.Errorf("error scanning job: %w", err)
			}
			jobs = append(jobs, job)
		}
		err = rows.Err()
		if err != nil {
			return nil, nil, fmt.Errorf("error in rows: %w", err)
		}
		if len(jobs) == 0 {
			return []data.StoreJob{}, nil, nil
		}
		lastJob := jobs[len(jobs)-1]
		return jobs, &lastJob.ID, nil
	}

	return &data.IterablePaginatedData[data.StoreJob]{GetPage: getPage}, nil
}
func (store *MySQLJobStore) Setup(isDestructive bool) error {
	if isDestructive {
		context, cancel := utils.TimeoutContext(requestTimeout)
		defer cancel()
		_, err := store.DB.ExecContext(context, `SET FOREIGN_KEY_CHECKS = 0`)
		if err != nil {
			return fmt.Errorf("error disabling FK checks: %w", err)
		}
		context, cancel = utils.TimeoutContext(requestTimeout)
		defer cancel()
		_, err = store.DB.ExecContext(context, `DROP TABLE IF EXISTS jobs`)
		if err != nil {
			return fmt.Errorf("error dropping jobs table: %w", err)
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
		CREATE TABLE IF NOT EXISTS jobs (
			job_id 				VARCHAR(36) NOT NULL,
			job_category 		VARCHAR(36) NOT NULL,
			job_start_timestamp BIGINT		NOT NULL,
			job_end_timestamp 	BIGINT		NOT NULL
		) ENGINE = InnoDB;
	`)
	if err != nil {
		return fmt.Errorf("error creating job table: %w", err)
	}
	return nil
}
func (store *MySQLJobStore) Export(filter data.JobFilter) error {
	return store.ExportInTimeRange(filter, nil, nil)
}
func (store *MySQLJobStore) ExportInTimeRange(filter data.JobFilter, startTime *int64, endTime *int64) error {
	// Ensure exports directory exists
	var OwnerReadWriteExecuteAndOthersReadExecute = 0755
	err := os.MkdirAll(db.EXPORT_DIR, os.FileMode(OwnerReadWriteExecuteAndOthersReadExecute))
	if err != nil {
		return fmt.Errorf("error creating export directory: %w", err)
	}

	// Generate filename
	now := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("%s/%s_jobs.csv", db.EXPORT_DIR, now)

	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating export file: %w", err)
	}
	defer utils.LogErrors(f.Close, fmt.Sprintf("error closing file %v", filename))

	w := csv.NewWriter(f)
	defer w.Flush()

	// Write CSV header
	err = w.Write([]string{
		"job_id",
		"job_category",
		"job_start_timestamp",
		"job_end_timestamp",
	})
	if err != nil {
		return fmt.Errorf("error writing CSV header: %w", err)
	}

	// Get data
	jobs, err := store.GetInTimeRange(filter, startTime, endTime)
	if err != nil {
		return fmt.Errorf("error getting jobs for export with filter %v: %w", filter, err)
	}

	// Write each row
	for {
		job, err := jobs.Next()
		if err != nil {
			utils.DefaultSafeLog(fmt.Sprintf("Error while fetching job while exporting: %v", err))
		}
		if job == nil {
			break
		}

		err = w.Write([]string{
			job.ID,
			job.Category,
			utils.EpochSecondsToExcelDate(job.StartTimestamp),
			utils.EpochSecondsToExcelDate(job.EndTimestamp),
		})
		if err != nil {
			utils.DefaultSafeLog(fmt.Sprintf("Error while writing csv row with data %v: %v", job, err))
		}
	}

	return nil
}
