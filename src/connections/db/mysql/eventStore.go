package mysql

import (
	"com/connections/db"
	"com/data"
	"com/utils"
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
)

var _ db.EventStore = (*MySQLEventStore)(nil)

type MySQLEventStore struct {
	DB *sql.DB
}

func (store *MySQLEventStore) Add(ctx context.Context, item data.Event) (string, error) {
	return sqlInsertHelper[data.Event](
		ctx,
		store.DB,
		"events",
		[]string{"event_id", "request_device_id", "event_source_device_id", "response_timestamp", "event_timestamp", "field_name", "field_value"},
		[]any{item.RequestDeviceID, item.EventSourceDeviceID, item.ResponseTimestamp, item.EventTimestamp, item.FieldName, item.FieldValue},
	)
}
func (store *MySQLEventStore) Delete(ctx context.Context, storeItem data.StoreEvent) error {
	return sqlDeleteHelper[data.StoreJob](ctx, store.DB, "events", storeItem.ID)
}
func (store *MySQLEventStore) Get(ctx context.Context, filter data.EventFilter) *data.IterablePaginatedData[data.StoreEvent] {
	return store.GetInTimeRange(ctx, filter, nil, nil)
}
func (store *MySQLEventStore) GetInTimeRange(ctx context.Context, filter data.EventFilter, startTime *int64, endTime *int64) *data.IterablePaginatedData[data.StoreEvent] {
	return sqlGetHelper(
		store.DB,
		[]string{"event_id = ?", "request_device_id = ?", "event_source_device_id = ?", "response_timestamp = ?", "event_timestamp = ?", "field_name = ?", "field_value = ?", "event_timestamp > ?", "event_timestamp < ?"},
		[]any{filter.ID, filter.RequestDeviceID, filter.EventSourceDeviceID, filter.ResponseTimestamp, filter.EventTimestamp, filter.FieldName, filter.FieldValue, startTime, endTime},
		"events",
		"event_id",
		func(rows *sql.Rows) (*data.StoreEvent, error) {
			var event data.StoreEvent
			err := rows.Scan(&event.ID, &event.RequestDeviceID, &event.EventSourceDeviceID, &event.ResponseTimestamp, &event.EventTimestamp, &event.FieldName, &event.FieldValue)
			if err != nil {
				return nil, fmt.Errorf("error scanning event: %w", err)
			}
			return &event, nil
		},
	)
}
func (store *MySQLEventStore) Setup(ctx context.Context, isDestructive bool) error {
	if isDestructive {
		err := dropTable(ctx, store.DB, "events")
		if err != nil {
			return err
		}
	}
	return sqlCreateTableHelper(ctx, store.DB, `		
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
	`, "events")
}
func (store *MySQLEventStore) Export(ctx context.Context, filter data.EventFilter) error {
	return store.ExportInTimeRange(ctx, filter, nil, nil)
}
func (store *MySQLEventStore) ExportInTimeRange(ctx context.Context, filter data.EventFilter, startTime *int64, endTime *int64) error {
	// Get data
	events := store.GetInTimeRange(ctx, filter, startTime, endTime)

	// Export data
	return sqlExportHelper(
		ctx,
		events,
		"events",
		[]string{"event_id", "request_device_id", "event_source_device_id", "response_timestamp", "event_timestamp", "field_name", "field_value"},
		func(writer *csv.Writer, event data.StoreEvent) error {
			return writer.Write([]string{
				event.ID,
				event.RequestDeviceID,
				event.EventSourceDeviceID,
				utils.EpochSecondsToExcelDate(event.ResponseTimestamp),
				utils.EpochSecondsToExcelDate(event.EventTimestamp),
				event.FieldName,
				event.FieldValue,
			})
		},
	)
}
