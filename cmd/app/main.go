package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	gapi "github.com/Iowel/app-auth-service/internal/delivery"
	"github.com/Iowel/app-auth-service/internal/pkg/worker"
	"github.com/Iowel/app-auth-service/internal/repository/postgres"
	"github.com/Iowel/app-auth-service/internal/service"
	"github.com/Iowel/app-auth-service/pkg/cache"
	"github.com/Iowel/app-auth-service/pkg/configs"
	"github.com/Iowel/app-auth-service/pkg/eventbus"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
)

var interruptSignals = []os.Signal{
	os.Interrupt,
	syscall.SIGTERM,
	syscall.SIGINT,
}

const (
	redisDB = 1
	exp     = 999999
)

func main() {
	cfg := configs.LoadConfig()

	ctx, cancel := signal.NotifyContext(context.Background(), interruptSignals...)
	defer cancel()

	// db
	db, err := pgxpool.New(ctx, cfg.DB.Dsn)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	log.Println("Successfully connected to PostgreSQL")
	defer db.Close()

	eventBus := eventbus.NewEventBus()

	// repository
	userRepo := postgres.NewUserRepo(db)
	tokenRepo := postgres.NewTokenRepository(db)
	mailRepo := postgres.NewEmailRepository(db)
	cacheRepo := cache.NewRedisCache(cfg.Redis.Port, redisDB, exp)
	statRepo := postgres.NewStatRepository(db)

	// service
	authServ := service.NewAuthService(userRepo, tokenRepo, cacheRepo, eventBus)
	mailServ := service.NewMailService(userRepo, mailRepo)
	statServ := service.NewStatService(eventBus, statRepo)

	// redis connection
	redisOpt := asynq.RedisClientOpt{
		Addr: cfg.Redis.Port,
	}
	taskDistributor := worker.NewRedisTaskDistributor(redisOpt)

	waitGroup, ctx := errgroup.WithContext(ctx)

	// servers
	worker.RunTaskProcessor(ctx, waitGroup, cfg, redisOpt, db)
	gapi.RunGrpcServer(ctx, waitGroup, cfg, db, authServ, mailServ, taskDistributor)
	gapi.RunGatewayServer(ctx, waitGroup, authServ, mailServ, cfg, db, taskDistributor)

	statServ.RegisterEvent(ctx, waitGroup)

	err = waitGroup.Wait()
	if err != nil {
		log.Fatalf("error from wait group %v\n", err)
	}

}
