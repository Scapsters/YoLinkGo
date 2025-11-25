package mysql

import (
	"com/connections"
	"com/connections/db"
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const DatabaseName = "yolinktesting"
const RequestTimeout = 60 * time.Second // TODO: look into how long a big request might take

var _ db.DBConnection = (*MySQLConnection)(nil)

type MySQLConnection struct {
	connectionString string
	db               *sql.DB
	eventStore       db.EventStore
	deviceStore      db.DeviceStore
	jobStore         db.JobStore
	logStore         db.LogStore
}

// connectionString excludes the database name and includes the slash at the end.
func NewMySQLConnection(ctx context.Context, connectionString string, isSetupDestructive bool) (*MySQLConnection, error) {
	mySQL := &MySQLConnection{connectionString: connectionString}
	err := mySQL.Open(ctx)
	if err != nil {
		return nil, fmt.Errorf("error while connecting to MySQL server: %w", err)
	}
	sqlctx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()
	_, err = mySQL.DB().ExecContext(sqlctx, fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", DatabaseName))
	if err != nil {
		return nil, fmt.Errorf("error while creating database: %w", err)
	}

	db := &MySQLConnection{connectionString: connectionString + DatabaseName}
	err = db.Open(ctx)
	if err != nil {
		return nil, fmt.Errorf("error while connecting to database: %w", err)
	}

	// Create stores
	devices := NewMySQLDeviceStore(db.db)
	err = devices.Setup(ctx, isSetupDestructive)
	if err != nil {
		return nil, fmt.Errorf("error setting up devices: %w", err)
	}
	db.deviceStore = &devices

	events := NewMySQLEventStore(db.db)
	err = events.Setup(ctx, isSetupDestructive)
	if err != nil {
		return nil, fmt.Errorf("error setting up events: %w", err)
	}
	db.eventStore = &events

	jobs := NewMySQLJobStore(db.db)
	err = jobs.Setup(ctx, isSetupDestructive)
	if err != nil {
		return nil, fmt.Errorf("error setting up events: %w", err)
	}
	db.jobStore = &jobs

	logs := NewMySQLLogStore(db.db)
	err = logs.Setup(ctx, isSetupDestructive)
	if err != nil {
		return nil, fmt.Errorf("error setting up events: %w", err)
	}
	db.logStore = &logs

	return db, nil
}
func (manager *MySQLConnection) Open(ctx context.Context) error {
	db, err := sql.Open("mysql", manager.connectionString)
	if err != nil {
		return fmt.Errorf("error opening to MySQL via connection string %v: %w", manager.connectionString, err)
	}
	context, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()
	err = db.PingContext(context)
	if err != nil {
		return fmt.Errorf("error pinging MySQL via connection string %v: %w", manager.connectionString, err)
	}
	manager.db = db
	return nil
}
func (manager *MySQLConnection) Close() error {
	err := manager.db.Close()
	if err != nil {
		return fmt.Errorf("error while disconnecting from msql db: %w", err)
	}
	return nil
}
func (manager *MySQLConnection) Status(ctx context.Context) (connections.PingResult, string) {
	if manager.db == nil {
		return connections.Bad, "db is nil"
	}
	context, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()
	err := manager.db.PingContext(context)
	if err != nil {
		return connections.Bad, "error on db ping"
	}
	return connections.Good, ""
}
func (manager *MySQLConnection) DB() *sql.DB {
	return manager.db
}
func (manager *MySQLConnection) Devices() db.DeviceStore {
	return manager.deviceStore
}
func (manager *MySQLConnection) Events() db.EventStore {
	return manager.eventStore
}
func (manager *MySQLConnection) Jobs() db.JobStore {
	return manager.jobStore
}
func (manager *MySQLConnection) Logs() db.LogStore {
	return manager.logStore
}
