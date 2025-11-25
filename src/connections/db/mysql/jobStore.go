package mysql

import (
	"com/connections/db"
	"com/data"
	"database/sql"
)

var _ db.ClosableStore[data.Job, data.StoreJob, data.JobFilter] = (*MySQLJobStore)(nil)
type MySQLJobStore struct {
	MySQLClosableStore[data.Job, data.StoreJob, data.JobFilter]
}

func NewMySQLJobStore(db *sql.DB) MySQLJobStore {
	return MySQLJobStore{
		MySQLClosableStore: MySQLClosableStore[data.Job, data.StoreJob, data.JobFilter]{
			closeKey: "job_end_timestamp",
			MySQLTimestampedDataStore: MySQLTimestampedDataStore[data.Job, data.StoreJob, data.JobFilter]{
				timestampKey: "job_start_timestamp",
				MySQLStore: MySQLStore[data.Job, data.StoreJob, data.JobFilter]{
					db: db,
					tableName: "jobs",
					tableCreationSQL: `		
					CREATE TABLE IF NOT EXISTS jobs (
						job_id 				VARCHAR(36) NOT NULL,
						parent_job_id		VARCHAR(36) NOT NULL,
						job_category 		VARCHAR(36) NOT NULL,
						job_start_timestamp BIGINT		NOT NULL,
						job_end_timestamp 	BIGINT		NOT NULL
					) ENGINE = InnoDB;
					`,
					tableColumns: []string{
						"job_id",
						"parent_job_id",
						"job_category",
						"job_start_timestamp",
						"job_end_timestamp",
					},
					primaryKey: "job_id",
				},
			},
		},
	}
}