package app

import (
	"context"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           int       `json:"id,omitempty" db:"id"`
	Email        string    `json:"email,omitempty" db:"email"`
	Token        string    `json:"token,omitempty"`
	PasswordHash string    `json:"-" db:"password_hash"`
	CreatedAt    time.Time `json:"-" db:"created_at"`
	UpdatedAt    time.Time `json:"-" db:"updated_at"`
}

func (u *User) User() *User {
	return &User{
		ID:    u.ID,
		Email: u.Email,
	}
}

var AnonymousUser User

type UserFilter struct {
	ID    *uint
	Email *string

	Limit  int
	Offset int
}

type UserPatch struct {
	Email        *string `json:"email"`
	PasswordHash *string `json:"-" db:"password_hash"`
}

func (u *User) SetPassword(password string) error {
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if err != nil {
		// return better error message
		return err
	}

	u.PasswordHash = string(hashBytes)

	return nil
}

func (u User) VerifyPassword(password string) bool {
	log.Printf("Running verify password")
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))

	return err == nil
}

func (u *User) IsAnonymous() bool {
	return u == &AnonymousUser
}

type UserService interface {
	Authenticate(ctx context.Context, email string, password string) (*User, error)

	CreateUser(context.Context, *User) error

	UserByID(context.Context, uint) (*User, error)

	UserByEmail(context.Context, string) (*User, error)

	Users(context.Context, UserFilter) ([]*User, error)

	UpdateUser(context.Context, *User, UserPatch) error

	DeleteUser(context.Context, uint) error
}
