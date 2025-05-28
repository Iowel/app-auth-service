package service

import (
	"context"
	"log"

	"github.com/Iowel/app-auth-service/internal/domain"
	"github.com/Iowel/app-auth-service/internal/repository/postgres"
	"github.com/Iowel/app-auth-service/pkg/eventbus"
	"golang.org/x/sync/errgroup"
)

type StatService struct {
	EventBus       *eventbus.EventBus
	StatRepository *postgres.StatRepository
}

func NewStatService(e *eventbus.EventBus, s *postgres.StatRepository) *StatService {
	return &StatService{
		EventBus:       e,
		StatRepository: s,
	}
}

func (s *StatService) RegisterEvent(ctx context.Context, waitGroup *errgroup.Group) {
	const op = "service.stat.RegisterEvent"

	waitGroup.Go(func() error {
		sub := s.EventBus.Subscribe()
		for msg := range sub {
			if msg.Type == eventbus.EventType {
				data, ok := msg.Data.(domain.Stat)
				if !ok {
					log.Printf("bad event data: %v, path: %s\n", msg.Data, op)
					continue
				}

				err := s.StatRepository.AddRegisterStat(data.UserID, data.Description)
				if err != nil {
					log.Printf("failed to AddRegisterStat, path: %s, error: %v\n", op, err)
				}
			}
		}
		return nil
	})

	// graceful shutdown
	waitGroup.Go(func() error {
		<-ctx.Done()
		log.Println("graceful shutdown Eventbus")

		s.EventBus.Close()

		log.Println("Eventbus successfully stopped")
		return nil
	})
}
