package data

// A log as read from a store. Mutations are not implicitly persisted.
type StoreLog struct {
	Log
	
	ID          string
}

// A log that is not necessarily associated with a Store object.
type Log struct {
	JobID       string
	Level       int
	StackTrace  string
	Description string
	Timestamp   int64
}

// A partial log for querying a store.
type LogFilter struct {
	ID          *string
	JobID       *string
	Level       *int
	StackTrace  *string
	Description *string
	Timestamp   *int64
}
