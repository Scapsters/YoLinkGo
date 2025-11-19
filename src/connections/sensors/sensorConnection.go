package sensors

import (
	"com/connections"
	"com/connections/db"
	"com/data"
)

type SensorConnection interface {
	connections.Connection
	// Queries the API for the current state of the device
	GetDeviceState(device *data.StoreDevice) ([]data.Event, error)
	// Queries the DB for all devices that are able to be managed by the connection
	GetManagedDevices(connection db.DBConnection) (*data.IterablePaginatedData[data.StoreDevice], error)
	// Queries the API for all available devices (if possible)
	UpdateManagedDevices(connection db.DBConnection) error
}
