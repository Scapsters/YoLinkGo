package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"com/data"
	"com/db"
	"com/db/mysql"
	"com/sensors"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load("../.env")
	if err != nil {
		log.Fatal("fatal:", err)
	}
	if err := run(); err != nil {
		log.Fatal("fatal:", err)
	}
}

func run() error {
	// err := DBTesting()
	// if err != nil {
	// 	return err
	// }

	err := YoLinkTesting()
	if err != nil {
		return err
	}

	return nil
}

func YoLinkTesting() error {
	uaid := strings.TrimSpace(os.Getenv("YOLINK_UAID"))
	secretKey := strings.TrimSpace(os.Getenv("YOLINK_SECRET_KEY"))
	yoLinkConnection, err := sensors.NewYoLinkConnection(uaid, secretKey)
	if err != nil {
		return fmt.Errorf("error while creating new YoLink connection: %w", err)
	}

	status, description := yoLinkConnection.Status()
	fmt.Printf("YoLink connection status: %v, description: %v\n", status, description)

	result, err := sensors.MakeYoLinkRequest[sensors.TypedBUDP[sensors.YoLinkDeviceList]](yoLinkConnection, sensors.SimpleBDDP{Method: sensors.HomeGetDeviceList})
	if err != nil {
		return fmt.Errorf("error while getting YoLink device list: %w", err)
	}
	if result == nil {
		return fmt.Errorf("YoLink device list null without associated error")
	}

	fmt.Println(result.Data.Devices)

	return nil
}

func DBTesting() error {
	fmt.Println("Connecting to MySQL, creating DB if neccesary, and connecting to DB...")
	dbManager, err := mysql.NewMySQLConnectionManager("root:101098@tcp(127.0.0.1:3306)/")
	if err != nil {
		return fmt.Errorf("error connecting to DB: %w", err)
	}
	fmt.Println("Connected successfully")
	defer func() {
		if err := dbManager.Close(); err != nil {
			fmt.Println("Warning: failed to close DB:", err)
		}
	}()
	connectionStatus, connectionDescription := dbManager.Status()
	fmt.Printf("Connection Status: %v\n", connectionStatus.String())
	if connectionDescription != "" {
		fmt.Printf("Connection Description: %v\n", connectionDescription)
	}

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
		Kind:      "TestKind",
		Name:      "TestName",
		Token:     "TestToken",
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