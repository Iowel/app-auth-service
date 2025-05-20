package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Iowel/app-auth-service/internal/domain"
	"github.com/Iowel/app-auth-service/pkg/pb"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ UserRepository = &userRepo{}

type UserRepository interface {
	CreateUser(email, password, name string) (*pb.User, error)
	GetUserByEmail(email string) (*pb.User, error)
	CreateUserTx(ctx context.Context, arg domain.CreateUserTxParams) (domain.CreateUserTxResult, error)
	GetUserByName(ctx context.Context, name string) (*pb.User, error)

	CreateProfile(profile *domain.Profile) error
	GetProfile(id int) (*domain.Profile, error)

	GetStatusIDByName(ctx context.Context, name string) (int, error)
}

type userRepo struct {
	db *pgxpool.Pool
}

func NewUserRepo(db *pgxpool.Pool) UserRepository {
	return &userRepo{
		db: db,
	}
}

func (r *userRepo) CreateUser(email, password, name string) (*pb.User, error) {
	query := `
		INSERT INTO users (email, password, name)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`

	user := &pb.User{}
	var createdAt time.Time
	var updatedAt time.Time

	err := r.db.QueryRow(context.Background(), query, email, password, name).Scan(
		&user.Id,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %v", err)
	}

	user.Email = email
	user.Name = name
	user.Password = password
	user.CreatedAt = timestamppb.New(createdAt)
	user.UpdatedAt = timestamppb.New(updatedAt)

	return user, nil
}

func (u *userRepo) GetUserByEmail(email string) (*pb.User, error) {
	query := `SELECT
		id, email, name, password, role, avatar, created_at, updated_at, is_email_verified
		FROM
			users
		WHERE
			email = $1
		`

	var user pb.User
	var createdAt time.Time
	var updatedAt time.Time

	err := u.db.QueryRow(context.Background(), query, email).Scan(
		&user.Id,
		&user.Email,
		&user.Name,
		&user.Password,
		&user.Role,
		&user.Avatar,
		&createdAt,
		&updatedAt,
		&user.Isemailverified,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to fetch user by email: %w", err)
	}

	// клнвертируем время в timestamppb.Timestamp
	user.CreatedAt = timestamppb.New(createdAt)
	user.UpdatedAt = timestamppb.New(updatedAt)

	return &user, nil
}

func (r *userRepo) CreateUserTx(ctx context.Context, arg domain.CreateUserTxParams) (domain.CreateUserTxResult, error) {
	const op = "repository.postgres.CreateUserTx"

	var result domain.CreateUserTxResult

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
    INSERT INTO users (email, password, name)
    VALUES ($1, $2, $3)
    RETURNING id, role, avatar, is_email_verified, created_at, updated_at 
`

	var user pb.User
	var createdAt, updatedAt time.Time

	err = tx.QueryRow(ctx, query, arg.User.Email, arg.User.Password, arg.User.Name).Scan(
		&user.Id,
		&user.Role,
		&user.Avatar,
		&user.Isemailverified,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				return result, fmt.Errorf("%s: %w", op, domain.ErrUserExists)
			}
		}
		return result, fmt.Errorf("%s: %w", op, err)
	}

	user.Email = arg.User.Email
	user.Name = arg.User.Name
	user.Password = arg.User.Password
	user.CreatedAt = timestamppb.New(createdAt)
	user.UpdatedAt = timestamppb.New(updatedAt)

	// вызов через колбэк пользовательской логики после создания юзверя
	if arg.AfterCreate != nil {
		if err := arg.AfterCreate(&user); err != nil {
			return result, fmt.Errorf("after create callback failed: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return result, fmt.Errorf("failed to commit transaction: %w", err)
	}

	result.User = &user
	return result, nil
}

func (u *userRepo) GetUserByName(ctx context.Context, name string) (*pb.User, error) {
	const op = "storage.postgres.GetUserByName"

	query := `
	SELECT
		id, email, name, password, created_at
	FROM
		users
	WHERE
		name = $1
	`

	row := u.db.QueryRow(ctx, query, name)

	var user pb.User
	var createdAt time.Time

	err := row.Scan(
		&user.Id,
		&user.Email,
		&user.Name,
		&user.Password,
		&createdAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, domain.ErrUserNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	user.CreatedAt = timestamppb.New(createdAt)

	return &user, nil
}

func (r *userRepo) CreateProfile(profile *domain.Profile) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
        INSERT INTO profiles (user_id, avatar, about, friends, wallet, status, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    `
	_, err := r.db.Exec(ctx, query, profile.UserID, profile.Avatar, profile.About, profile.Friends, profile.Wallet, profile.Status, profile.CreatedAt, profile.UpdatedAt)
	if err != nil {
		return fmt.Errorf("profile create failed: %w", err)
	}

	return nil
}

func (r *userRepo) GetStatusIDByName(ctx context.Context, name string) (int, error) {
	var id int
	err := r.db.QueryRow(ctx, "SELECT id FROM statuses WHERE name = $1", name).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *userRepo) GetProfile(id int) (*domain.Profile, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var profile domain.Profile

	err := r.db.QueryRow(ctx, "SELECT id, status, wallet FROM profiles WHERE user_id = $1", id).Scan(
		&profile.ID,
		&profile.Status,
		&profile.Wallet,
	)
	if err != nil {
		return nil, err
	}
	return &profile, nil
}
