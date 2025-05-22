package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/Iowel/app-auth-service/internal/domain"
	"github.com/Iowel/app-auth-service/pkg/util"

	"github.com/hibiken/asynq"
)

// структура содержащая все данные задачи которые мы хотим сохранить в Redis, позже worker сможет извлечь их из очереди
type PayloadSendVerifyEmail struct {
	Name string `json:"name"`
}

// даем asynq понять какую задачу нужно выполнить или обработать
const (
	TaskSendVerifyEmail = "task:send_verify_email"
)

func (distributor *RedisTaskDistributor) DistributeTaskSendVerifyEmail(ctx context.Context, payload *PayloadSendVerifyEmail, opts ...asynq.Option) error {
	// Тут создаем новую задачу

	// десириализуем payload объект в json
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal task payload: %w", err)
	}

	// создаем задачу чтобы отправить её в очередь
	task := asynq.NewTask(TaskSendVerifyEmail, jsonPayload, opts...)

	// ставим задачу в очередь
	info, err := distributor.client.EnqueueContext(ctx, task)
	if err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	log.Printf("Получена новая задача - отправка письма. %v, payload: %s, queue: %v, max_retry: %v", task.Type(), task.Payload(), info.Queue, info.MaxRetry)
	return nil
}

// Данная функция будет вызвана воркером, когда появится задача типа send:verify_email
func (processor *RedisTaskProcessor) ProcessTaskSendVerifyEmail(ctx context.Context, task *asynq.Task) error {
	var payload PayloadSendVerifyEmail

	// парсим payload из задачи
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", asynq.SkipRetry)
	}

	// извлекаем запись юзверя из базы
	user, err := processor.userRepo.GetUserByName(ctx, payload.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("user doesn't exist: %w", asynq.SkipRetry)
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	// на данном этапе отправляем юзверю письмо
	verifyEmail, err := processor.mailRepo.CreateVerifyEmail(ctx, domain.CreateVerifyEmailParams{
		User_id:    user.Id,
		Email:      user.Email,
		SecretCode: util.RandomString(32),
	})
	if err != nil {
		return fmt.Errorf("failed to create verify email: %w", err)
	}

	// отправляем письмо
	subject := "Hello"
	verifyURL := fmt.Sprintf("http://158.160.74.150:8082/verify_email?email_id=%d&secret_code=%s", verifyEmail.ID, verifyEmail.SecretCode)

	// для теста на локальной машине (TODO: вынести в конфиг)
	// verifyURL := fmt.Sprintf("http://localhost:8082/verify_email?email_id=%d&secret_code=%s", verifyEmail.ID, verifyEmail.SecretCode)

	log.Printf("verifyURL %s\n", verifyURL)

	content := fmt.Sprintf(`Hellooo %s,<br/>
		Hello<br/>
		Click <a href="%s">HERE</a> to verify you account.<br/>
	`, user.Name, verifyURL)

	to := []string{user.Email}

	err = processor.mailer.Sendmail(subject, content, to, nil, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to send verify email: %w", err)
	}

	log.Printf("Задача успешно выполнена, письмо отправлено. type: %v, payload: %s, email: %v", task.Type(), task.Payload(), user.Email)
	return nil
}
