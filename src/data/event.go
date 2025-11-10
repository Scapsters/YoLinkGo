package data

// An event as read from a store. Mutations are not implicitly persisted
type StoreEvent struct {
	ID                  int
	RequestDeviceID     string
	EventSourceDeviceID string
	ResponseTimestamp   string
	EventTimestamp      string
	FieldName           string
	FieldValue          string
}

// An event that is not neccesarily associated with a Store object
type Event struct {
	RequestDeviceID     string
	EventSourceDeviceID string
	ResponseTimestamp   string
	EventTimestamp      string
	FieldName           string
	FieldValue          string
}

// A partial event for querying a store
type EventFilter struct {
	ID                  *int
	RequestDeviceID     *string
	EventSourceDeviceID *string
	ResponseTimestamp   *string
	EventTimestamp      *string
	FieldName           *string
	FieldValue          *string
}

// A partial device object that excludes id for editing
type EventEdit struct {
	RequestDeviceID     *string
	EventSourceDeviceID *string
	ResponseTimestamp   *string
	EventTimestamp      *string
	FieldName           *string
	FieldValue          *string
}
