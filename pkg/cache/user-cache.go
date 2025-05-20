package cache

import (
	"github.com/Iowel/app-auth-service/internal/domain"
)

type IPostCache interface {
	Set(key string, value *domain.UserCache)
	Get(key string) *domain.UserCache
	GetAll() []*domain.UserCache
	Delete(key string)
}
