package gapi

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/Iowel/app-auth-service/internal/domain"
	"github.com/Iowel/app-auth-service/internal/pkg/worker"
	pb "github.com/Iowel/app-auth-service/pkg/pb"

	"github.com/hibiken/asynq"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Server) RegisterUser(ctx context.Context, req *pb.RegisterUserRequest) (*pb.RegisterResponsePayload, error) {
	const op = "delivery.handlers.RegisterUser"

	params := domain.CreateUserTxParams{
		User: &pb.User{Email: req.Email, Password: req.Password, Name: req.Name},

		AfterCreate: func(user *pb.User) error {
			taskPayload := &worker.PayloadSendVerifyEmail{
				Name: user.Name,
			}

			opts := []asynq.Option{
				asynq.MaxRetry(10),
				asynq.ProcessIn(5 * time.Second),
				asynq.Queue(worker.QueueCritical),
			}

			return h.taskDistributor.DistributeTaskSendVerifyEmail(ctx, taskPayload, opts...)
		},
	}

	user, err := h.authService.RegisterTx(ctx, params)
	if err != nil {
		if errors.Is(err, domain.ErrUserExists) {
			return &pb.RegisterResponsePayload{
				Error:   true,
				Message: fmt.Sprintf("Пользователь уже существует"),
			}, nil
		}
		log.Printf("User creation failed: path: %s, error: %v", op, err)
		return &pb.RegisterResponsePayload{
			Error:   true,
			Message: fmt.Sprintf("Что то пошло не так..."),
		}, nil
	}

	log.Printf("user %v\n", user)

	// Создание профиля
	// TODO: ВЫНЕСТИ ОТСЮДА ЕТОТ УЖАС
	profile := &domain.Profile{
		UserID:    int(user.User.Id),
		Avatar:    "static/fox-icon.png",
		About:     "",
		Friends:   []int{},
		Wallet:    0,
		Status:    "Серебренный",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = h.authService.CreateProfile(profile)
	if err != nil {
		return &pb.RegisterResponsePayload{
			Error:   true,
			Message: fmt.Sprintf("failed create profile: %s", err.Error()),
		}, nil
	}

	// респонс
	return &pb.RegisterResponsePayload{
		Error:   false,
		Message: "Register success!",
	}, nil
}

func (h *Server) LoginUser(ctx context.Context, req *pb.LoginUserRequest) (*pb.LoginResponsePayload, error) {
	token, err := h.authService.Login(req.Email, req.Password)
	if err != nil {
		switch {

		case errors.Is(err, domain.ErrWrongCredentials), errors.Is(err, domain.ErrUserNotFound):
			return &pb.LoginResponsePayload{
				Error:   true,
				Message: "неверный email или пароль",
			}, nil

		case errors.Is(err, domain.Isemailverified):
			return &pb.LoginResponsePayload{
				Error:   true,
				Message: "Email не подтвержден",
			}, nil

		default:
			return nil, status.Errorf(codes.Internal, "ошибка входа: %v", err)
		}
	}

	return &pb.LoginResponsePayload{
		Error:   false,
		Message: "success",
		Token:   token,
	}, nil
}

func (h *Server) VerifyEmail(ctx context.Context, req *pb.VerifyEmailRequest) (*pb.VerifyEmailResponse, error) {
	const op = "delivery.VerifyEmail"

	txResult, err := h.mailService.VerifyEmailTx(ctx, domain.VerifyEmailTxParams{
		EmailId:    req.GetEmailId(),
		SecretCode: req.GetSecretCode(),
	})
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "failed to verify email: path: %s, error: %s", op, err)
	}

	resp := &pb.VerifyEmailResponse{
		IsVerified: txResult.User.Isemailverified,
	}

	return resp, nil
}

func (h *Server) VerifyToken(ctx context.Context, req *pb.VerifyTokenRequest) (*pb.VerifyTokenResponse, error) {
	const op = "delivery.VerifyToken"

	user, err := h.authService.AuthorizeUser(ctx)
	if err != nil {
		return &pb.VerifyTokenResponse{
			Error:   true,
			Message: "failed to verify token",
		}, err
	}

	resp := &pb.VerifyTokenResponse{
		Error:   false,
		Message: fmt.Sprintf("authenticated user %s", user.Email),
	}

	return resp, nil
}

func (h *Server) VerifyRole(ctx context.Context, req *pb.VerifyRoleRequest) (*pb.VerifyRoleResponse, error) {
	const op = "delivery.VerifyToken"

	user, err := h.authService.AuthorizeUser(ctx)
	if err != nil {
		return &pb.VerifyRoleResponse{
			Error:   true,
			Message: "failed to verify token",
		}, nil
	}

	if user.Role != "admin" && user.Role != "moderator" {
		return &pb.VerifyRoleResponse{
			Error:   true,
			Message: "failed to verify role",
		}, nil
	}

	resp := &pb.VerifyRoleResponse{
		Error:   false,
		Message: fmt.Sprintf("authenticated user with role %s", user.Role),
	}

	return resp, nil
}
