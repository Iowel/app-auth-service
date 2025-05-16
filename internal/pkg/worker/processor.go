package worker

import (
	"context"
	"log"

	"github.com/Iowel/app-auth-service/internal/pkg/mail"
	"github.com/Iowel/app-auth-service/internal/repository/postgres"
	"github.com/Iowel/app-auth-service/pkg/configs"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
)

// Обработчик задач
// Забирает задачи из очереди Redis и обрабатывает их

// название очередей
const (
	QueueCritical = "critical"
	QueueDefault  = "default"
)

type TaskProcessor interface {
	Start() error
	Shutdown()
	ProcessTaskSendVerifyEmail(ctx context.Context, task *asynq.Task) error
}

type RedisTaskProcessor struct {
	server   *asynq.Server
	userRepo postgres.UserRepository
	mailRepo postgres.EmailRepositoryI
	mailer   mail.EmailSender
}

// Создание нового обработчика задач Redis
func NewRedisTaskProcessor(redisOpt asynq.RedisClientOpt, userRepo postgres.UserRepository, mailRepo postgres.EmailRepositoryI, mailer mail.EmailSender) TaskProcessor {
	server := asynq.NewServer(redisOpt, asynq.Config{
		Queues: map[string]int{
			QueueCritical: 10,
			QueueDefault:  5,
		},

		ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
			log.Printf("Задача не выполнена, ошибка: %s, payload: %s, type: %v", err.Error(), task.Payload(), task.Type())
		}),

		Logger: NewLogger(),
	},
	)

	return &RedisTaskProcessor{
		server:   server,
		userRepo: userRepo,
		mailRepo: mailRepo,
		mailer:   mailer,
	}

}

func (processor *RedisTaskProcessor) Start() error {
	mux := asynq.NewServeMux()

	// Регистрируем задачи
	mux.HandleFunc(TaskSendVerifyEmail, processor.ProcessTaskSendVerifyEmail)

	return processor.server.Start(mux)
}

func (processor *RedisTaskProcessor) Shutdown() {
	processor.server.Shutdown()
}

func RunTaskProcessor(ctx context.Context, waitGroup *errgroup.Group, config *configs.Config, redisOpt asynq.RedisClientOpt, db *pgxpool.Pool) {
	const op = "pkg.worker.RunTaskProcessor"

	mailRepo := postgres.NewEmailRepository(db)
	userRepo := postgres.NewUserRepo(db)

	mailer := mail.NewGmailSender(config.SmtpGmail.SenderName, config.SmtpGmail.SenderAddress, config.SmtpGmail.SenderPassword)

	taskProcessor := NewRedisTaskProcessor(redisOpt, userRepo, mailRepo, mailer)
	log.Println("start task processor")

	err := taskProcessor.Start()
	if err != nil {
		log.Fatalf("cannot launch Task Processor, path: %s, error: %s\n", op, err)
	}

	// прослушиваем сигналы прерывания и корректно уничтожаем сервер
	waitGroup.Go(func() error {
		<-ctx.Done()
		log.Println("graceful shutdown task processor")

		taskProcessor.Shutdown()
		log.Println("task processor successfully stopped")

		return nil
	})
}
