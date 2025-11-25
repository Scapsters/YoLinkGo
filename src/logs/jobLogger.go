package logs

import (
	"com/connections/db"
	"com/data"
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
	"time"
)

const logDepth int = 100
const LOG_DIR = "../logs"

// Allow contexts to provide and get loggers.
type loggerKey struct{}

func ContextWithLogger(ctx context.Context, l *JobLogger) context.Context {
	return context.WithValue(ctx, loggerKey{}, l)
}
func Logger(ctx context.Context) *JobLogger {
	v := ctx.Value(loggerKey{})
	if v == nil {
		return nil
	}
	jobLogger, ok := v.(*JobLogger)
	if !ok {
		log.Panic("Non job logger value found in job logger context key. loggerKey is package private. How?")
	}
	return jobLogger
}

type JobCategory string

const (
	Main   JobCategory = "MAIN"
	Export JobCategory = "EXPORT"
	Import JobCategory = "IMPORT"
)

func CreateJob(ctx context.Context, db db.DBConnection, category JobCategory) (*JobLogger, error) {
	return createChildJob(ctx, db, category, nil)
}
func createChildJob(ctx context.Context, db db.DBConnection, category JobCategory, parentJobLogger *JobLogger) (*JobLogger, error) {
	// Create job in db
	timestamp := time.Now().UTC().Unix()
	var parentJobID string
	if parentJobLogger != nil {
		parentJobID = parentJobLogger.job.ID
	}
	job := data.Job{
		ParentID:       parentJobID,
		Category:       string(category),
		StartTimestamp: timestamp,
		EndTimestamp:   0,
	}
	id, err := db.Jobs().Add(ctx, job)
	if err != nil {
		return nil, fmt.Errorf("unable to create job under ctx %v and category %v with connection %v: %w", ctx, category, db, err)
	}

	// Create log file
	timestampDate := time.Unix(timestamp, 0).Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf(
		"%s/%s_job_log_%v.csv",
		LOG_DIR,
		timestampDate,
		id,
	)
	file, err := os.Create(filename)
	if err != nil {
		FDefaultLog("error creating or opening export file: %v", err)
	}
	defer LogErrorsWithContext(ctx, file.Close, fmt.Sprintf("error closing file %v", filename))

	// Return logger
	return &JobLogger{
		db:              db,
		job:			 data.StoreJob{Job: job, HasID: data.HasID{ID: id}},
		fileMutex:       &sync.Mutex{},
		timestamp:       timestamp,
		filename:        filename,
		parentJobLogger: parentJobLogger,
	}, nil
}

// Logs to the database, a file, and to stdout.
type JobLogger struct {
	db              db.DBConnection
	timestamp       int64
	job             data.StoreJob
	parentJobLogger *JobLogger
	filename        string
	fileMutex       *sync.Mutex
}
func (l *JobLogger) End(ctx context.Context) {
	err := l.db.Jobs().Close(ctx, l.job)
	if err != nil {
		l.Error(ctx, "Unable to end log %v: %v", l, err)
	}
}
func (l *JobLogger) Debug(ctx context.Context, fstring string, args ...any) {
	l.log(ctx, 4, fstring, args...)
}
func (l *JobLogger) Info(ctx context.Context, fstring string, args ...any) {
	l.log(ctx, 3, fstring, args...)
}
func (l *JobLogger) Warn(ctx context.Context, fstring string, args ...any) {
	l.log(ctx, 2, fstring, args...)
}
func (l *JobLogger) Error(ctx context.Context, fstring string, args ...any) {
	l.log(ctx, 1, fstring, args...)
}
func (l *JobLogger) CreateChildJob(ctx context.Context, category JobCategory) (*JobLogger, error) {
	return createChildJob(ctx, l.db, category, l)
}
func (l *JobLogger) log(ctx context.Context, level int, fstring string, args ...any) {
	// Create entry
	stackBuffer := make([]byte, 64*1024)
	numBytes := runtime.Stack(stackBuffer, false)
	entry := data.Log{
		JobID:       l.job.ID,
		Level:       level,
		StackTrace:  string(stackBuffer[:numBytes]),
		Description: fmt.Sprintf(fstring, args...),
		Timestamp:   time.Now().UTC().Unix(),
	}
	formattedEntry := fmt.Sprintf(
		"[%v] %v: %v [Job ID: %v]",
		entry.Level,
		time.Unix(entry.Timestamp, 0).UTC().Format("2006-01-02 15:04:05"),
		entry.Description,
		entry.JobID,
	)

	// Log to default
	FDefaultLog("%s", formattedEntry)

	// Log to db
	_, err := l.db.Logs().Add(ctx, entry)
	if err != nil {
		FDefaultLog("error adding log to database: %v", err)
	}

	// Log to file
	l.logToFileAndParentFiles(ctx, formattedEntry)
}
func (l *JobLogger) logToFileAndParentFiles(ctx context.Context, stringToLog string) {
	if l.parentJobLogger != nil {
		l.parentJobLogger.logToFileAndParentFiles(ctx, stringToLog)
	}

	// Create directory
	var OwnerReadWriteExecuteAndOthersReadExecute = 0755
	err := os.MkdirAll(LOG_DIR, os.FileMode(OwnerReadWriteExecuteAndOthersReadExecute))
	if err != nil {
		FDefaultLog("error creating export directory: %v", err)
	}

	// Open file
	var OwnerReadWriteAndOthersRead = 0644
	f, err := os.OpenFile(l.filename, os.O_RDWR, os.FileMode(OwnerReadWriteAndOthersRead))
	if err != nil {
		FDefaultLog("error creating or opening export file: %v", err)
	}
	defer LogErrorsWithContext(ctx, f.Close, fmt.Sprintf("error closing file %v", l.filename))

	// Write to file
	l.fileMutex.Lock()
	defer l.fileMutex.Unlock()
	_, err = f.WriteString(stringToLog + "\n")
	if err != nil {
		FDefaultLog("error writing to file: %v", err)
	}
}
