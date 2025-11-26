package data

type HasID struct {
	ID string
}
type HasIDGetter interface {
	GetID() string
}

type Spreadable interface {
	// Spread elements, in order.
	Spread() []any
}
type SpreadableAddresses[T any] interface {
	// Returns a pointer to a copy of the value, and then its spread addresses, in order.
	SpreadAddresses() (*T, []any)
}
type SpreadableForExport interface {
	// Spread elements for export, in order.
	SpreadForExport() []string
}

type HasIDGetterAndSpreadable[T any] interface {
	HasIDGetter
	Spreadable
	SpreadableAddresses[T]
	SpreadableForExport
}
