package db

import "com/src/data"

type Store struct {
	devices DeviceStore
	events EventStore
}

type GenericStore[T any, S any, F any] interface {
	Add(item T) error
	Delete(storeItem S) error
	Get(filter F) ([]S, error)
}

type DeviceStore interface {
	GenericStore[data.Device, data.StoreDevice, data.DeviceFilter]
}

type EventStore interface {
	GenericStore[data.Event, data.StoreEvent, data.EventFilter]
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