package data

// An event as read from a store. Mutations are not implicitly persisted.
type StoreEvent struct {
	Event
	
	ID                  string
}

// An event that is not necessarily associated with a Store object.
type Event struct {
	RequestDeviceID     string
	EventSourceDeviceID string
	ResponseTimestamp   int64
	EventTimestamp      int64
	FieldName           string
	FieldValue          string
}

// A partial event for querying a store.
type EventFilter struct {
	ID                  *string
	RequestDeviceID     *string
	EventSourceDeviceID *string
	ResponseTimestamp   *int64
	EventTimestamp      *int64
	FieldName           *string
	FieldValue          *string
}
