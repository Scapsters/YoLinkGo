package data

// A device as read from a Store. Mutations are not implicitly persisted
type StoreEvent struct {
	Id        string
	Kind      string
	Name      string
	Token     string
	Timestamp string
}

// A device that is not neccesarily associated with a Store object 
type Device struct {
	Kind      string
	Name      string
	Token     string
	Timestamp string
}

// A partial device object for querying
type DeviceFilter struct {
	Id        *string
	Kind      *string
	Name      *string
	Token     *string
	Timestamp *string
}
