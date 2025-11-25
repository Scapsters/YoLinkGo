package sensors

import (
	"com/connections"
	"com/connections/db"
	"com/data"
	"context"
)

type SensorConnection interface {
	connections.Connection
	// Queries the API for the current state of the device
	GetDeviceState(ctx context.Context, device *data.StoreDevice) ([]data.Event, error)
	// Queries the DB for all devices that are able to be managed by the connection
	GetManagedDevices(ctx context.Context, connection db.DBConnection) (*data.IterablePaginatedData[data.StoreDevice], error)
	// Queries the API for all available devices (if possible)
	UpdateManagedDevices(ctx context.Context, connection db.DBConnection) error
}
