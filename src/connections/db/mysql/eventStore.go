package mysql

import (
	"com/connections/db"
	"com/data"
	"database/sql"
)

var _ db.TimestampedDataStore[data.Event, data.StoreEvent, data.EventFilter] = (*MySQLEventStore)(nil)

type MySQLEventStore struct {
	MySQLTimestampedDataStore[data.Event, data.StoreEvent, data.EventFilter]
}

func NewMySQLEventStore(db *sql.DB) MySQLEventStore {
	return MySQLEventStore{
		MySQLTimestampedDataStore: MySQLTimestampedDataStore[data.Event, data.StoreEvent, data.EventFilter]{
			timestampKey: "event_timestamp",
			MySQLStore: MySQLStore[data.Event, data.StoreEvent, data.EventFilter]{
				db:        db,
				tableName: "events",
				tableCreationSQL: `		
					CREATE TABLE IF NOT EXISTS events (
						event_id VARCHAR(36) NOT NULL,

						request_device_id 		VARCHAR(36) NOT NULL, -- //TODO: what do these two columns mean
						event_source_device_id 	VARCHAR(36) NOT NULL, -- //TODO: what do these two columns mean
						response_timestamp 		BIGINT      NOT NULL,

						event_timestamp BIGINT      NOT NULL,
						field_name 		VARCHAR(45) NOT NULL,
						field_value 	VARCHAR(45) NOT NULL,

						PRIMARY KEY (event_id),

						INDEX event_source_device_id_idx (event_source_device_id ASC),
						CONSTRAINT event_source_device_id
							FOREIGN KEY (event_source_device_id)
							REFERENCES devices (device_id)
							ON DELETE NO ACTION
							ON UPDATE NO ACTION
							
					) ENGINE = InnoDB;
				`,
				tableColumns: []string{
					"event_id",
					"request_device_id",
					"event_source_device_id",
					"response_timestamp",
					"event_timestamp",
					"field_name",
					"field_value",
				},
				primaryKey: "event_id",
			},
		},
	}
}
