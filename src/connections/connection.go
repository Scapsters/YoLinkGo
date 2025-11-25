package connections

import "context"

type PingResult int

const (
	Unknown PingResult = iota
	Good
	Bad
)

func (status PingResult) String() string {
	switch status {
	case Unknown:
		return "Unknown"
	case Good:
		return "Connected"
	case Bad:
		return "Disconnected"
	}
	return "Out of range"
}

type Connection interface {
	// No error implies Status will be Good
	Open(ctx context.Context) error
	// No error implies Status will be Bad
	Close() error
	// Check the status of the connection with no chance of an error being thrown
	// string is a description of the result, usually if Bad
	Status(ctx context.Context) (PingResult, string)
}
