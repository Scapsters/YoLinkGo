package main

import (
	"com/connections/db"
	"com/connections/db/mysql"
	"com/connections/sensors"
	"com/data"
	"com/utils"
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
	dbConnection, err := mysql.NewMySQLConnection("root:101098@tcp(127.0.0.1:3306)/", true)
	if err != nil {
		return fmt.Errorf("error connecting to DB: %w", err)
	}
	defer utils.LogErrors(dbConnection.Close, fmt.Sprintf("error closing db connection %v", dbConnection))

	// Connect to YoLink
	yoLinkConnection, err := utils.Retry2(3, func() (*sensors.YoLinkConnection, error) {
		return sensors.NewYoLinkConnection(
			strings.TrimSpace(os.Getenv("YOLINK_UAID")),
			strings.TrimSpace(os.Getenv("YOLINK_SECRET_KEY")),
		)
	})
	if err != nil {
		return fmt.Errorf("error while creating new YoLink connection: %w", err)
	}
	err = yoLinkConnection.UpdateManagedDevices(dbConnection)
	if err != nil {
		return fmt.Errorf("error while updating YoLink device data: %w", err)
	}

	// Store sensor data 
	fmt.Println("Initial run starting...")
	err = storeAllConnectionSensorData(dbConnection, yoLinkConnection)
	if err != nil {
		return fmt.Errorf("error while storing sensor data: %w", err)
	}

	// Repeat job for 72h. Currently, this function is blocking
	fmt.Println("Scheduling starting...")
	err = scheduleJob(
		func() error {
			fmt.Println("starting")
			err = storeAllConnectionSensorData(dbConnection, yoLinkConnection)
			if err != nil {
				return fmt.Errorf("error while storing sensor data: %w", err)
			}
			return nil
		},
		15*time.Minute,
	)
	if err != nil {
		return fmt.Errorf("error scheduling job: %w", err)
	}

	// Export
	err = utils.Retry1(3, func() error {
		return dbConnection.Events().Export(data.EventFilter{})
	})
	if err != nil {
		return fmt.Errorf("error exporting: %w", err)
	}

	return nil
}

func storeAllConnectionSensorData(dbConnection db.DBConnection, sensorConnection sensors.SensorConnection) error {
	// Get all devices
	devices, err := utils.Retry2(3, func() (*data.IterablePaginatedData[data.StoreDevice], error) { 
		return sensorConnection.GetManagedDevices(dbConnection)
	})
	if err != nil {
		return fmt.Errorf("error while searching for devices: %w", err)
	}

	for {
		device, err := devices.Next()
		if err != nil {
			return fmt.Errorf("error getting next item: %w", err)
		}
		if device == nil {
			break
		}

		// Get device data
		events, err := utils.Retry2(3, func() ([]data.Event, error) { 
			return sensorConnection.GetDeviceState(device)
		})
		if err != nil {
			utils.DefaultSafeLog(fmt.Sprintf("\nerror getting events from device %v: %v\n", device, err))
		}
		// Store device data
		for _, event := range events {
			err = utils.Retry1(3, func() error {
				return dbConnection.Events().Add(event)
			})
			if err != nil {
				utils.DefaultSafeLog(fmt.Sprintf("\nerror adding event to DB %v: %v\n", event, err))
			}
		}
	}
	return nil
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
	time.Sleep(72 * time.Hour)
	err = s.Shutdown()
	if err != nil {
		return fmt.Errorf("error shutting down: %w", err)
	}
	return nil
}