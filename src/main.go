package main

import (
	"fmt"
	"log"

	"com/src/data"
	"com/src/db"
	"com/src/db/mysql"
)

func main() {
	if err := run(); err != nil {
		log.Fatal("fatal:", err)
	}
}

func run() error {
	fmt.Println("Connecting to MySQL, creating DB if neccesary, and connecting to DB...")
	dbManager, err := mysql.NewMySQLConnectionManager("root:101098@tcp(127.0.0.1:3306)/")
	if err != nil {
		return fmt.Errorf("error connecting to DB: %w", err)
	}
	fmt.Println("Connected successfully")
	defer func() {
		if err := dbManager.Disconnect(); err != nil {
			fmt.Println("Warning: failed to disconnect DB:", err)
		}
	}()
	fmt.Printf("Connection Status: %v\n", dbManager.Status().String())

	stores := db.StoreCollection{
		Devices: &mysql.MySQLDeviceStore{DB: dbManager.DB()},
		Events:  &mysql.MySQLEventStore{DB: dbManager.DB()},
	}
	err = stores.Devices.Setup(true)
	if err != nil {
		fmt.Print(err)
	}
	err = stores.Events.Setup(true)
	if err != nil {
		fmt.Print(err)
	}

	err = stores.Devices.Add(data.Device{
		Kind: "TestKind",
		Name: "TestName",
		Token: "TestToken",
		Timestamp: "TestTimestamp",
	})
	if err != nil {
		fmt.Print(err)
	}

	kindSearch := "TestKind"
	result, err := stores.Devices.Get(data.DeviceFilter{
		Kind: &kindSearch,
	})
	if err != nil {
		fmt.Print(err)
	}
	fmt.Println(result)

	return nil
}
