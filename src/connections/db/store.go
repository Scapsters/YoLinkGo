package db

import "com/data"

const EXPORT_DIR string = "../export"

// T represents the base type of the store
// S represents the store object type, which is typically the base type with an id field
// F represents the filter object type, which is typically a partial version of the store object type
type GenericStore[T any, S any, F any] interface {
	Add(item T) error
	Delete(storeItem S) error
	Get(filter F) (*data.IterablePaginatedData[S], error)
	// Create the objects neccesary to store data.
	// if isDestructive is false, tables or data should not be destroyed
	Setup(isDestructive bool) error
	// Export all rows that match the given filter into a csv file into /exports at the root directory of the project (1 above src).
	// Names should follow [export date]_[data label].csv
	Export(filter F) error
}

type EditableStore[T any, S any, F any] interface {
	GenericStore[T, S, F]
	// Edit the item matched by the store item's ID to have all other values in the item
	Edit(storeItem S) error
}

type TimestampedDataStore[T any, S any, F any] interface {
	GenericStore[T, S, F]
	// Exports all rows that match the given filter and date range. The date range should refer to the recording time of the data in the row.
	ExportInTimeRange(filter F, startTime *int64, endTime *int64) error
	GetInTimeRange(filter F, startTime *int64, endTime *int64) (*data.IterablePaginatedData[S], error)
}

type DeviceStore interface {
	EditableStore[data.Device, data.StoreDevice, data.DeviceFilter]
}

type EventStore interface {
	TimestampedDataStore[data.Event, data.StoreEvent, data.EventFilter]
}
