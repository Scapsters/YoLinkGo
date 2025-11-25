package mysql

import (
	"com/connections/db"
	"com/data"
	"com/logs"
	"com/utils"
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
)

var _ db.DeviceStore = (*MySQLDeviceStore)(nil)

type MySQLDeviceStore struct {
	DB *sql.DB
}

func (store *MySQLDeviceStore) Add(ctx context.Context, item data.Device) (string, error) {
	return sqlInsertHelper[data.StoreDevice](
		ctx,
		store.DB,
		"devices",
		[]string{"device_id", "brand_device_id", "device_brand", "device_kind", "device_name", "device_token", "device_timestamp"},
		[]any{item.BrandID, item.Brand, item.Kind, item.Name, item.Token, item.Timestamp},
	)
}
func (store *MySQLDeviceStore) Delete(ctx context.Context, device data.StoreDevice) error {
	return sqlDeleteHelper[data.StoreJob](ctx, store.DB, "devices", device.ID)
}
func (store *MySQLDeviceStore) Get(ctx context.Context, filter data.DeviceFilter) *data.IterablePaginatedData[data.StoreDevice] {
	return sqlGetHelper(
		store.DB,
		[]string{"device_id = ?", "brand_device_id = ?", "device_brand = ?", "device_kind = ?", "device_name = ?"},
		[]any{filter.ID, filter.BrandID, filter.Brand, filter.Kind, filter.Name},
		"devices",
		"device_id",
		func(rows *sql.Rows) (*data.StoreDevice, error) {
			var device data.StoreDevice
			err := rows.Scan(&device.ID, &device.BrandID, &device.Brand, &device.Kind, &device.Name, &device.Token, &device.Timestamp)
			if err != nil {
				return nil, fmt.Errorf("error scanning device: %w", err)
			}
			return &device, nil
		},
	)
}
func (store *MySQLDeviceStore) Setup(ctx context.Context, isDestructive bool) error {
	if isDestructive {
		err := dropTable(ctx, store.DB, "devices")
		if err != nil {
			return err
		}
	}
	return sqlCreateTableHelper(ctx, store.DB, `
		CREATE TABLE IF NOT EXISTS devices (
			device_id 			VARCHAR(40) NOT NULL,
			brand_device_id 	VARCHAR(40) NOT NULL,
			device_brand	    VARCHAR(20) NOT NULL,
			device_kind 		VARCHAR(45) NOT NULL,
			device_name 		VARCHAR(60) NOT NULL,
			device_token 		VARCHAR(60) NOT NULL, 
			device_timestamp 	VARCHAR(45) NOT NULL,
			PRIMARY KEY (device_id)
		) ENGINE = InnoDB;
	`, "devices")
}
func (store *MySQLDeviceStore) Edit(ctx context.Context, device data.StoreDevice) error {
	sqlctx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()
	res, err := store.DB.ExecContext(sqlctx, `
        UPDATE devices
        SET
			brand_device_id = ?, device_brand = ?, device_kind = ?, device_name = ?, device_token = ?, device_timestamp = ?
        WHERE device_id = ?
        `,
		device.BrandID, device.Brand, device.Kind, device.Name, device.Token, device.Timestamp, device.ID,
	)
	if err != nil {
		return fmt.Errorf("error updating device %v: %w", device, err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking rows affected for device %v: %w", device, err)
	}
	if rows == 0 {
		logs.WarnWithContext(ctx, "device %v was not found when attempting to update it", device)
		return fmt.Errorf("no device updated with ID %v", device.ID)
	}

	return nil
}
func (store *MySQLDeviceStore) Export(ctx context.Context, filter data.DeviceFilter) error {
	// Get data
	devices := store.Get(ctx, filter)

	// Export data
	return sqlExportHelper(
		ctx,
		devices,
		"devices",
		[]string{"device_id", "brand_device_id", "device_brand", "device_type", "device_name", "device_token", "device_timestamp"},
		func(writer *csv.Writer, device data.StoreDevice) error {
			return writer.Write([]string{
				device.ID,
				device.BrandID,
				device.Brand,
				device.Kind,
				device.Name,
				device.Token,
				utils.EpochSecondsToExcelDate(device.Timestamp),
			})
		},
	)
}
