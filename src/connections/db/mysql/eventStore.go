package mysql

import (
	"com/connections/db"
	"com/data"
	"database/sql"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

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
func (store *MySQLEventStore) Get(filter data.EventFilter) (*data.IterablePaginatedData[data.StoreEvent], error) {
	// Build conditions
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

	// Add pagination condition
	if len(conditions) > 0 {
		query += " AND "
	} else {
		query += " WHERE "
	}
	query += "event_id > ? ORDER BY event_id LIMIT ?"

	getPage := func(lastID *string) ([]data.StoreEvent, *string, error) {

		rows, err := store.DB.Query(query, append(args, lastID, data.PAGE_SIZE))
		if err != nil {
			return nil, nil, fmt.Errorf("error querying events with filter %v: %w", filter, err)
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
				return nil, nil, fmt.Errorf("error scanning event: %w", err)
			}
			events = append(events, event)
		}
		if len(events) == 0 {
			return []data.StoreEvent{}, nil, nil
		}
		lastEvent := events[len(events)-1]
		return events, &lastEvent.ID, nil
	}

	return &data.IterablePaginatedData[data.StoreEvent]{GetPage: getPage}, nil
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
				REFERENCES devices (device_id)
				ON DELETE NO ACTION
				ON UPDATE NO ACTION
				
		) ENGINE = InnoDB;
	`)
	if err != nil {
		return fmt.Errorf("error creating event table: %w", err)
	}
	return nil
}
func (store *MySQLEventStore) Export(filter data.EventFilter) error {

	// Ensure exports directory exists
	if err := os.MkdirAll(db.EXPORT_DIR, 0755); err != nil {
		return fmt.Errorf("error creating export directory: %w", err)
	}

	// Generate filename
	now := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("%s/%s_events.csv", db.EXPORT_DIR, now)

	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating export file: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Write CSV header
	if err := w.Write([]string{
		"event_id",
		"request_device_id",
		"event_source_device_id",
		"response_timestamp",
		"event_timestamp",
		"field_name",
		"field_value",
	}); err != nil {
		return fmt.Errorf("error writing CSV header: %w", err)
	}

	// Get data
	events, err := store.Get(filter)
	if err != nil {
		return fmt.Errorf("error getting devices for export with filter %v: %w", filter, err)
	}

	// Write each row
	for {
		event, err := events.Next()
		if err != nil {
			log.Default().Output(1, fmt.Sprintf("Error while fetching device while exporting: %v", err))
		}
		if event == nil {
			break
		}

		err = w.Write([]string{
			fmt.Sprint(event.ID),
			event.RequestDeviceID,
			event.EventSourceDeviceID,
			fmt.Sprint(event.ResponseTimestamp),
			fmt.Sprint(event.EventTimestamp),
			event.FieldName,
			event.FieldValue,
		})
		if err != nil {
			log.Default().Output(1, fmt.Sprintf("Error while writing csv row with data %v: %v", event, err))
		}
	}

	return nil
}
