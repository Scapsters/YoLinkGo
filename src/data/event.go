package data

// An event as read from a store. Mutations are not implicitly persisted
type StoreEvent struct {
	ID                int
	DeviceName        string
	DeviceKind        string
	RequestDeviceID   string
	ResponseDeviceID  string
	ResponseTimestamp string
	EventTimestamp    string
	FieldName         string
	FieldSource       string
	FieldValue        string
}

// An event that is not neccesarily associated with a Store object 
type Event struct {
	DeviceName        string
	DeviceKind        string
	RequestDeviceID   string
	ResponseDeviceID  string
	ResponseTimestamp string
	EventTimestamp    string
	FieldName         string
	FieldSource       string
	FieldValue        string
}

// A partial event for querying a store
type EventFilter struct {
	ID                *int
	DeviceName        *string
	DeviceKind        *string
	RequestDeviceID   *string
	ResponseDeviceID  *string
	ResponseTimestamp *string
	EventTimestamp    *string
	FieldName         *string
	FieldSource       *string
	FieldValue        *string
}
