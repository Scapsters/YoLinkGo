package db

import (
	"com/data"
	"context"
)

const EXPORT_DIR string = "../export"

// T represents the base type of the store.
// S represents the store object type, which is typically the base type with an id field.
// F represents the filter object type, which is typically a partial version of the store object type.
type GenericStore[T any, S data.HasIDGetter, F any] interface {
	// Add the object, return the ID.
	Add(context context.Context, item T) (string, error)
	// Fully remove the given item.
	Delete(context context.Context, storeItem S) error
	// Data is lazily fetched, so there is no error returned from the getter, which merely sets up the query.
	Get(context context.Context, filter F) *data.IterablePaginatedData[S]
	// Create the objects necessary to store data.
	// if isDestructive is false, tables or data should not be destroyed.
	Setup(context context.Context, isDestructive bool) error
	// Export all rows that match the given filter into a csv file into /exports at the root directory of the project (1 above src).
	// Names should follow [export date]_[data label].csv.
	Export(context context.Context, storeItems *data.IterablePaginatedData[S]) error
}

type EditableStore[T any, S data.HasIDGetter, F any] interface {
	GenericStore[T, S, F]
	// Edit the item matched by the store item's ID to have all other values in the item. Errors if target item isn't found
	Edit(context context.Context, storeItem S) error
}

// Stores that have timestamped data, allowing for special querying and exporting methods.
type TimestampedDataStore[T any, S data.HasIDGetter, F any] interface {
	GenericStore[T, S, F]
	// Data is lazily fetched, so there is no error returned from the getter, which merely sets up the query.
	GetInTimeRange(context context.Context, filter F, startTime *int64, endTime *int64) *data.IterablePaginatedData[S]
}

// Stores that have ongoing anc closable events that can be ended.
type ClosableStore[T any, S data.HasIDGetter, F any] interface {
	GenericStore[T, S, F]
	// Ends the given item, typically by setting its end date to the current time.
	Close(context context.Context, storeItem S) error
}

type DeviceStore interface {
	EditableStore[data.Device, data.StoreDevice, data.DeviceFilter]
}

type EventStore interface {
	TimestampedDataStore[data.Event, data.StoreEvent, data.EventFilter]
}

type LogStore interface {
	TimestampedDataStore[data.Log, data.StoreLog, data.LogFilter]
}

type JobStore interface {
	TimestampedDataStore[data.Job, data.StoreJob, data.JobFilter]
	ClosableStore[data.Job, data.StoreJob, data.JobFilter]
}
