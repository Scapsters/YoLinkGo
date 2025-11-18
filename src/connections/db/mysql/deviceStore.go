package mysql

import (
	"com/connections/db"
	"com/data"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"com/utils"
	"encoding/csv"
	"os"
	"time"
)

var _ db.DeviceStore = (*MySQLDeviceStore)(nil)

type MySQLDeviceStore struct {
	DB *sql.DB
}

func (store *MySQLDeviceStore) Add(item data.Device) error {
	_, err := store.DB.Exec(
		`
        INSERT INTO devices (
            yolink_device_id,
			device_brand,
            device_type,
            device_name,
            device_token,
            device_timestamp
        ) VALUES (?, ?, ?, ?, ?, ?)
        `,
		item.ID,
		item.Brand,
		item.Kind,
		item.Name,
		item.Token, // TODO: token is very yolink specific. device info needs its own denormalized table?
		item.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("error adding device %v: %w", item, err)
	}
	return nil
}
func (store *MySQLDeviceStore) Delete(device data.StoreDevice) error {
	res, err := store.DB.Exec(`DELETE FROM devices WHERE yolink_device_id = ?`, device.ID)
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
func (store *MySQLDeviceStore) Get(filter data.DeviceFilter) (*data.IterablePaginatedData[data.StoreDevice], error) {
	// Build conditions
	args := []any{}
	conditions := []string{}
	if filter.ID != nil {
		conditions = append(conditions, "yolink_device_id = ?")
		args = append(args, *filter.ID)
	}
	if filter.Brand != nil {
		conditions = append(conditions, "device_brand = ?")
		args = append(args, *filter.Brand)
	}
	if filter.Kind != nil {
		conditions = append(conditions, "device_type = ?")
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
	query += "yolink_device_id > ? ORDER BY yolink_device_id LIMIT ?"

	getPage := func (index int) ([]data.StoreDevice, error) {
		rows, err := store.DB.Query(query, append(args, index, data.PAGE_SIZE))
		if err != nil {
			return nil, fmt.Errorf("error querying devices: %w", err)
		}
		defer rows.Close()

		var devices []data.StoreDevice
		for rows.Next() {
			var device data.StoreDevice
			err := rows.Scan(
				&device.ID,
				&device.Brand,
				&device.Kind,
				&device.Name,
				&device.Token,
				&device.Timestamp,
			)
			if err != nil {
				return nil, fmt.Errorf("error scanning device: %w", err)
			}
			devices = append(devices, device)
		}
		return devices, nil
	}

	return &data.IterablePaginatedData[data.StoreDevice]{GetPage: getPage}, nil
}
func (store *MySQLDeviceStore) Setup(isDestructive bool) error {
	if isDestructive {
		if _, err := store.DB.Exec(`SET FOREIGN_KEY_CHECKS = 0`); err != nil {
			return fmt.Errorf("error disabling FK checks: %w", err)
		}
		if _, err := store.DB.Exec(`DROP TABLE IF EXISTS devices`); err != nil {
			return fmt.Errorf("error dropping devices table: %w", err)
		}
		if _, err := store.DB.Exec(`SET FOREIGN_KEY_CHECKS = 1`); err != nil {
			return fmt.Errorf("error enabling FK checks: %w", err)
		}
	}
	_, err := store.DB.Exec(`
        CREATE TABLE IF NOT EXISTS devices (
            yolink_device_id 	VARCHAR(40) NOT NULL,
			device_brand	    VARCHAR(20) NOT NULL,
            device_type 		VARCHAR(45) NOT NULL,
            device_name 		VARCHAR(60) NOT NULL,
            device_token 		VARCHAR(60) NOT NULL,
            device_timestamp 	VARCHAR(45) NOT NULL,
            PRIMARY KEY (yolink_device_id)
        ) ENGINE = InnoDB;
    `)
	if err != nil {
		return fmt.Errorf("error creating devices table: %w", err)
	}
	return nil
}
func (store *MySQLDeviceStore) Edit(device data.StoreDevice) error {
	res, err := store.DB.Exec(
		`
        UPDATE devices
        SET
            device_brand     = ?,
            device_type      = ?,
            device_name      = ?,
            device_token     = ?,
            device_timestamp = ?
        WHERE yolink_device_id = ?
        `,
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
		log.Default().Output(1, fmt.Sprintf("device %v was not found when attempting to update it", device))
		return fmt.Errorf("no device updated with ID %v", device.ID)
	}

	return nil
}
func (store *MySQLDeviceStore) Export(filter data.DeviceFilter) error {

	// Ensure exports directory exists
	if err := os.MkdirAll(db.EXPORT_DIR, 0755); err != nil {
		return fmt.Errorf("error creating export directory: %w", err)
	}

	// Generate filename
	now := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("%s/%s_devices.csv", db.EXPORT_DIR, now)

	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating export file: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Write CSV header
	if err := w.Write([]string{
		"yolink_id",
		"brand", // TODO: change device schema to include normal id
		"type",
		"name",
		"token",
		"timestamp",
	}); err != nil {
		return fmt.Errorf("error writing CSV header: %w", err)
	}

	// Get data
	devices, err := store.Get(filter)
	if err != nil {
		return fmt.Errorf("error getting devices for export with filter %v: %w", filter, err)
	}

	// Write each row
	for {
		device, err := devices.Next()
		if err != nil {
			log.Default().Output(1, fmt.Sprintf("Error while fetching device while exporting: %v", err))
		}
		if device == nil {
			break
		}

		err = w.Write([]string{
			device.ID,
			device.Brand,
			device.Kind,
			device.Name,
			device.Token,
			utils.EpochSecondsToExcelDate(device.Timestamp), // TODO: This isnt right
		})
		if err != nil {
			log.Default().Output(1, fmt.Sprintf("Error while writing csv row with data %v: %v", device, err))
		}
	}

	return nil
}