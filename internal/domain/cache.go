package domain

import "time"

type UserCache struct {
	ID              int    `json:"id"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	Name            string `json:"name"`
	Token           string
	Role            string    `json:"role"`
	Avatar          string    `json:"avatar"`
	// IsEmailVerified bool      `json:"is_email_verified"`
	Status          string    `json:"status"`
	Wallet          *int      `json:"wallet"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
