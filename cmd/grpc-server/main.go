package main

import (
	"log/slog"
	"net"
	"os"

	grpcHandlers "github.com/brenocoelho/messaging-app-go/internal/grpc"
	"github.com/brenocoelho/messaging-app-go/internal/repositories"
	"github.com/brenocoelho/messaging-app-go/internal/services"
	"github.com/brenocoelho/messaging-app-go/pkg/config"
	"github.com/brenocoelho/messaging-app-go/pkg/jwt"
	"github.com/brenocoelho/messaging-app-go/pkg/pgconn"
	"github.com/brenocoelho/messaging-app-go/pkg/redisconn"
	"google.golang.org/grpc"
)

type Config struct {
	PostgresReadHost     string `mapstructure:"POSTGRES_READ_HOST"`
	PostgresReadPort     string `mapstructure:"POSTGRES_READ_PORT"`
	PostgresReadUser     string `mapstructure:"POSTGRES_READ_USER"`
	PostgresReadPassword string `mapstructure:"POSTGRES_READ_PASSWORD"`
	PostgresReadDBName   string `mapstructure:"POSTGRES_READ_DB_NAME"`

	PostgresWriteHost     string `mapstructure:"POSTGRES_WRITE_HOST"`
	PostgresWritePort     string `mapstructure:"POSTGRES_WRITE_PORT"`
	PostgresWriteUser     string `mapstructure:"POSTGRES_WRITE_USER"`
	PostgresWritePassword string `mapstructure:"POSTGRES_WRITE_PASSWORD"`
	PostgresWriteDBName   string `mapstructure:"POSTGRES_WRITE_DB_NAME"`

	GRPCPort string `mapstructure:"GRPC_PORT"`

	RedisHost      string `mapstructure:"REDIS_HOST"`
	RedisPort      string `mapstructure:"REDIS_PORT"`
	RedisPassword  string `mapstructure:"REDIS_PASSWORD"`
	IdempotencyTTL int    `mapstructure:"IDEMPOTENCY_TTL_MINUTES"`
}

func main() {
	if err := run(); err != nil {
		slog.Error("Error running the application", "error", err)
		os.Exit(1)
	}
}

func run() error {
	slog.Info("Starting gRPC server...")

	cfg := Config{}
	err := config.LoadConfig(&cfg)
	if err != nil {
		slog.Error("Error loading config", "error", err)
		return err
	}

	readerPool, err := pgconn.NewPostgres(&pgconn.Config{
		Host:     cfg.PostgresReadHost,
		Port:     cfg.PostgresReadPort,
		Password: cfg.PostgresReadPassword,
		User:     cfg.PostgresReadUser,
		Name:     cfg.PostgresReadDBName,
	})
	if err != nil {
		slog.Error("could not create the read db client")
		return err
	}

	writerPool, err := pgconn.NewPostgres(&pgconn.Config{
		Host:     cfg.PostgresWriteHost,
		Port:     cfg.PostgresWritePort,
		Password: cfg.PostgresWritePassword,
		User:     cfg.PostgresWriteUser,
		Name:     cfg.PostgresWriteDBName,
	})
	if err != nil {
		slog.Error("could not create the write db client")
		return err
	}

	cacheClient, err := redisconn.NewRedis(&redisconn.Config{
		Host:     cfg.RedisHost,
		Port:     cfg.RedisPort,
		Password: cfg.RedisPassword,
	})
	if err != nil {
		slog.Error("could not create the redis client")
		return err
	}

	port := cfg.GRPCPort
	if port == "" {
		port = "50051"
	}

	repos := repositories.NewRepositories(readerPool, writerPool)

	svcs := services.NewServices(repos, cacheClient, cfg.IdempotencyTTL)

	jwtInterceptor := jwt.NewInterceptor(svcs.JWT)

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}

	server := grpc.NewServer(
		grpc.UnaryInterceptor(jwtInterceptor.UnaryInterceptor),
		grpc.StreamInterceptor(jwtInterceptor.StreamInterceptor),
	)

	grpcServer := grpcHandlers.NewGRPCServer(
		svcs.Messages,
		svcs.Chats,
		svcs.Users,
		svcs.Realtime,
	)
	grpcServer.RegisterServices(server)

	slog.Info("gRPC server listening", "port", port)

	if err := server.Serve(lis); err != nil {
		return err
	}

	return nil
}
