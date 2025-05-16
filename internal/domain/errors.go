package domain

import "errors"

const (
	ErrGetAllUsers = "Error to get all users"
)

var (
	ErrWrongCredentials = errors.New("wrong email or password")
	ErrUserNotFound     = errors.New("user not found")
	ErrUserExists       = errors.New("user already exists")
	ErrQueryFailed      = errors.New("query failed")

	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidAppID       = errors.New("invalid app_id")

	ErrVerifyEmailNotFound = errors.New("verification record not found or expired")

	Isemailverified = errors.New("Email не подтвержден")
)
