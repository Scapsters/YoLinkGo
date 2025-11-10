package mysql

import (
	"com/src/connection"
	"com/src/db"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

const DatabaseName = "yolinktesting"

var _ db.DBConnection = (*MySQLConnection)(nil)

type MySQLConnection struct {
	ConnectionString string
	db               *sql.DB
}

// connectionString excludes the database name and includes the slash at the end
func NewMySQLConnectionManager(connectionString string) (*MySQLConnection, error) {

	mySQL := &MySQLConnection{ConnectionString: connectionString}
	err := mySQL.Open()
	if err != nil {
		return nil, fmt.Errorf("error while connecting to MySQL server: %w", err)
	}
	_, err = mySQL.DB().Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", DatabaseName))
	if err != nil {
		return nil, fmt.Errorf("error while creating database: %w", err)
	}

	db := &MySQLConnection{ConnectionString: connectionString + DatabaseName}
	err = db.Open()
	if err != nil {
		return nil, fmt.Errorf("error while connecting to database: %w", err)
	}
	return db, nil
}
func (manager *MySQLConnection) Open() error {
	db, err := sql.Open("mysql", manager.ConnectionString)
	if err != nil {
		return fmt.Errorf("error opening to MySQL via connection string %v: %w", manager.ConnectionString, err)
	}
	if err = db.Ping(); err != nil {
		return fmt.Errorf("error pinging MySQL via connection string %v: %w", manager.ConnectionString, err)
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
func (manager *MySQLConnection) Status() (connection.PingResult, string) {
	if manager.db == nil {
		return connection.Bad, "db is nil"
	}
	err := manager.db.Ping()
	if err != nil {
		return connection.Bad, "error on db ping"
	}
	return connection.Good, ""
}
func (manager *MySQLConnection) DB() *sql.DB {
	return manager.db
}
