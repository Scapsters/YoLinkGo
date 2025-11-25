package data

// A job as read from a store. Mutations are not implicitly persisted.
var _ HasIDGetterAndSpreadable = StoreJob{}
type StoreJob struct {
	HasID
	Job
}
func (j StoreJob) GetID() string {
	return j.ID
}
func (j StoreJob) Spread() []any {
	return []any{
		j.ID,
		j.ParentID,
		j.Category,
		j.StartTimestamp,
		j.EndTimestamp,
	}
}
func (j StoreJob) SpreadAddresses() []any {
	return []any{
		j.ID,
		j.ParentID,
		j.Category,
		j.StartTimestamp,
		j.EndTimestamp,
	}
}
func (j StoreJob) SpreadForExport() []string {
	return []string{
		j.ID,
		j.ParentID,
		j.Category,
		EpochSecondsToExcelDate(j.StartTimestamp),
		EpochSecondsToExcelDate(j.EndTimestamp),
	}
}

// A job that is not necessarily associated with a Store object.
var _ Spreadable = Job{}
type Job struct {
	ParentID       string
	Category       string
	StartTimestamp int64
	EndTimestamp   int64
}
func (j Job) Spread() []any {
	return []any{
		j.ParentID,
		j.Category,
		j.StartTimestamp,
		j.EndTimestamp,
	}
}

// A partial job for querying a store.
var _ Spreadable = JobFilter{}
type JobFilter struct {
	ID             *string
	ParentID       *string
	Category       *string
	StartTimestamp *int64
	EndTimestamp   *int64
}
func (j JobFilter) Spread() []any {
	return []any{
		j.ParentID,
		j.Category,
		j.StartTimestamp,
		j.EndTimestamp,
	}
}
