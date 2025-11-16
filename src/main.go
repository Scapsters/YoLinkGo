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
	status, description := yoLinkConnection.Status()
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
		err := stores.Devices.Add(data.Device{
			Brand: 	   sensors.YOLINK_BRAND_NAME,
			Kind:      device.Kind,
			Name:      device.Name,
			Token:     device.Token,
			ID:        device.DeviceID,
			Timestamp: utils.TimeSeconds(),
		})
		if err != nil {
			return fmt.Errorf("error adding device %v: %w", device, err)
		}
	}

	// Gather data
	err = gatherAllConnectionSensorData(stores, yoLinkConnection)
	if err != nil {
		return fmt.Errorf("error while gathering YoLink sensor data: %w", err)
	}

	// create a scheduler
	s, err := gocron.NewScheduler()
	if err != nil {
		return fmt.Errorf("error creating scheduler: %w", err)
	}

	// add a job to the scheduler
	j, err := s.NewJob(
		gocron.DurationJob(
			5*time.Minute,
		),
		gocron.NewTask(
			func() {
				fmt.Println("starting")
				gatherAllConnectionSensorData(stores, yoLinkConnection)
			},
		),
	)
	if err != nil {
		return fmt.Errorf("error creating job: %w", err)
	}
	// each job has a unique id
	fmt.Println(j.ID())

	// start the scheduler
	s.Start()
	fmt.Println(s.JobsWaitingInQueue())

	// block until you are ready to shut down
	time.Sleep(24 * time.Hour)

	// when you're done, shut it down
	err = s.Shutdown()
	if err != nil {
		return fmt.Errorf("error shutting down: %w", err)
	}

	return nil
}

func gatherAllConnectionSensorData(stores db.StoreCollection, sensorConnection sensors.SensorConnection) error {
	devices, err := sensorConnection.GetManagedDevices(stores.Devices)
	if err != nil {
		return fmt.Errorf("error while seraching for devices: %w", err)
	}
	if len(devices) == 0 {
		return fmt.Errorf("devices not found")
	}

	for _, device := range devices {
		events, err := sensorConnection.GetDeviceState(device)
		if err != nil {
			log.Default().Output(1, fmt.Sprintf("error getting events from YoLink device %v: %v", device, err))
		}
		for _, event := range events {
			err := stores.Events.Add(event)
			if err != nil {
				log.Default().Output(1, fmt.Sprintf("error adding event to DB %v: %v", event, err))
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
	status, description := dbConnection.Status()
	fmt.Printf("MySQL connection status: %v, description: %v\n", status, description)
	stores := db.StoreCollection{
		Devices: &mysql.MySQLDeviceStore{DB: dbConnection.DB()},
		Events:  &mysql.MySQLEventStore{DB: dbConnection.DB()},
	}
	stores.Devices.Setup(true)
	stores.Events.Setup(true)
	return &stores, nil
}