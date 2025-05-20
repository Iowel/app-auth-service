package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"time"

	"github.com/Iowel/app-auth-service/pkg/pb"

	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	ScopeAuthentication = "authentication"
)

// генерируем токен с заданным сроком действия
func GenerateToken(userID int64, ttl time.Duration, scope string) (*pb.Token, error) {
	token := &pb.Token{
		Userid: int64(userID),
		Expiry: timestamppb.New(time.Now().Add(ttl)),
		Scope:  scope,
	}

	// присваиваем токену случайные байты для уникальности токена
	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}

	// генерируем токен который отправим пользователю
	token.Plaintext = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)

	// генерируем хеш
	hash := sha256.Sum256(([]byte(token.Plaintext)))
	token.Hash = hash[:]
	return token, nil
}
