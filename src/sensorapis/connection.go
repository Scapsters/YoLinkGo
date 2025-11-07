package sensors

type StoreConnectionStatus int
const (
	Unknown StoreConnectionStatus = iota
	Ready
	NotReady
)

type ConnectionManager interface {
	// No error implies connection status is Ready
	Ready(connectionString string) error
	// No error implies connection status is NotReady
	Unready() error
	Status() StoreConnectionStatus
}