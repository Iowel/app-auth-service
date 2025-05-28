package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type StatRepository struct {
	Db *pgxpool.Pool
}

func NewStatRepository(db *pgxpool.Pool) *StatRepository {
	return &StatRepository{Db: db}
}

func (repo *StatRepository) AddRegisterStat(userID int64, description string) error {
	const op = "repository.postgres.AddRegisterStat"

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		INSERT INTO user_stat (user_id, event_description)
		VALUES ($1, $2)
	`

	_, err := repo.Db.Exec(ctx, query, userID, description)
	if err != nil {
		return fmt.Errorf("failed to insert in user_stat: path: %s, %v", op, err)
	}

	return nil
}
