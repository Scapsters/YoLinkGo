package db

import (
	"com/connection"
)

type DBConnection interface {
	connection.Connection
	Devices() DeviceStore
	Events() EventStore
}
