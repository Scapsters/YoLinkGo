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
	"os"
	"strings"
	"time"

	"github.com/samborkent/uuidv7"
)

var _ db.DeviceStore = (*MySQLDeviceStore)(nil)

type MySQLDeviceStore struct {
	DB *sql.DB
}

func (store *MySQLDeviceStore) Add(ctx context.Context, item data.Device) (string, error) {
	context, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()
	id := uuidv7.New().String() // MySQL does not support uuidv7 and is notably slower
	_, err := store.DB.ExecContext(context,
		`
        INSERT INTO devices (
			device_id,
            brand_device_id,
			device_brand,
            device_kind,
            device_name,
            device_token,
            device_timestamp
        ) VALUES (?, ?, ?, ?, ?, ?, ?)
        `,
		id,
		item.BrandID,
		item.Brand,
		item.Kind,
		item.Name,
		item.Token,
		item.Timestamp,
	)
	if err != nil {
		return "", fmt.Errorf("error adding device %v: %w", item, err)
	}
	return id, nil
}
func (store *MySQLDeviceStore) Delete(ctx context.Context, device data.StoreDevice) error {
	sqlctx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()
	res, err := store.DB.ExecContext(sqlctx, `DELETE FROM devices WHERE brand_device_id = ?`, device.ID)
	if err != nil {
		return fmt.Errorf("error deleting device %v: %w", device, err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking rows affected for device %v: %w", device, err)
	}
	if rows == 0 {
		return fmt.Errorf("no device deleted with ID %v", device.ID)
	}
	return nil
}
func (store *MySQLDeviceStore) Get(ctx context.Context, filter data.DeviceFilter) (*data.IterablePaginatedData[data.StoreDevice], error) {
	// Build conditions
	args := []any{}
	conditions := []string{}
	if filter.ID != nil {
		conditions = append(conditions, "device_id = ?")
		args = append(args, *filter.ID)
	}
	if filter.BrandID != nil {
		conditions = append(conditions, "brand_device_id = ?")
		args = append(args, *filter.BrandID)
	}
	if filter.Brand != nil {
		conditions = append(conditions, "device_brand = ?")
		args = append(args, *filter.Brand)
	}
	if filter.Kind != nil {
		conditions = append(conditions, "device_kind = ?")
		args = append(args, *filter.Kind)
	}
	if filter.Name != nil {
		conditions = append(conditions, "device_name = ?")
		args = append(args, *filter.Name)
	}
	query := "SELECT * FROM devices"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Add pagination condition
	if len(conditions) > 0 {
		query += " AND "
	} else {
		query += " WHERE "
	}
	query += "device_id > ? ORDER BY device_id LIMIT ?"

	// Gets the next page of results, starting from lastID (non-inclusive) and returns the id to use next call.
	// On error returns the same id that was given to it
	getPage := func(ctx context.Context, lastID *string) ([]data.StoreDevice, *string, error) {
		var filterID string
		if lastID != nil {
			filterID = *lastID
		}
		sqlctx, cancel := context.WithTimeout(ctx, RequestTimeout)
		defer cancel()
		rows, err := store.DB.QueryContext(sqlctx, query, append(args, filterID, data.PAGE_SIZE)...)
		if err != nil {
			return nil, lastID, fmt.Errorf("error querying devices: %w", err)
		}
		defer logs.LogErrorsWithContext(ctx, rows.Close, fmt.Sprintf("error closing rows for query %v and lastID %v", query, lastID))

		var devices []data.StoreDevice
		for rows.Next() {
			var device data.StoreDevice
			err := rows.Scan(
				&device.ID,
				&device.BrandID,
				&device.Brand,
				&device.Kind,
				&device.Name,
				&device.Token,
				&device.Timestamp,
			)
			if err != nil {
				return nil, lastID, fmt.Errorf("error scanning device: %w", err)
			}
			devices = append(devices, device)
		}
		err = rows.Err()
		if err != nil {
			return nil, nil, fmt.Errorf("error in rows: %w", err)
		}
		if len(devices) == 0 {
			return []data.StoreDevice{}, nil, nil
		}
		lastDevice := devices[len(devices)-1]
		return devices, &lastDevice.ID, nil
	}

	return &data.IterablePaginatedData[data.StoreDevice]{GetPage: getPage}, nil
}
func (store *MySQLDeviceStore) Setup(ctx context.Context, isDestructive bool) error {
	if isDestructive {
		sqlctx, cancel := context.WithTimeout(ctx, RequestTimeout)
		defer cancel()
		_, err := store.DB.ExecContext(sqlctx, `SET FOREIGN_KEY_CHECKS = 0`)
		if err != nil {
			return fmt.Errorf("error disabling FK checks: %w", err)
		}
		sqlctx, cancel = context.WithTimeout(ctx, RequestTimeout)
		defer cancel()
		_, err = store.DB.ExecContext(sqlctx, `DROP TABLE IF EXISTS devices`)
		if err != nil {
			return fmt.Errorf("error dropping devices table: %w", err)
		}
		sqlctx, cancel = context.WithTimeout(ctx, RequestTimeout)
		defer cancel()
		_, err = store.DB.ExecContext(sqlctx, `SET FOREIGN_KEY_CHECKS = 1`)
		if err != nil {
			return fmt.Errorf("error enabling FK checks: %w", err)
		}
	}
	sqlctx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()
	_, err := store.DB.ExecContext(sqlctx, `
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
    `)
	if err != nil {
		return fmt.Errorf("error creating devices table: %w", err)
	}
	return nil
}
func (store *MySQLDeviceStore) Edit(ctx context.Context, device data.StoreDevice) error {
	sqlctx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()
	res, err := store.DB.ExecContext(sqlctx, `
        UPDATE devices
        SET
			brand_device_id  = ?
            device_brand     = ?,
            device_kind      = ?,
            device_name      = ?,
            device_token     = ?,
            device_timestamp = ?
        WHERE device_id = ?
        `,
		device.BrandID,
		device.Brand,
		device.Kind,
		device.Name,
		device.Token,
		device.Timestamp,
		device.ID,
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
	// Ensure exports directory exists
	var OwnerReadWriteExecuteAndOthersReadExecute = 0755
	err := os.MkdirAll(db.EXPORT_DIR, os.FileMode(OwnerReadWriteExecuteAndOthersReadExecute))
	if err != nil {
		return fmt.Errorf("error creating export directory: %w", err)
	}

	// Generate filename
	now := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("%s/%s_devices.csv", db.EXPORT_DIR, now)

	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating export file: %w", err)
	}
	defer logs.LogErrorsWithContext(ctx, f.Close, fmt.Sprintf("error closing file %v", filename))

	w := csv.NewWriter(f)
	defer w.Flush()

	// Write CSV header
	err = w.Write([]string{
		"internal_device_id",
		"brand_device_id",
		"device_brand",
		"device_type",
		"device_name",
		"device_token",
		"device_timestamp",
	})
	if err != nil {
		return fmt.Errorf("error writing CSV header: %w", err)
	}

	// Get data
	devices, err := store.Get(ctx, filter)
	if err != nil {
		return fmt.Errorf("error getting devices for export with filter %v: %w", filter, err)
	}

	// Write each row
	for {
		device, err := devices.Next(ctx)
		if err != nil {
			logs.ErrorWithContext(ctx, "Error while fetching device while exporting: %v", err)
		}
		if device == nil {
			break
		}

		err = w.Write([]string{
			device.ID,
			device.BrandID,
			device.Brand,
			device.Kind,
			device.Name,
			device.Token,
			utils.EpochSecondsToExcelDate(device.Timestamp),
		})
		if err != nil {
			logs.ErrorWithContext(ctx, "Error while writing csv row with data %v: %v", device, err)
		}
	}

	return nil
}
