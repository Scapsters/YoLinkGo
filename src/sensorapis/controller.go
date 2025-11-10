package sensors

type ControllerStatus int

const (
	Unknown ControllerStatus = iota
	Ready
	NotReady
)

func (status ControllerStatus) String() string {
	switch status {
	case Unknown:
		return "Unknown"
	case Ready:
		return "Ready"
	case NotReady:
		return "NotReady"
	}
	return "Out of range"
}

type SensorController interface {
	// No error implies connection status is Ready
	Ready() error
	// No error implies connection status is NotReady
	Unready() error
	Status() ControllerStatus
}
