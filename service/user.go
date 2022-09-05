package service

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Username       string
	HashedPassword string
	Role           string
}

func NewUser(username, password, role string) (*User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if err != nil {
		return nil, fmt.Errorf("cannot hash password: %w", err)
	}

	user := &User{
		Username:       username,
		HashedPassword: string(hashedPassword),
		Role:           role,
	}

	return user, nil
}

// IsCorrectPassword checks if the given password is correct or not
func (user *User) IsCorrectPassword(password string) bool {
	// call bcrypt.CompareHashAndPassword() function, 
	// pass in the user’s hashed password, and the given plaintext password.
	err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(password))

	// Then return true if error is nil.
	return err == nil

}

// Clone clones a user to store
func (user *User) Clone() *User {
    return &User{
        Username:       user.Username,
        HashedPassword: user.HashedPassword,
        Role:           user.Role,
    }
}

// CreateUser create a user given its username, password and role, 
// and saves it to the user store.
func CreateUser(userStore UserStore, username, password, role string) error {
    user, err := NewUser(username, password, role)
    if err != nil {
        return err
    }
    return userStore.Save(user)
}

func SeedUsers(userStore UserStore) error {
    err := CreateUser(userStore, "admin1", "secret", "admin")
    if err != nil {
        return err
    }
    return CreateUser(userStore, "user1", "secret", "user")
}



