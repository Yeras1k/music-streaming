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

	pb "github.com/music-streaming/proto/payment"
	"github.com/music-streaming/payment-service/internal/handler"
	"github.com/music-streaming/payment-service/internal/repository"
	"github.com/music-streaming/payment-service/internal/service"
	"github.com/music-streaming/payment-service/pkg/events"
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
}

func main() {
	config := loadConfig()

	// Initialize database
	db := initDatabase(config)

	// Auto migrate
	if err := db.AutoMigrate(&repository.SubscriptionModel{}, &repository.PaymentTransactionModel{}, &repository.CouponModel{}, &repository.PricingPlanModel{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Seed initial data
	seedDatabase(db)

	// Initialize Redis
	redisClient := initRedis(config)

	// Initialize NATS
	nc := initNATS(config)
	defer nc.Close()

	// Initialize repositories
	subscriptionRepo := repository.NewSubscriptionRepository(db)
	paymentRepo := repository.NewPaymentRepository(db)
	planRepo := repository.NewPricingPlanRepository(db)
	couponRepo := repository.NewCouponRepository(db)

	// Initialize event publisher
	eventPublisher := events.NewEventPublisher(nc)

	// Initialize service
	paymentService := service.NewPaymentService(
		subscriptionRepo,
		paymentRepo,
		planRepo,
		couponRepo,
		eventPublisher,
		redisClient,
	)

	// Initialize gRPC handler
	paymentHandler := handler.NewPaymentHandler(paymentService)

	// Create gRPC server
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(loggingInterceptor),
		grpc.MaxConcurrentStreams(1000),
	)

	// Register services
	pb.RegisterPaymentServiceServer(grpcServer, paymentHandler)

	// Register health check
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("payment-service", grpc_health_v1.HealthCheckResponse_SERVING)

	// Start listening
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", config.GRPCPort))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Graceful shutdown
	go func() {
		log.Printf("Payment service starting on port %s", config.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down payment service...")
	grpcServer.GracefulStop()
	log.Println("Payment service stopped")
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
		GRPCPort:   getEnv("GRPC_PORT", "50053"),
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

func seedDatabase(db *gorm.DB) {
	var count int64
	db.Model(&repository.PricingPlanModel{}).Count(&count)
	if count > 0 {
		return
	}

	plans := []repository.PricingPlanModel{
		{Name: "Free", Price: 0, Currency: "USD", Interval: "month", Quality: 128, OfflineMode: false},
		{Name: "Premium", Price: 9.99, Currency: "USD", Interval: "month", Quality: 320, OfflineMode: true},
		{Name: "Family", Price: 14.99, Currency: "USD", Interval: "month", Quality: 320, OfflineMode: true},
		{Name: "Student", Price: 4.99, Currency: "USD", Interval: "month", Quality: 320, OfflineMode: true},
	}

	for _, plan := range plans {
		db.Create(&plan)
	}

	coupons := []repository.CouponModel{
		{Code: "WELCOME20", Discount: 20, ExpiresAt: time.Now().AddDate(1, 0, 0)},
		{Code: "SAVE10", Discount: 10, ExpiresAt: time.Now().AddDate(0, 6, 0)},
	}

	for _, coupon := range coupons {
		db.Create(&coupon)
	}

	log.Println("Database seeded with pricing plans and coupons")
}

func loggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	log.Printf("Method: %s, Duration: %v, Error: %v", info.FullMethod, time.Since(start), err)
	return resp, err
}