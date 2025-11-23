package main

import (
	"com/connections/db"
	"com/connections/db/mysql"
	"com/connections/sensors"
	"com/data"
	"com/logs"
	"com/utils"
	"context"
	"errors"
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
	err = run(context.Background())
	if err != nil {
		log.Fatal("fatal:", err)
	}
}
func run(ctx context.Context) error {
	// Connect to DB
	dbConnection, err := mysql.NewMySQLConnection(ctx, strings.TrimSpace(os.Getenv("MYSQL_CONNECTION_STRING")), false)
	if err != nil {
		return fmt.Errorf("error connecting to DB: %w", err)
	}
	defer logs.LogErrorsWithContext(ctx, dbConnection.Close, fmt.Sprintf("error closing db connection %v", dbConnection))

	// Create job
	jobLogger, err := logs.CreateJob(ctx, dbConnection, logs.Main)
	if err != nil {
		return fmt.Errorf("error while creating job: %w", err)
	}
	ctx = logs.WithLogger(ctx, jobLogger)

	// Connect to YoLink
	yoLinkConnection, err := utils.Retry2(3, func() (*sensors.YoLinkConnection, error) {
		return sensors.NewYoLinkConnection(ctx,
			strings.TrimSpace(os.Getenv("YOLINK_UAID")),
			strings.TrimSpace(os.Getenv("YOLINK_SECRET_KEY")),
		)
	}, nil)
	if err != nil {
		return fmt.Errorf("error while creating new YoLink connection: %w", err)
	}
	err = yoLinkConnection.UpdateManagedDevices(ctx, dbConnection)
	if err != nil {
		return fmt.Errorf("error while updating YoLink device data: %w", err)
	}

	// Store sensor data
	jobLogger.Info(ctx, "Initial run starting...")
	err = storeAllConnectionSensorData(ctx, dbConnection, yoLinkConnection)
	if err != nil {
		return fmt.Errorf("error while storing sensor data: %w", err)
	}

	// Repeat job for 72h. Currently, this function is blocking
	logs.FDefaultLog("Scheduling starting...")
	err = scheduleJob(
		func() error {
			err = storeAllConnectionSensorData(ctx, dbConnection, yoLinkConnection)
			if err != nil {
				return fmt.Errorf("error while storing sensor data: %w", err)
			}
			return nil
		},
		20*time.Minute,
	)
	if err != nil {
		return fmt.Errorf("error scheduling job: %w", err)
	}

	// Export
	err = utils.Retry1(3, func() error {
		return dbConnection.Events().Export(ctx, data.EventFilter{})
	}, nil)
	if err != nil {
		return fmt.Errorf("error exporting: %w", err)
	}

	return nil
}

func storeAllConnectionSensorData(ctx context.Context, dbConnection db.DBConnection, sensorConnection sensors.SensorConnection) error {
	logger, err := logs.Logger(ctx).CreateChildJob(ctx, logs.Import)
	if err != nil {
		return errors.New("error while creating logger while storing sensor data")
	}

	// Get all devices
	devices, err := utils.Retry2(3, func() (*data.IterablePaginatedData[data.StoreDevice], error) {
		return sensorConnection.GetManagedDevices(ctx, dbConnection)
	}, nil)
	if err != nil {
		return fmt.Errorf("error while searching for devices: %w", err)
	}

	for {
		device, err := devices.Next(ctx)
		if err != nil {
			return fmt.Errorf("error getting next item: %w", err)
		}
		if device == nil {
			break
		}

		// Get device data
		events, err := utils.Retry2(3, func() ([]data.Event, error) {
			return sensorConnection.GetDeviceState(ctx, device)
		}, []any{sensors.ErrYoLinkAPIError})
		if err != nil {
			logger.Error(ctx, "error getting events from device %v: %v", device, err)
		}
		time.Sleep(10 * time.Second) //TODO: better than this
		// Store device data
		for _, event := range events {
			_, err = utils.Retry2(3, func() (string, error) {
				return dbConnection.Events().Add(ctx, event)
			}, nil)
			if err != nil {
				logger.Error(ctx, "error adding event to DB %v: %v", event, err)
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
