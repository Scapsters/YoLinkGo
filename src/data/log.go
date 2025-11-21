package data

// A log as read from a store. Mutations are not implicitly persisted.
type StoreLog struct {
	ID string
	Level int
	StackTrace string
	Description string
}

// A log that is not necessarily associated with a Store object.
type Log struct {
	Level int
	StackTrace string
	Description string
}

// A partial log for querying a store.
type LogFilter struct {
	ID *string
	Level *int
	StackTrace *string
	Description *string
}
