package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/music-streaming/api-gateway/internal/handler"
	"github.com/music-streaming/api-gateway/internal/middleware"
	pb "github.com/music-streaming/proto/music"
	paymentpb "github.com/music-streaming/proto/payment"
	userpb "github.com/music-streaming/proto/user"
)

func main() {
	// Connect to gRPC services
	userConn, err := grpc.Dial(
		os.Getenv("USER_SERVICE_ADDR"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(50*1024*1024)),
	)
	if err != nil {
		log.Fatalf("Failed to connect to user service: %v", err)
	}
	defer userConn.Close()

	musicConn, err := grpc.Dial(
		os.Getenv("MUSIC_SERVICE_ADDR"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(100*1024*1024)),
	)
	if err != nil {
		log.Fatalf("Failed to connect to music service: %v", err)
	}
	defer musicConn.Close()

	paymentConn, err := grpc.Dial(
		os.Getenv("PAYMENT_SERVICE_ADDR"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("Failed to connect to payment service: %v", err)
	}
	defer paymentConn.Close()

	// Initialize Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
	})

	// Create gRPC clients
	userClient := userpb.NewUserServiceClient(userConn)
	musicClient := pb.NewMusicServiceClient(musicConn)
	paymentClient := paymentpb.NewPaymentServiceClient(paymentConn)

	// Create handlers
	userHandler := handler.NewUserHandler(userClient)
	musicHandler := handler.NewMusicHandler(musicClient)
	paymentHandler := handler.NewPaymentHandler(paymentClient)

	// Create middleware
	authMiddleware := middleware.NewAuthMiddleware(userClient, redisClient)

	// Setup Gin router
	router := gin.Default()

	// CORS configuration
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:3001"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Public routes
	auth := router.Group("/auth")
	{
		auth.POST("/register", userHandler.Register)
		auth.POST("/login", userHandler.Login)
		auth.POST("/verify", userHandler.VerifyEmail)
		auth.POST("/forgot-password", userHandler.ForgotPassword)
		auth.POST("/reset-password", userHandler.ResetPassword)
	}

	// Protected routes
	api := router.Group("/api")
	api.Use(authMiddleware.Authenticate())
	{
		// User routes
		api.GET("/user/profile", userHandler.GetProfile)
		api.PUT("/user/profile", userHandler.UpdateProfile)
		api.POST("/user/change-password", userHandler.ChangePassword)
		api.POST("/user/logout", userHandler.Logout)

		// Music routes
		api.POST("/tracks/upload", musicHandler.UploadTrack)
		api.GET("/tracks/:id", musicHandler.GetTrack)
		api.GET("/tracks", musicHandler.ListTracks)
		api.GET("/tracks/search", musicHandler.SearchTracks)
		api.POST("/tracks/:id/like", musicHandler.LikeTrack)

		// Playlist routes
		api.POST("/playlists", musicHandler.CreatePlaylist)
		api.GET("/playlists", musicHandler.GetUserPlaylists)
		api.GET("/playlists/:id", musicHandler.GetPlaylist)
		api.POST("/playlists/:id/tracks", musicHandler.AddToPlaylist)
		api.DELETE("/playlists/:id/tracks/:trackId", musicHandler.RemoveFromPlaylist)

		// Queue routes
		api.POST("/queue/add", musicHandler.AddToQueue)
		api.GET("/queue", musicHandler.GetQueue)

		// Recommendations
		api.GET("/recommendations", musicHandler.GetRecommendations)

		// Payment routes
		api.GET("/subscription", paymentHandler.GetSubscription)
		api.POST("/subscription", paymentHandler.CreateSubscription)
		api.DELETE("/subscription/:id", paymentHandler.CancelSubscription)
		api.POST("/payments/process", paymentHandler.ProcessPayment)
		api.GET("/payments/history", paymentHandler.GetPaymentHistory)
		api.POST("/coupons/apply", paymentHandler.ApplyCoupon)
		api.GET("/pricing-plans", paymentHandler.GetPricingPlans)
	}

	// Start server
	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	log.Println("API Gateway running on :8080")

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("API Gateway stopped")
}
