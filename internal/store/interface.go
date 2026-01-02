package store

// Store defines the behavior for subscriber persistence
type Store interface {
	Add(email string) error
	Remove(email string) error
	GetAll() ([]string, error)
}
