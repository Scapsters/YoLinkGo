package data

type HasID struct {
	HasIDGetter
	
	ID 	string
}
type HasIDGetter interface {
	GetID() string
}

