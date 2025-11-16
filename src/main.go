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
			Kind:      device.Kind,
			Name:      device.Name,
			Token:     device.Token,
			ID:        device.DeviceID,
			Timestamp: utils.TimeSeconds(),
		})
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
				gatherTHData(stores, yoLinkConnection)
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

func gatherTHData(stores db.StoreCollection, yoLinkConnection *sensors.YoLinkConnection) error {
	// Query devices
	kind := "THSensor"
	devices, err := stores.Devices.Get(data.DeviceFilter{Kind: &kind})
	if err != nil {
		return fmt.Errorf("error while seraching for device: %w", err)
	}
	if len(devices) == 0 {
		return fmt.Errorf("device not found")
	}

	// Make request
	for _, device := range devices {
		deviceState, err := sensors.MakeYoLinkRequest[sensors.BUDP](yoLinkConnection, sensors.SimpleBDDP{Method: sensors.THSensorGetState, TargetDevice: &device.ID, Token: &device.Token})
		if deviceState.Code != "000000" {
			log.Default().Output(1, fmt.Sprintf("code was non-zero: %v for device %v (name: %v) at time %v", deviceState.Code, device.ID, device.Name, utils.Time()))
		}
		if err != nil {
			return fmt.Errorf("error while quering device: %w", err)
		}
		// Process response
		dataMap, err := utils.ToMap[any](deviceState.Data)
		if err != nil {
			return fmt.Errorf("error converting data %v: %w", deviceState.Data, err)
		}
		pairs := utils.TraverseMap(dataMap, []utils.KVPair{}, "")
		fmt.Println(pairs)

		// Ensure neccesary keys exist
		var hasReportAt bool
		for k := range dataMap {
			if k == "reportAt" {
				hasReportAt = true
			}
		}
		if !hasReportAt {
			logger := log.Default()
			logger.Output(1, fmt.Sprintf("reportAt missing for sensor %v (name %v) at time %v", device.ID, device.Name, time.Now()))
			continue
		}

		eventTimestamp, err := time.Parse(time.RFC3339Nano, dataMap["reportAt"].(string))
		if err != nil {
			return fmt.Errorf("error converting time %v to epoch seconds: %w", dataMap["reportAt"], err)
		}
		for _, pair := range pairs {
			err := stores.Events.Add(data.Event{
				EventSourceDeviceID: device.ID,
				RequestDeviceID:     "1",
				ResponseTimestamp:   deviceState.Time,
				EventTimestamp:      eventTimestamp.Unix(),
				FieldName:           pair.K,
				FieldValue:          pair.V,
			})
			if err != nil {
				return fmt.Errorf("error while adding event: %w", err)
			}
		}
	}
	return nil
}
