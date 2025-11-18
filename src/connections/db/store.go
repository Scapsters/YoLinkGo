package db

import "com/data"

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
type EditableStore[T any, S any, F any, E any] interface {
	GenericStore[T, S, F]
	// Edit the item matched by the store item and update it with any values present in E
	Edit(storeItem S, item E) error
}

type DeviceStore interface {
	GenericStore[data.Device, data.StoreDevice, data.DeviceFilter]
}

type EventStore interface {
	GenericStore[data.Event, data.StoreEvent, data.EventFilter]
}