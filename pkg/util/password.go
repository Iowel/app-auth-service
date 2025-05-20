package util

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// вычисление хеш строки пароля
func HashPassword(pasword string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(pasword), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedPassword), nil
}

// проверка совпадает ли введенный пароль с предоставленным хеш-паролем
func CheckPassword(password, hashedPassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
