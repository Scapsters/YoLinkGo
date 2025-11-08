package db

var _ DeviceStore = (*MySQLStore)(nil)

type MySQLStore struct {
	connection = "Im connected!"
}
func (store MySQLStore) Get(filter data.)