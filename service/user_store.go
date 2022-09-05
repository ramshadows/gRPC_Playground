package service

import "sync"

type UserStore interface {
	Save(user *User) error
	Find(username string) (*User, error)
}

type InMemoryUserStore struct {
	mutex sync.RWMutex
	users map[string]*User
}

func NewInMemoryUserStore() *InMemoryUserStore {
	return &InMemoryUserStore{
		users: make(map[string]*User),
	}
}

func (store *InMemoryUserStore) Save(user *User) error {
	// acquire the write lock.
	store.mutex.Lock()
	defer store.mutex.Unlock()

	// check if a user with the same username already exists or not.
	if store.users[user.Username] != nil {
		// It it does, we return an error.
		return ErrAlreadyExists
	}

	// Otherwise, we just clone the input user and put it into the map.
	store.users[user.Username] = user.Clone()
	return nil
}

func (store *InMemoryUserStore) Find(username string) (*User, error) {
	store.mutex.RLock()
	defer store.mutex.RUnlock()

	user := store.users[username]
	if user == nil {
		return nil, nil
	}

	return user.Clone(), nil
}
