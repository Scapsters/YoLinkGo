package main

import (
	"com/connections/db/mysql"
	"com/connections/sensors"
	"com/data"
	"com/jobs"
	"com/logs"
	"com/utils"
	"context"
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
	ctx = logs.ContextWithLogger(ctx, jobLogger)

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
	err = jobs.StoreAllConnectionSensorData(ctx, dbConnection, yoLinkConnection)
	if err != nil {
		return fmt.Errorf("error while storing sensor data: %w", err)
	}

	// Repeat job for 72h. Currently, this function is blocking
	logs.FDefaultLog("Scheduling starting...")
	err = scheduleJob(
		func() error {
			err = jobs.StoreAllConnectionSensorData(ctx, dbConnection, yoLinkConnection)
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
