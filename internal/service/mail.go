package service

import (
	"context"
	"fmt"

	"github.com/Iowel/app-auth-service/internal/domain"
	"github.com/Iowel/app-auth-service/internal/repository/postgres"
)

var _ IMailService = &mailService{}


type IMailService interface {
	VerifyEmailTx(ctx context.Context, arg domain.VerifyEmailTxParams) (postgres.VerifyEmailTxResult, error)
}

type mailService struct {
	userRepo postgres.UserRepository
	mailRepo postgres.EmailRepositoryI
}

func NewMailService(u postgres.UserRepository, mail postgres.EmailRepositoryI) IMailService {
	return &mailService{
		userRepo: u,
		mailRepo: mail,
	}
}

func (m *mailService) VerifyEmailTx(ctx context.Context, arg domain.VerifyEmailTxParams) (postgres.VerifyEmailTxResult, error) {
	const op = "service.VerifyEmailTx"

	mail, err := m.mailRepo.VerifyEmailTx(ctx, arg)
	if err != nil {
		return mail, fmt.Errorf("%s: %w", op, err)
	}

	return mail, nil
}
