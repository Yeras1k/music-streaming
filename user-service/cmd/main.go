package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/nats-io/nats.go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	pb "github.com/music-streaming/proto/user"
	"github.com/music-streaming/user-service/internal/handler"
	"github.com/music-streaming/user-service/internal/repository"
	"github.com/music-streaming/user-service/internal/service"
	"github.com/music-streaming/user-service/pkg/cache"
	"github.com/music-streaming/user-service/pkg/email"
	"github.com/music-streaming/user-service/pkg/events"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	RedisAddr  string
	NATSURL    string
	GRPCPort   string
	JWTSecret  string
	SMTPHost   string
	SMTPPort   string
	SMTPUser   string
	SMTPPass   string
}

func main() {
	config := loadConfig()

	// Initialize database
	db := initDatabase(config)

	// Auto migrate
	if err := db.AutoMigrate(&repository.UserModel{}, &repository.SessionModel{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Initialize Redis
	redisClient := initRedis(config)

	// Initialize NATS
	nc := initNATS(config)
	defer nc.Close()

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)

	// Initialize cache
	userCache := cache.NewUserCache(redisClient)

	// Initialize email sender
	emailSender := email.NewEmailSender(config.SMTPHost, config.SMTPPort, config.SMTPUser, config.SMTPPass)

	// Initialize event publisher
	eventPublisher := events.NewEventPublisher(nc)

	// Initialize service
	userService := service.NewUserService(
		userRepo,
		sessionRepo,
		userCache,
		emailSender,
		eventPublisher,
		config.JWTSecret,
	)

	// Initialize gRPC handler
	userHandler := handler.NewUserHandler(userService)

	// Create gRPC server with interceptors
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(loggingInterceptor),
		grpc.MaxConcurrentStreams(1000),
	)

	// Register services
	pb.RegisterUserServiceServer(grpcServer, userHandler)

	// Register health check
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("user-service", grpc_health_v1.HealthCheckResponse_SERVING)

	// Start listening
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", config.GRPCPort))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Graceful shutdown
	go func() {
		log.Printf("User service starting on port %s", config.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down user service...")
	grpcServer.GracefulStop()
	log.Println("User service stopped")
}

func loadConfig() *Config {
	return &Config{
		DBHost:     getEnv("DB_HOST", "postgres"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "music_user"),
		DBPassword: getEnv("DB_PASSWORD", "music_password"),
		DBName:     getEnv("DB_NAME", "music_db"),
		RedisAddr:  getEnv("REDIS_ADDR", "redis:6379"),
		NATSURL:    getEnv("NATS_URL", "nats://nats:4222"),
		GRPCPort:   getEnv("GRPC_PORT", "50051"),
		JWTSecret:  getEnv("JWT_SECRET", "default-secret-change-in-production"),
		SMTPHost:   getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:   getEnv("SMTP_PORT", "587"),
		SMTPUser:   getEnv("SMTP_USER", ""),
		SMTPPass:   getEnv("SMTP_PASS", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func initDatabase(config *Config) *gorm.DB {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		config.DBHost, config.DBUser, config.DBPassword, config.DBName, config.DBPort)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	return db
}

func initRedis(config *Config) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: config.RedisAddr,
	})
	if err := client.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	return client
}

func initNATS(config *Config) *nats.Conn {
	nc, err := nats.Connect(config.NATSURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	return nc
}

func loggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	log.Printf("Method: %s, Duration: %v, Error: %v", info.FullMethod, time.Since(start), err)
	return resp, err
}