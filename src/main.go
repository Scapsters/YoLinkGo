package main

import (
	"com/db"
	"com/db/mysql"
	"com/sensors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/go-co-op/gocron/v2"
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
	stores, err := connectToStores()
	if err != nil {
		return fmt.Errorf("error while connecting to stores: %w", err)
	}

	// Connect to YoLink
	yoLinkConnection, err := sensors.NewYoLinkConnection(
		strings.TrimSpace(os.Getenv("YOLINK_UAID")),
		strings.TrimSpace(os.Getenv("YOLINK_SECRET_KEY")),
	)
	if err != nil {
		return fmt.Errorf("error while creating new YoLink connection: %w", err)
	}
	err = yoLinkConnection.UpdateManagedDevices(stores.Devices)
	if err != nil {
		return fmt.Errorf("error while updating YoLink device data: %w", err)
	}

	// Store sensor data 
	err = storeAllConnectionSensorData(stores, yoLinkConnection)
	if err != nil {
		return fmt.Errorf("error while storing sensor data: %w", err)
	}

	// Repeat job for 24h. Currently, this function is blocking
	scheduleJob(
		func() {
			fmt.Println("starting")
			storeAllConnectionSensorData(stores, yoLinkConnection)
		},
		5*time.Minute,
	)
	return nil
}

func storeAllConnectionSensorData(stores *db.StoreCollection, sensorConnection sensors.SensorConnection) error {
	// Get all devices
	devices, err := sensorConnection.GetManagedDevices(stores.Devices)
	if err != nil {
		return fmt.Errorf("error while seraching for devices: %w", err)
	}
	if len(devices) == 0 {
		return fmt.Errorf("devices not found")
	}

	// Get device data
	for _, device := range devices {
		events, err := sensorConnection.GetDeviceState(device)
		if err != nil {
			log.Default().Output(1, fmt.Sprintf("\nerror getting events from device %v: %v\n", device, err))
		}
		for _, event := range events {
			err := stores.Events.Add(event)
			if err != nil {
				log.Default().Output(1, fmt.Sprintf("\nerror adding event to DB %v: %v\n", event, err))
			}
		}
	}
	return nil
}

func connectToStores() (*db.StoreCollection, error) {
	// Connect to DB
	dbConnection, err := mysql.NewMySQLConnection("root:101098@tcp(127.0.0.1:3306)/")
	if err != nil {
		return nil, fmt.Errorf("error connecting to DB: %w", err)
	}
	defer dbConnection.Close() // ignore error
	
	// Create stores
	stores := db.StoreCollection{
		Devices: &mysql.MySQLDeviceStore{DB: dbConnection.DB()},
		Events:  &mysql.MySQLEventStore{DB: dbConnection.DB()},
	}
	stores.Devices.Setup(true)
	stores.Events.Setup(true)
	return &stores, nil
}

func scheduleJob(function any, interval time.Duration) error {
	s, err := gocron.NewScheduler()
	if err != nil {
		return fmt.Errorf("error creating scheduler: %w", err)
	}
	_, err = s.NewJob(gocron.DurationJob(interval), gocron.NewTask(function))
	if err != nil {
		return fmt.Errorf("error creating job: %w", err)
	}
	s.Start()
	time.Sleep(24 * time.Hour)
	err = s.Shutdown()
	if err != nil {
		return fmt.Errorf("error shutting down: %w", err)
	}
	return nil
}