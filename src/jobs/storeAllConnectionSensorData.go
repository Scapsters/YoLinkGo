package jobs

import (
	"com/connections/db"
	"com/connections/sensors"
	"com/data"
	"com/logs"
	"com/utils"
	"context"
	"errors"
	"fmt"
	"time"
)

func StoreAllConnectionSensorData(ctx context.Context, dbConnection db.DBConnection, sensorConnection sensors.SensorConnection) error {
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