package data

type Device struct {
	Id        string
	Kind      string
	Name      string
	Token     string
	Timestamp string
}

type DeviceFilter struct {
	Id        *string
	Kind      *string
	Name      *string
	Token     *string
	Timestamp *string
}
