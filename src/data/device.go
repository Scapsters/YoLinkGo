package data

// A device as read from a Store. Mutations are not implicitly persisted.
var _ HasIDGetterAndSpreadable = StoreDevice{}
type StoreDevice struct {
	HasID
	Device
}
func (device StoreDevice) GetID() string {
	return device.ID
}
func (e StoreDevice) Spread() []any {
	return []any{
		e.ID,
		e.BrandID,
		e.Brand,
		e.Kind,
		e.Name,
		e.Token,
		e.Timestamp,
	}
}
func (e StoreDevice) SpreadAddresses() []any {
	return []any{
		&e.ID,
		&e.BrandID,
		&e.Brand,
		&e.Kind,
		&e.Name,
		&e.Token,
		&e.Timestamp,
	}
}
func (e StoreDevice) SpreadForExport() []string {
	return []string{
		e.ID,
		e.BrandID,
		e.Brand,
		e.Kind,
		e.Name,
		e.Token,
		EpochSecondsToExcelDate(e.Timestamp),
	}
}

// A device that is not necessarily associated with a Store object.
var _ Spreadable = Device{}
type Device struct {
	BrandID   string
	Brand     string
	Kind      string
	Name      string
	Token     string
	Timestamp int64
}
func (e Device) Spread() []any {
	return []any{
		e.BrandID,
		e.Brand,
		e.Kind,
		e.Name,
		e.Token,
		e.Timestamp,
	}
}

// A partial device object for querying.
var _ Spreadable = DeviceFilter{}
type DeviceFilter struct {
	ID        *string
	BrandID   *string
	Brand     *string
	Kind      *string
	Name      *string
	Token     *string
	Timestamp *int64
}

func (d DeviceFilter) Spread() []any {
	return []any{
		d.ID,
		d.BrandID,
		d.Brand,
		d.Kind,
		d.Name,
		d.Token,
		d.Timestamp,
	}
}
