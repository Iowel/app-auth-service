package domain

import (
	"time"

	"github.com/Iowel/app-auth-service/pkg/pb"
)

type User struct {
	ID              int       `json:"id"`
	Email           string    `json:"email"`
	Password        string    `json:"password"`
	Name            string    `json:"name"`
	IsEmailVerified bool      `json:"is_email_verified"`
	Avatar          string    `json:"avatar"`
	Role            string    `json:"role"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type UpdateUserParams struct {
	Name            string    `json:"username"`
	Password        string    `json:"password"`
	Email           string    `json:"email"`
	IsEmailVerified bool      `json:"is_email_verified"`
	ID              int64     `json:"id"`
	CreatedAt       time.Time `json:"created_at"`
}

type CreateUserTxParams struct {
	User        *pb.User
	AfterCreate func(*pb.User) error

	// AdditionalData - дополнительные данные или флаги, которые могут понадобиться
	// в процессе обработки. можно использовать для логирования, валидации и прочего
	AdditionalData map[string]interface{} `json:"additional_data,omitempty"`
}

type CreateUserTxResult struct {
	User *pb.User
}
