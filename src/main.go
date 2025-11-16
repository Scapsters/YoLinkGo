package main

import (
	"com/data"
	"com/db"
	"com/db/mysql"
	"com/sensors"
	utils "com/util"
	"fmt"
	"log"
	"os"
	"strings"

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
	// Connect to DB
	dbConnection, err := mysql.NewMySQLConnection("root:101098@tcp(127.0.0.1:3306)/")
	if err != nil {
		return fmt.Errorf("error connecting to DB: %w", err)
	}
	defer dbConnection.Close() // ignore error
	status, description := dbConnection.Status()
	fmt.Printf("MySQL connection status: %v, description: %v\n", status, description)
	stores := db.StoreCollection{
		Devices: &mysql.MySQLDeviceStore{DB: dbConnection.DB()},
		Events:  &mysql.MySQLEventStore{DB: dbConnection.DB()},
	}
	stores.Devices.Setup(true)
	stores.Events.Setup(true)

	// Connect to YoLink
	yoLinkConnection, err := sensors.NewYoLinkConnection(
		strings.TrimSpace(os.Getenv("YOLINK_UAID")),
		strings.TrimSpace(os.Getenv("YOLINK_SECRET_KEY")),
	)
	if err != nil {
		return fmt.Errorf("error while creating new YoLink connection: %w", err)
	}
	status, description = yoLinkConnection.Status()
	fmt.Printf("YoLink connection status: %v, description: %v\n", status, description)

	// Get device List
	result, err := sensors.MakeYoLinkRequest[sensors.TypedBUDP[sensors.YoLinkDeviceList]](yoLinkConnection, sensors.SimpleBDDP{Method: sensors.HomeGetDeviceList})
	if err != nil {
		return fmt.Errorf("error while getting YoLink device list: %w", err)
	}
	if result == nil {
		return fmt.Errorf("YoLink device list null without associated error")
	}

	// Store devices
	for _, device := range result.Data.Devices {
		stores.Devices.Add(data.Device{
			Kind: device.Kind,
			Name: device.Name,
			Token: device.Token,
			ID: device.DeviceID,
			Timestamp: fmt.Sprintf("%v", utils.TimeSeconds()),
		})
	}

	// Query device info
	ID := "d88b4c01000277a9"
	devices, err := stores.Devices.Get(data.DeviceFilter{
		ID: &ID,
	})
	if err != nil {
		return fmt.Errorf("error while seraching for device: %w", err)
	}
	if len(devices) == 0 {
		return fmt.Errorf("device not found")
	}
	device := devices[0]

	deviceState, err := sensors.MakeYoLinkRequest[sensors.BUDP](yoLinkConnection, sensors.SimpleBDDP{Method: sensors.THSensorGetState, TargetDevice: &device.ID, Token: &device.Token})
	if err != nil {
		return fmt.Errorf("error while quering device: %w", err)
	}
	fmt.Println(deviceState.Data)

	return nil
}