package mysql

import (
	"com/connections/db"
	"com/data"
	"database/sql"
)

var _ db.TimestampedDataStore[data.Log, data.StoreLog, data.LogFilter] = (*MySQLLogStore)(nil)
type MySQLLogStore struct {
	MySQLTimestampedDataStore[data.Log, data.StoreLog, data.LogFilter]
}
func NewMySQLLogStore(db *sql.DB) MySQLLogStore {
	return MySQLLogStore{
		MySQLTimestampedDataStore: MySQLTimestampedDataStore[data.Log, data.StoreLog, data.LogFilter]{
			timestampKey: "log_timestamp",
			MySQLStore: MySQLStore[data.Log, data.StoreLog, data.LogFilter]{
				db: db,
				tableName: "logs",
				tableCreationSQL: `		
					CREATE TABLE IF NOT EXISTS logs (
						log_id 			VARCHAR(36) NOT NULL,
						job_id 			VARCHAR(36) NOT NULL,
						log_level 		INT			NOT NULL,
						log_stack_trace TEXT 		NOT NULL,
						log_description TEXT		NOT NULL,
						log_timestamp   BIGINT		NOT NULL
					) ENGINE = InnoDB;
				`,
				tableColumns: []string{
					"log_id",
					"job_id",
					"log_level",
					"log_stack_trace",
					"log_description",
					"log_timestamp",
				},
				primaryKey: "log_id",
			},
		},
	}
}
