package configs

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DB        Dbconfig
	Auth      AuthConfig
	Grpc      Grpc
	Redis     Redis
	Web       WebConfig
	SmtpGmail SmtpGmail
}

type WebConfig struct {
	Port           string
	Api            string
	Dsn            string
	Env            string
	AllowedOrigins string

	Frontend_port string
	Backend_port  string
	ServerAPI     string
}

type Grpc struct {
	Port string
}

type Dbconfig struct {
	Dsn string
}

type AuthConfig struct {
	Secret   string
	TokenTTL string
}

type Redis struct {
	Port string
}

type SmtpGmail struct {
	SenderName     string
	SenderAddress  string
	SenderPassword string
}

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file, using default config")
	}

	return &Config{
		DB:        Dbconfig{Dsn: os.Getenv("DSN")},
		Web:       WebConfig{Port: os.Getenv("HTTP_PORT"), Dsn: os.Getenv("DSN"), Env: os.Getenv("ENV"), AllowedOrigins: os.Getenv("ALLOWED_ORIGINS"), Frontend_port: os.Getenv("FRONTEND_PORT"), Backend_port: os.Getenv("BACKEND_PORT"), ServerAPI: os.Getenv("SERVRER_API")},
		Auth:      AuthConfig{Secret: os.Getenv("SECRET"), TokenTTL: os.Getenv("TOKENTTL")},
		Grpc:      Grpc{Port: os.Getenv("GRPC_SERVER_ADDRESS")},
		Redis:     Redis{Port: os.Getenv("REDIS_PORT")},
		SmtpGmail: SmtpGmail{SenderName: os.Getenv("EMAIL_SENDER_NAME"), SenderAddress: os.Getenv("EMAIL_SENDER_ADDRESS"), SenderPassword: os.Getenv("EMAIL_SENDER_PASSWORD")},
	}
}
