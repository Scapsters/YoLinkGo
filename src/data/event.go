package data

// An event as read from a store. Mutations are not implicitly persisted.
var _ HasIDGetterAndSpreadable = StoreEvent{}
type StoreEvent struct {
	HasID
	Event
}
func (event StoreEvent) GetID() string {
	return event.ID
}
func (e StoreEvent) Spread() []any {
	return []any{
		e.RequestDeviceID,
		e.EventSourceDeviceID,
		e.ResponseTimestamp,
		e.EventTimestamp,
		e.FieldName,
		e.FieldValue,
	}
}
func (e StoreEvent) SpreadAddresses() []any {
	return []any{
		&e.RequestDeviceID,
		&e.EventSourceDeviceID,
		&e.ResponseTimestamp,
		&e.EventTimestamp,
		&e.FieldName,
		&e.FieldValue,
	}
}
func (e StoreEvent) SpreadForExport() []string {
	return []string{
		e.RequestDeviceID,
		e.EventSourceDeviceID,
		EpochSecondsToExcelDate(e.ResponseTimestamp),
		EpochSecondsToExcelDate(e.EventTimestamp),
		e.FieldName,
		e.FieldValue,
	}
}

// An event that is not necessarily associated with a Store object.
var _ Spreadable = Event{}
type Event struct {
	RequestDeviceID     string
	EventSourceDeviceID string
	ResponseTimestamp   int64
	EventTimestamp      int64
	FieldName           string
	FieldValue          string
}
func (e Event) Spread() []any {
	return []any{
		e.RequestDeviceID,
		e.EventSourceDeviceID,
		e.ResponseTimestamp,
		e.EventTimestamp,
		e.FieldName,
		e.FieldValue,
	}
}

// A partial device object for querying.
var _ Spreadable = StoreDevice{}
type EventFilter struct {
	ID                  *string
	RequestDeviceID     *string
	EventSourceDeviceID *string
	ResponseTimestamp   *int64
	EventTimestamp      *int64
	FieldName           *string
	FieldValue          *string
}
func (e EventFilter) Spread() []any {
	return []any{
		e.ID,
		e.RequestDeviceID,
		e.EventSourceDeviceID,
		e.ResponseTimestamp,
		e.EventTimestamp,
		e.FieldName,
		e.FieldValue,
	}
}

