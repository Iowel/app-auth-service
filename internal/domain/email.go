package domain

import "time"

type VerifyEmail struct {
	ID         int64     `json:"id"`
	User_id    int64     `json:"user_id"`
	Email      string    `json:"email"`
	SecretCode string    `json:"secret_code"`
	IsUsed     bool      `json:"is_used"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiredAt  time.Time `json:"expired_at"`
}

type VerifyEmailTxParams struct {
	EmailId    int64  `json:"emailId"`
	SecretCode string `json:"secretCode"`
}

type VerifyEmailTxResult struct {
	User        *User
	VerifyEmail *VerifyEmail
}

type CreateVerifyEmailParams struct {
	User_id    int64  `json:"user_id"`
	Email      string `json:"email"`
	SecretCode string `json:"secret_code"`
}

type UpdateVerifyEmailParams struct {
	ID         int64  `json:"id"`
	SecretCode string `json:"secret_code"`
}

