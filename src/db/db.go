package db

import "com/src/data"

type Store interface {
	DeviceStore
	EventStore
}

type DeviceStore interface {
	AddDevice(device data.Device) error
	DeleteDevice(device data.Device) error
	GetDevices(filter data.DeviceFilter) ([]data.Device, error)
}

type EventStore interface {
	AddEvent(event data.Event) error
	DeleteEvent(event data.Event) error
	GetEvents(filter data.EventFilter)
}

type StoreConnectionStatus int
const (
	Unknown StoreConnectionStatus = iota
	Connected
	Disconnected
)

type ConnectionManager interface {
	// No error implies connection status is Connected
	Connect(connectionString string) error
	// No error implies connection status is Disconnected
	Disconnect() error
	Status() StoreConnectionStatus
}