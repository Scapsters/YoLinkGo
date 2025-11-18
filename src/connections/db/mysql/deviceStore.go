package mysql

import (
	"com/connections/db"
	"com/data"
	"com/utils"
	"database/sql"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/samborkent/uuidv7"
)

var _ db.DeviceStore = (*MySQLDeviceStore)(nil)

type MySQLDeviceStore struct {
	DB *sql.DB
}

func (store *MySQLDeviceStore) Add(item data.Device) error {
	_, err := store.DB.Exec(
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
		uuidv7.New().String(), // MySQL does not support uuidv7 and is notably slower
		item.BrandID,
		item.Brand,
		item.Kind,
		item.Name,
		item.Token,
		item.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("error adding device %v: %w", item, err)
	}
	return nil
}
func (store *MySQLDeviceStore) Delete(device data.StoreDevice) error {
	res, err := store.DB.Exec(`DELETE FROM devices WHERE brand_device_id = ?`, device.ID)
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
		conditions = append(conditions, "device_id = ?")
		args = append(args, *filter.ID)
	}
	if filter.BrandID != nil {
		conditions = append(conditions, "brand_device_id = ?")
		args = append(args, *filter.ID)
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
	getPage := func(lastID *string) ([]data.StoreDevice, *string, error) {
		var filterID string
		if lastID != nil {
			filterID = *lastID
		}
		rows, err := store.DB.Query(query, append(args, filterID, data.PAGE_SIZE)...)
		if err != nil {
			return nil, lastID, fmt.Errorf("error querying devices: %w", err)
		}
		defer rows.Close()

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
		if len(devices) == 0 {
			return []data.StoreDevice{}, nil, nil
		}
		lastDevice := devices[len(devices)-1]
		return devices, &lastDevice.ID, nil
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
	} // TODO: token (and others) is very yolink specific. device info needs its own denormalized table?
	_, err := store.DB.Exec(`
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
func (store *MySQLDeviceStore) Edit(device data.StoreDevice) error {
	res, err := store.DB.Exec(`
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
		"internal_device_id",
		"brand_device_id",
		"device_brand",
		"device_type",
		"device_name",
		"device_token",
		"device_timestamp",
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
			device.BrandID,
			device.Brand,
			device.Kind,
			device.Name,
			device.Token,
			utils.EpochSecondsToExcelDate(device.Timestamp),
		})
		if err != nil {
			log.Default().Output(1, fmt.Sprintf("Error while writing csv row with data %v: %v", device, err))
		}
	}

	return nil
}
