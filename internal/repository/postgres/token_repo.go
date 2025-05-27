package postgres

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/Iowel/app-auth-service/pkg/pb"

	"github.com/jackc/pgx"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TokenRepository struct {
	Db *pgxpool.Pool
}

func NewTokenRepository(db *pgxpool.Pool) *TokenRepository {
	return &TokenRepository{Db: db}
}

func (m *TokenRepository) InsertToken(t *pb.Token, u *pb.User) error {
	// Контекст с таймаутом 3 секунды
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Удаляем все предыдущие токены для пользователя
	const deleteStmt = `DELETE FROM tokens WHERE user_id = $1`

	if _, err := m.Db.Exec(ctx, deleteStmt, u.Id); err != nil {
		return fmt.Errorf("failed to delete existing tokens: %w", err)
	}

	// Добавляем новый токен
	const insertStmt = `
		INSERT INTO tokens (
			user_id,
			name,
			email,
			token_hash,
			expiry,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := m.Db.Exec(ctx, insertStmt,
		u.Id,
		u.Name,
		u.Email,
		t.Hash,
		t.Expiry.AsTime(),
		time.Now(),
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to insert new token: %w", err)
	}
	return nil
}

func (m *TokenRepository) GetUserForToken(token string) (*pb.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// хншируем токен
	tokenHash := sha256.Sum256([]byte(token))


	var user pb.User

	query := `
		SELECT
			u.id, u.name, u.email, u.role
		FROM
			users u
		INNER JOIN tokens t ON u.id = t.user_id
		WHERE
			t.token_hash = $1 AND t.expiry > $2
	`

	err := m.Db.QueryRow(ctx, query, tokenHash[:], time.Now()).Scan(
		&user.Id,
		&user.Name,
		&user.Email,
		&user.Role,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Println("Token not found or expired")
			return nil, nil
		}
		log.Println("DB error:", err)
		return nil, err
	}

	return &user, nil
}
