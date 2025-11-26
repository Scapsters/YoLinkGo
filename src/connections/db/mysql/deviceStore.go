package mysql

import (
	"com/connections/db"
	"com/data"
	"database/sql"
)

var _ db.GenericStore[data.Device, data.StoreDevice, data.DeviceFilter] = (*MySQLDeviceStore)(nil)

type MySQLDeviceStore struct {
	MySQLEditableStore[data.Device, data.StoreDevice, data.DeviceFilter]
}

func NewMySQLDeviceStore(db *sql.DB) MySQLDeviceStore {
	return MySQLDeviceStore{
		MySQLEditableStore: MySQLEditableStore[data.Device, data.StoreDevice, data.DeviceFilter]{
			MySQLStore: MySQLStore[data.Device, data.StoreDevice, data.DeviceFilter]{
				db:        db,
				tableName: "devices",
				tableCreationSQL: `
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
				`,
				tableColumns: []string{
					"device_id",
					"brand_device_id",
					"device_brand",
					"device_kind",
					"device_name",
					"device_token",
					"device_timestamp",
				},
				primaryKey: "device_id",
			},
		},
	}
}
