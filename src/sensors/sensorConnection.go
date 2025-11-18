package sensors

import (
	"com/connection"
	"com/data"
	"com/db"
)

type SensorConnection interface {
	connection.Connection
	// Queries the API for the current state of the device
	GetDeviceState(data.StoreDevice) ([]data.Event, error)
	// Queries the DB for all devices that are able to be managed by the connection
	GetManagedDevices(db.DBConnection) ([]data.StoreDevice, error)
	// Queries the API for all available devices (if possible)
	UpdateManagedDevices(db.DBConnection) (error)
}