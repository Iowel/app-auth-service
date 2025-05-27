package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/Iowel/app-auth-service/internal/domain"
	"github.com/Iowel/app-auth-service/internal/repository/postgres"
	"github.com/Iowel/app-auth-service/pkg/cache"
	"github.com/Iowel/app-auth-service/pkg/pb"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/metadata"
)

var _ IAuthService = &authService{}

type IAuthService interface {
	Register(email, password, name string) (*pb.User, error)
	Login(email, password string) (*pb.Token, error)
	RegisterTx(ctx context.Context, params domain.CreateUserTxParams) (*domain.CreateUserTxResult, error)
	AuthorizeUser(ctx context.Context) (*pb.User, error)

	CreateProfile(profile *domain.Profile) error
	GetStatusIDByName(ctx context.Context, name string) (int, error)
}

type authService struct {
	userRepo  postgres.UserRepository
	tokenRepo *postgres.TokenRepository
	cache     cache.IPostCache
}

func NewAuthService(u postgres.UserRepository, tokenRepo *postgres.TokenRepository, cache cache.IPostCache) IAuthService {
	return &authService{
		userRepo:  u,
		tokenRepo: tokenRepo,
		cache:     cache,
	}
}

func (a *authService) Register(email, password, name string) (*pb.User, error) {
	const op = "service.auth.Register"

	existedUser, _ := a.userRepo.GetUserByEmail(email)
	if existedUser != nil {
		return nil, domain.ErrUserExists
	}

	// create password
	hashPass, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// create user
	createdUser, err := a.userRepo.CreateUser(email, string(hashPass), name)
	if err != nil {
		return nil, err
	}

	return createdUser, nil
}

func (a *authService) RegisterTx(ctx context.Context, params domain.CreateUserTxParams) (*domain.CreateUserTxResult, error) {
	const op = "service.auth.RegisterTx"

	// проверяем на существование юзера
	existedUser, _ := a.userRepo.GetUserByEmail(params.User.Email)
	if existedUser != nil {
		return nil, domain.ErrUserExists
	}

	// генерим пароль
	hashPass, err := bcrypt.GenerateFromPassword([]byte(params.User.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	params.User.Password = string(hashPass)

	// делаем юзверя
	user, err := a.userRepo.CreateUserTx(ctx, params)
	if err != nil {
		return nil, domain.ErrWrongCredentials
	}

	return &user, nil
}

func (a *authService) Login(email, password string) (*pb.Token, error) {
	const op = "service.auth.Login"

	// проверяем на наличие юзверя
	existUser, err := a.userRepo.GetUserByEmail(email)
	if err != nil {
		log.Printf("GetUserByEmail failed: %s, error: %s", op, err)
		return nil, domain.ErrWrongCredentials
	}

	// проверяем пароль
	err = bcrypt.CompareHashAndPassword([]byte(existUser.Password), []byte(password))
	if err != nil {
		log.Printf("Password comparison failed: path: %s, error: %s", op, err)
		return nil, domain.ErrWrongCredentials
	}

	if !existUser.Isemailverified {
		log.Printf("Isemailverified failed: %s, error: %s", op, err)
		return nil, domain.Isemailverified
	}

	// генерим токен
	token, err := GenerateToken(existUser.Id, 48*time.Hour, ScopeAuthentication)
	if err != nil {
		log.Printf("GenerateToken failed: path: %s, error: %s", op, err)
		return nil, domain.ErrWrongCredentials
	}

	// сохрпняем токен в базе
	err = a.tokenRepo.InsertToken(token, existUser)
	if err != nil {
		log.Printf("InsertToken failed: %s, error: %s", op, err)
		return nil, domain.ErrWrongCredentials
	}

	existProfile, err := a.userRepo.GetProfile(int(existUser.Id))
	if err != nil {
		log.Printf("InsertToken failed: %s, error: %s", op, err)
		return nil, domain.ErrWrongCredentials
	}

	// save to redis cache
	var u = &domain.UserCache{
		ID:       int(existUser.Id),
		Email:    existUser.Email,
		Password: existUser.Password,
		Name:     existUser.Name,
		// IsEmailVerified: existUser.Isemailverified,
		Avatar:    existUser.Avatar,
		Role:      existUser.Role,
		Status:    existProfile.Status,
		Wallet:    &existProfile.Wallet,
		CreatedAt: existUser.CreatedAt.AsTime(),
		UpdatedAt: existUser.UpdatedAt.AsTime(),
	}

	idStr := "user:" + strconv.Itoa(u.ID)
	a.cache.Set(idStr, u)

	return token, nil
}

const (
	authorizationHeader = "authorization"
	authorizationBearer = "bearer"
)

func (a *authService) AuthorizeUser(ctx context.Context) (*pb.User, error) {
	const op = "service.auth.AuthorizeUser"

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("missing metadata")
	}

	values := md.Get(authorizationHeader)
	if len(values) == 0 {
		return nil, fmt.Errorf("missing authorization header")
	}
	authHeader := values[0]
	fields := strings.Fields(authHeader)
	if len(fields) < 2 {
		return nil, fmt.Errorf("invalid authorization header format")
	}

	authType := strings.ToLower(fields[0])
	if authType != authorizationBearer {
		return nil, fmt.Errorf("unsupported authorization type: %s", authType)

	}

	accessToken := fields[1]

	// get the user from tokens table
	user, err := a.tokenRepo.GetUserForToken(accessToken)
	if err != nil {
		log.Printf("GetUserForToken failed: %s, error: %s", op, err)
		return nil, errors.New("No matching user found for the given token")
	}

	return user, nil
}

func (a *authService) CreateProfile(profile *domain.Profile) error {
	const op = "service.auth.CreateProfile"
	err := a.userRepo.CreateProfile(profile)
	if err != nil {
		log.Printf("failed create profile: %s, error: %s", op, err)
		return fmt.Errorf("failed create profile")
	}

	return nil
}

func (a *authService) GetStatusIDByName(ctx context.Context, name string) (int, error) {
	const op = "service.auth.CreateStatusTable"

	status, err := a.userRepo.GetStatusIDByName(ctx, name)
	if err != nil {
		log.Printf("failed to get status: %s, error: %s", op, err)
		return 0, fmt.Errorf("failed to get status")
	}

	return status, nil
}
