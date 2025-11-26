package jobs

import (
	"com/connections/db"
	"com/connections/sensors"
	"com/data"
	"com/logs"
	"com/utils"
	"context"
	"fmt"
)

func StoreAllConnectionSensorData(ctx context.Context, dbConnection db.DBConnection, sensorConnection sensors.SensorConnection) error {
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
			logs.ErrorWithContext(ctx, "error getting events from device %v: %v", device, err)
			continue
		}

		// Store device data
		for _, event := range events {
			_, err = utils.Retry2(3, func() (string, error) {
				return dbConnection.Events().Add(ctx, event)
			}, nil)
			if err != nil {
				logs.ErrorWithContext(ctx, "error adding event to DB %v: %v", event, err)
				continue
			}
		}
	}
	return nil
}
