package data

// A device as read from a Store. Mutations are not implicitly persisted.
type StoreDevice struct {
	HasID
	Device
}
func (device StoreDevice) GetID() string {
	return device.ID
}

// A device that is not necessarily associated with a Store object.
type Device struct {
	BrandID   string
	Brand     string
	Kind      string
	Name      string
	Token     string
	Timestamp int64
}

// A partial device object for querying.
type DeviceFilter struct {
	ID        *string
	BrandID   *string
	Brand     *string
	Kind      *string
	Name      *string
	Token     *string
	Timestamp *int64
}
