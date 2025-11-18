package mysql

import (
	"com/data"
	"com/connections/db"
	"database/sql"
	"fmt"
	"strings"

	"github.com/samborkent/uuidv7"
)

var _ db.EventStore = (*MySQLEventStore)(nil)

type MySQLEventStore struct {
	DB *sql.DB
}

func (store *MySQLEventStore) Add(item data.Event) error {
	_, err := store.DB.Exec(
		`
			INSERT INTO events (
				event_id,
				request_device_id,
				event_source_device_id,
				response_timestamp,
				event_timestamp,
				field_name,
				field_value
			) VALUES (?, ?, ?, ?, ?, ?, ?)
		`,
		uuidv7.New().String(), // MySQL does not support uuidv7 and is notably slower
		item.RequestDeviceID,
		item.EventSourceDeviceID,
		item.ResponseTimestamp,
		item.EventTimestamp,
		item.FieldName,
		item.FieldValue,
	)
	if err != nil {
		return fmt.Errorf("error while adding %v to event store: %w", item, err)
	}
	return nil
}
func (store *MySQLEventStore) Delete(storeItem data.StoreEvent) error {
	response, err := store.DB.Exec(
		`DELETE FROM events WHERE (event_id = ?)`,
		storeItem.ID,
	)
	if err != nil {
		return fmt.Errorf("error while deleting %v from event store: %w", storeItem, err)
	}
	rows, err := response.RowsAffected()
	if err != nil {
		return fmt.Errorf("error while checking rows affected for deleting of %v from event store. query response: %v, error: %w", storeItem, response, err)
	}
	if rows == 0 {
		return fmt.Errorf("no rows deleted while deleting %v from event store", storeItem)
	}
	return nil
}
func (store *MySQLEventStore) Get(filter data.EventFilter) ([]data.StoreEvent, error) {
	args := []any{}
	conditions := []string{}
	if filter.ID != nil {
		conditions = append(conditions, "event_id = ?")
		args = append(args, *filter.ID)
	}
	if filter.RequestDeviceID != nil {
		conditions = append(conditions, "request_device_id = ?")
		args = append(args, *filter.RequestDeviceID)
	}
	if filter.EventSourceDeviceID != nil {
		conditions = append(conditions, "event_source_device_id = ?")
		args = append(args, *filter.EventSourceDeviceID)
	}
	if filter.ResponseTimestamp != nil {
		conditions = append(conditions, "response_timestamp = ?")
		args = append(args, *filter.ResponseTimestamp)
	}
	if filter.EventTimestamp != nil {
		conditions = append(conditions, "event_timestamp = ?")
		args = append(args, *filter.EventSourceDeviceID)
	}
	if filter.FieldName != nil {
		conditions = append(conditions, "field_name = ?")
		args = append(args, *filter.EventSourceDeviceID)
	}
	if filter.FieldValue != nil {
		conditions = append(conditions, "field_value = ?")
		args = append(args, *filter.EventSourceDeviceID)
	}

	query := "SELECT * FROM events"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	rows, err := store.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("error querying events with filter %v: %w", filter, err)
	}
	defer rows.Close() // ignore error

	var events []data.StoreEvent
	for rows.Next() {
		var event data.StoreEvent
		err := rows.Scan(
			&event.ID,
			&event.RequestDeviceID,
			&event.EventSourceDeviceID,
			&event.ResponseTimestamp,
			&event.EventTimestamp,
			&event.FieldName,
			&event.FieldValue,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning event: %w", err)
		}
		events = append(events, event)
	}

	return events, nil
}
func (store *MySQLEventStore) Setup(isDestructive bool) error {
	if isDestructive {
		if _, err := store.DB.Exec(`SET FOREIGN_KEY_CHECKS = 0`); err != nil {
			return fmt.Errorf("error disabling FK checks: %w", err)
		}
		if _, err := store.DB.Exec(`DROP TABLE IF EXISTS events`); err != nil {
			return fmt.Errorf("error dropping events table: %w", err)
		}
		if _, err := store.DB.Exec(`SET FOREIGN_KEY_CHECKS = 1`); err != nil {
			return fmt.Errorf("error enabling FK checks: %w", err)
		}
	}
	_, err := store.DB.Exec(`		
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
				REFERENCES devices (internal_device_id)
				ON DELETE NO ACTION
				ON UPDATE NO ACTION
				
		) ENGINE = InnoDB;
	`)
	if err != nil {
		return fmt.Errorf("error creating event table: %w", err)
	}
	return nil
}
