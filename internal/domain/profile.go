package domain

import "time"

type Profile struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Avatar    string    `json:"avatar"`
	About     string    `json:"about"`
	Friends   []int     `json:"friends"`
	Wallet    int       `json:"wallet"`
	Status    string       `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
