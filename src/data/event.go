package data

type Event struct {
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
