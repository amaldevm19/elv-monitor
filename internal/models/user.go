package models

import "time"

type UserRole string

const (
	RoleAdmin UserRole = "admin"
	RoleUser  UserRole = "user"
)

type User struct {
	Id           string    `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	PasswordHash string    `json:"_" db:"pasword_hash"`
	Role         UserRole  `json:"role" db:"role"`
	CreatedAt    time.Time `json:"createdAt" db:"created_at"`
	LastLoginAt  time.Time `json:"lastloginat,omitempty" db:"last_logn_at"`
}

type UserDTO struct {
	Id       string   `json:"id"`
	Username string   `json:"username"`
	Role     UserRole `json:"role"`
}

func (u User) ToDTO() UserDTO {
	return UserDTO{
		Id:       u.Id,
		Username: u.Username,
		Role:     u.Role,
	}
}
