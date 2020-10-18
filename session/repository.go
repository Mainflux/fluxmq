package session

import "sync"

type Repository struct {
	Sessions map[string]Session
	Mu       sync.RWMutex
}

// NewRepository creates a new SessionRepo.
func NewRepository() *Repository {
	return &Repository{
		Sessions: make(map[string]Session),
	}
}

func (r *Repository) Session(clientID string) (Session, bool) {
	val, ok := r.Sessions[clientID]
	return val, ok
}
