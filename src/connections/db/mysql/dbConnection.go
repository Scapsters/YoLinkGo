package mysql

import (
	"com/connections"
	"com/connections/db"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

const DatabaseName = "yolinktesting"

var _ db.DBConnection = (*MySQLConnection)(nil)

type MySQLConnection struct {
	connectionString string
	eventStore       db.EventStore
	deviceStore      db.DeviceStore
	db               *sql.DB
}

// connectionString excludes the database name and includes the slash at the end
func NewMySQLConnection(connectionString string, isSetupDestructive bool) (*MySQLConnection, error) {
	mySQL := &MySQLConnection{connectionString: connectionString}
	err := mySQL.Open()
	if err != nil {
		return nil, fmt.Errorf("error while connecting to MySQL server: %w", err)
	}
	_, err = mySQL.DB().Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", DatabaseName))
	if err != nil {
		return nil, fmt.Errorf("error while creating database: %w", err)
	}

	db := &MySQLConnection{connectionString: connectionString + DatabaseName}
	err = db.Open()
	if err != nil {
		return nil, fmt.Errorf("error while connecting to database: %w", err)
	}

	// Create stores
	devices := MySQLDeviceStore{DB: db.DB()}
	err = devices.Setup(isSetupDestructive)
	if err != nil {
		return nil, fmt.Errorf("error setting up devices: %w", err)
	}
	db.SetDevices(&devices)

	events := MySQLEventStore{DB: db.DB()}
	err = events.Setup(isSetupDestructive)
	if err != nil {
		return nil, fmt.Errorf("error setting up events: %w", err)
	}
	db.SetEvents(&events)

	return db, nil
}
func (manager *MySQLConnection) Open() error {
	db, err := sql.Open("mysql", manager.connectionString)
	if err != nil {
		return fmt.Errorf("error opening to MySQL via connection string %v: %w", manager.connectionString, err)
	}
	if err = db.Ping(); err != nil {
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
func (manager *MySQLConnection) Status() (connections.PingResult, string) {
	if manager.db == nil {
		return connections.Bad, "db is nil"
	}
	err := manager.db.Ping()
	if err != nil {
		return connections.Bad, "error on db ping"
	}
	return connections.Good, ""
}
func (manager *MySQLConnection) Devices() db.DeviceStore {
	return manager.deviceStore
}
func (manager *MySQLConnection) Events() db.EventStore {
	return manager.eventStore
}
func (manager *MySQLConnection) DB() *sql.DB {
	return manager.db
}
func (manager *MySQLConnection) SetDevices(deviceStore db.DeviceStore) {
	manager.deviceStore = deviceStore
}
func (manager *MySQLConnection) SetEvents(eventStore db.EventStore) {
	manager.eventStore = eventStore
}
