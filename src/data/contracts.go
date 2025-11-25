package data

type HasID struct {
	ID 	string
}
type HasIDGetter interface {
	GetID() string
}

type Spreadable interface {
	// Spread elements, in order.
	Spread() []any
}
type SpreadableAddresses interface {
	// Spread addresses, in order.
	SpreadAddresses() []any
}
type SpreadableForExport interface {
	// Spread elements for export, in order.
	SpreadForExport() []string
}

type HasIDGetterAndSpreadable interface {
	HasIDGetter
	Spreadable
	SpreadableAddresses
	SpreadableForExport
}