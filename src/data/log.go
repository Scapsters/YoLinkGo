package data

import "strconv"

// A log as read from a store. Mutations are not implicitly persisted.
var _ HasIDGetterAndSpreadable[StoreLog] = StoreLog{}
type StoreLog struct {
	HasID
	Log
}
func (log StoreLog) GetID() string {
	return log.ID
}
func (l StoreLog) Spread() []any {
	return []any{
		l.ID,
		l.JobID,
		l.Level,
		l.StackTrace,
		l.Description,
		l.Timestamp,
	}
}
func (l StoreLog) SpreadForExport() []string {
	return []string{
		l.ID,
		l.JobID,
		strconv.Itoa(l.Level),
		l.StackTrace,
		l.Description,
		EpochSecondsToExcelDate(l.Timestamp),
	}
}
func (l StoreLog) SpreadAddresses() (*StoreLog, []any) {
	return &l, []any{
		&l.ID,
		&l.JobID,
		&l.Level,
		&l.StackTrace,
		&l.Description,
		&l.Timestamp,
	}
}


// A log that is not necessarily associated with a Store object.
var _ Spreadable = Log{}
type Log struct {
	JobID       string
	Level       int
	StackTrace  string
	Description string
	Timestamp   int64
}
func (l Log) Spread() []any {
	return []any{
		l.JobID,
		l.Level,
		l.StackTrace,
		l.Description,
		l.Timestamp,
	}
}

// A partial log for querying a store.
var _ Spreadable = LogFilter{}
type LogFilter struct {
	ID          *string
	JobID       *string
	Level       *int
	StackTrace  *string
	Description *string
	Timestamp   *int64
}
func (l LogFilter) Spread() []any {
	return []any{
		l.ID,
		l.JobID,
		l.Level,
		l.StackTrace,
		l.Description,
		l.Timestamp,
	}
}