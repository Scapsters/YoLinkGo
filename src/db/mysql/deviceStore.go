package mysql

import (
	"com/data"
	"com/db"
	"database/sql"
	"fmt"
	"strings"
)

var _ db.DeviceStore = (*MySQLDeviceStore)(nil)

type MySQLDeviceStore struct {
	DB *sql.DB
}

func (store *MySQLDeviceStore) Add(item data.Device) error {
	_, err := store.DB.Exec(
		`
        INSERT INTO devices (
            internal_device_id,
            device_type,
            device_name,
            device_token,
            device_timestamp
        ) VALUES (?, ?, ?, ?, ?)
        `,
		item.ID,
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
	res, err := store.DB.Exec(`DELETE FROM devices WHERE internal_device_id = ?`, device.ID)
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
func (store *MySQLDeviceStore) Get(filter data.DeviceFilter) ([]data.StoreDevice, error) {
	args := []any{}
	conditions := []string{}

	if filter.ID != nil {
		conditions = append(conditions, "internal_device_id = ?")
		args = append(args, *filter.ID)
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

	rows, err := store.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("error querying devices: %w", err)
	}
	defer rows.Close()

	var devices []data.StoreDevice
	for rows.Next() {
		var device data.StoreDevice
		err := rows.Scan(
			&device.ID,
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
            internal_device_id 	VARCHAR(40) NOT NULL,
            device_type 		VARCHAR(45) NOT NULL,
            device_name 		VARCHAR(60) NOT NULL,
            device_token 		VARCHAR(60) NOT NULL,
            device_timestamp 	VARCHAR(45) NOT NULL,
            PRIMARY KEY (internal_device_id)
        ) ENGINE = InnoDB;
    `)
	if err != nil {
		return fmt.Errorf("error creating devices table: %w", err)
	}
	return nil
}
