package data

// A job as read from a store. Mutations are not implicitly persisted.
type StoreJob struct {
	ID  	  string
	Category  string
	StartTimestamp int64
	EndTimestamp int64
}

// A job that is not necessarily associated with a Store object.
type Job struct {
	Category  string
	StartTimestamp int64
	EndTimestamp	int64
}

// A partial job for querying a store.
type JobFilter struct {
	ID  	  *string
	Category  *string
	StartTimestamp *int64
	EndTimestamp	*int64
}
