package postgres

import (
	"github.com/Iowel/app-auth-service/internal/domain"
	"github.com/Iowel/app-auth-service/pkg/pb"

	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ EmailRepositoryI = &emailRepository{}


type EmailRepositoryI interface {
	CreateVerifyEmail(ctx context.Context, arg domain.CreateVerifyEmailParams) (*domain.VerifyEmail, error)
	UpdateVerifyEmail(ctx context.Context, arg domain.UpdateVerifyEmailParams) (*domain.VerifyEmail, error)
	VerifyEmailTx(ctx context.Context, arg domain.VerifyEmailTxParams) (VerifyEmailTxResult, error)
}

type emailRepository struct {
	Db *pgxpool.Pool
}

func NewEmailRepository(db *pgxpool.Pool) EmailRepositoryI {
	return &emailRepository{Db: db}
}

func (e *emailRepository) CreateVerifyEmail(ctx context.Context, arg domain.CreateVerifyEmailParams) (*domain.VerifyEmail, error) {
	const op = "db.CreateVerifyEmail"

	query := `
		INSERT INTO verify_emails (user_id, email, secret_code)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, email, secret_code, is_used, created_at, expired_at;
	`

	row := e.Db.QueryRow(ctx, query, arg.User_id, arg.Email, arg.SecretCode)

	var ve domain.VerifyEmail

	err := row.Scan(
		&ve.ID,
		&ve.User_id,
		&ve.Email,
		&ve.SecretCode,
		&ve.IsUsed,
		&ve.CreatedAt,
		&ve.ExpiredAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, domain.ErrUserNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &ve, nil
}

func (e *emailRepository) UpdateVerifyEmail(ctx context.Context, arg domain.UpdateVerifyEmailParams) (*domain.VerifyEmail, error) {
	const op = "db.UpdateVerifyEmail"

	query := `
		UPDATE verify_emails
		SET is_used = TRUE
		WHERE id = $1
		  AND secret_code = $2
		  AND is_used = FALSE
		  AND expired_at > now()
		RETURNING id, user_id, email, secret_code, is_used, created_at, expired_at;
	`

	row := e.Db.QueryRow(ctx, query, arg.ID, arg.SecretCode)

	var ve domain.VerifyEmail
	err := row.Scan(
		&ve.ID,
		&ve.User_id,
		&ve.Email,
		&ve.SecretCode,
		&ve.IsUsed,
		&ve.CreatedAt,
		&ve.ExpiredAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, domain.ErrUserNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &ve, nil
}

type VerifyEmailTxResult struct {
	VerifyEmail *domain.VerifyEmail
	User        *pb.User
}

func (r *emailRepository) VerifyEmailTx(ctx context.Context, arg domain.VerifyEmailTxParams) (VerifyEmailTxResult, error) {
	const op = "repository.postgres.VerifyEmailTx"

	var result VerifyEmailTxResult

	tx, err := r.Db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return result, fmt.Errorf("%s: begin tx failed: %w", op, err)
	}
	defer tx.Rollback(ctx)

	// Обновляем verify_emails (is_used = true)
	updateVerifyEmailQuery := `
		UPDATE verify_emails
		SET is_used = TRUE
		WHERE id = $1
		  AND secret_code = $2
		  AND is_used = FALSE
		  AND expired_at > now()
		RETURNING id, user_id, email, secret_code, is_used, created_at, expired_at;
	`

	var ve domain.VerifyEmail
	row := tx.QueryRow(ctx, updateVerifyEmailQuery, arg.EmailId, arg.SecretCode)
	err = row.Scan(
		&ve.ID,
		&ve.User_id,
		&ve.Email,
		&ve.SecretCode,
		&ve.IsUsed,
		&ve.CreatedAt,
		&ve.ExpiredAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return result, fmt.Errorf("%s: %w", op, domain.ErrVerifyEmailNotFound)
		}
		return result, fmt.Errorf("%s: %w", op, err)
	}
	result.VerifyEmail = &ve

	// Обновляем пользователя (is_email_verified = true)
	updateUserQuery := `
		UPDATE users
		SET is_email_verified = TRUE
		WHERE id = $1
		RETURNING id, email, name, password, is_email_verified, created_at, updated_at;
	`

	var user pb.User
	var createdAt time.Time
	var updatedAt time.Time

	row = tx.QueryRow(ctx, updateUserQuery, ve.User_id)
	err = row.Scan(
		&user.Id,
		&user.Email,
		&user.Name,
		&user.Password,
		&user.Isemailverified,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return result, fmt.Errorf("%s: failed to update user: %w", op, err)
	}

	user.CreatedAt = timestamppb.New(createdAt)
	user.UpdatedAt = timestamppb.New(updatedAt)
	result.User = &user

	if err := tx.Commit(ctx); err != nil {
		return result, fmt.Errorf("%s: commit failed: %w", op, err)
	}

	return result, nil
}
