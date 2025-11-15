package db

import (
	"com/connection"
	"com/data"
	"database/sql"
)

type StoreCollection struct {
	Devices DeviceStore
	Events  EventStore
}

// T represents the base type of the store
// S represents the store object type, which is typically the base type with an id field
// F represents the filter object type, which is typically a partial version of the store object type
type GenericStore[T any, S any, F any] interface {
	Add(item T) error
	Delete(storeItem S) error
	Get(filter F) ([]S, error)
	// Create the objects neccesary to store data.
	// if isDestructive is false, tables or data should not be destroyed
	Setup(isDestructive bool) error
}

// E represents the edit object type, which is typically a partial version of the base type
type EditableStore[E any] interface {
	// Edit the item corresponding to id to have the information of item.
	Edit(id int, item E) error
}

type DeviceStore interface {
	GenericStore[data.Device, data.StoreDevice, data.DeviceFilter]
}

type EventStore interface {
	GenericStore[data.Event, data.StoreEvent, data.EventFilter]
}

type DB interface {
	DB() *sql.DB
}

type DBConnection interface {
	connection.Connection
	DB
}
