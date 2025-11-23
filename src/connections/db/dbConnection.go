package db

import (
	"com/connections"
)

type DBConnection interface {
	connections.Connection
	Devices() DeviceStore
	Events() EventStore
	Logs() LogStore
	Jobs() JobStore
}
