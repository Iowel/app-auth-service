package gapi

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/Iowel/app-auth-service/internal/pkg/worker"
	"github.com/Iowel/app-auth-service/internal/service"
	"github.com/Iowel/app-auth-service/pkg/configs"
	pb "github.com/Iowel/app-auth-service/pkg/pb"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/cors"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"
)

type Server struct {
	pb.UnimplementedAuthServiceServer
	cfg             *configs.Config
	db              *pgxpool.Pool
	authService     service.IAuthService
	mailService     service.IMailService
	taskDistributor worker.TaskDistributor
}

func NewServer(cfg *configs.Config, db *pgxpool.Pool, authService service.IAuthService, mailService service.IMailService, taskDistrib worker.TaskDistributor) (*Server, error) {
	server := &Server{
		cfg:             cfg,
		db:              db,
		authService:     authService,
		mailService:     mailService,
		taskDistributor: taskDistrib,
	}
	return server, nil
}

func RunGrpcServer(ctx context.Context, waitGroup *errgroup.Group, cfg *configs.Config, db *pgxpool.Pool, authService service.IAuthService, mailService service.IMailService, taskDistrib worker.TaskDistributor) {
	const op = "delivery.server.RunGrpcServer"

	server, err := NewServer(cfg, db, authService, mailService, taskDistrib)
	if err != nil {
		log.Fatal("cannot create server")
	}

	grpcLogger := grpc.UnaryInterceptor(GrpcLogger)

	grpcServer := grpc.NewServer(grpcLogger)
	pb.RegisterAuthServiceServer(grpcServer, server)
	reflection.Register(grpcServer)

	listener, err := net.Listen("tcp", cfg.Grpc.Port)
	if err != nil {
		log.Fatalf("cannot launch gRPC server, path: %s, error: %v\n", op, err)
	}

	// запускаем сервер в отдельной горутине
	waitGroup.Go(func() error {
		log.Printf("Starting GRPC server on port %s\n", cfg.Grpc.Port)

		if serveErr := grpcServer.Serve(listener); serveErr != nil {
			if errors.Is(serveErr, grpc.ErrServerStopped) {
				return nil
			}
			log.Printf("gRPC server failed to serve, path: %s, error: %v\n", op, serveErr)
			return serveErr
		}

		return nil
	})

	// graceful shutdown
	waitGroup.Go(func() error {
		<-ctx.Done()
		log.Println("graceful shutdown gRPC server")

		grpcServer.GracefulStop()
		log.Println("gRPC server is stopped")

		return nil
	})
}

func RunGatewayServer(ctx context.Context, waitGroup *errgroup.Group, authService service.IAuthService, mailService service.IMailService, cfg *configs.Config, db *pgxpool.Pool, taskDistrib worker.TaskDistributor) {
	const op = "delivery.server.RunGatewayServer"

	server, err := NewServer(cfg, db, authService, mailService, taskDistrib)
	if err != nil {
		log.Fatalf("cannot create server, path: %s, error: %v\n", op, err)
	}

	jsonOption := runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
		MarshalOptions: protojson.MarshalOptions{
			UseProtoNames: true,
		},
		UnmarshalOptions: protojson.UnmarshalOptions{
			DiscardUnknown: true,
		},
	})

	grpcMux := runtime.NewServeMux(jsonOption)

	// регистрируем gRPC gateway хендлер
	err = pb.RegisterAuthServiceHandlerServer(ctx, grpcMux, server)
	if err != nil {
		log.Fatalf("cannot register gRPC gateway handler, path: %s, error: %v\n", op, err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", grpcMux)

	// TODO: вынести в конфиг
	allowedOrigins := []string{
		"http://localhost:8081",
		"http://localhost:8082",
		"http://0.0.0.0:8082",
		"http://0.0.0.0:8081",
		fmt.Sprintf("%s:8082", cfg.Web.ServerAPI),
		fmt.Sprintf("%s:8081", cfg.Web.ServerAPI),
	}

	c := cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowCredentials: true,
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
	})

	handler := c.Handler(HttpLogger(mux))

	httpServer := &http.Server{
		Handler:      handler,
		Addr:         cfg.Web.Port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	// graceful shutdown
	waitGroup.Go(func() error {
		log.Printf("start HTTP gateway server at %s\n", httpServer.Addr)

		err = httpServer.ListenAndServe()
		if err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				return nil
			}

			log.Printf("HTTP gateway server failed to serve, path: %s, error: %v\n", op, err)
			return err
		}
		return nil
	})

	waitGroup.Go(func() error {
		<-ctx.Done()
		log.Println("graceful shutdown HTTP server")

		httpServer.Shutdown(context.Background())
		if err != nil {
			log.Printf("failed to shutdown HTTP gateway server, path: %s, error: %v\n", op, err)
			return err
		}

		log.Println("HTTP gateway server successfully stopped")
		return nil
	})

}
