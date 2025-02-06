package main

import (
	"context"
	"metadatatool/internal/config"
	"metadatatool/internal/handler"
	"metadatatool/internal/pkg/database"
	"metadatatool/internal/pkg/domain"
	"metadatatool/internal/pkg/errortracking"
	"metadatatool/internal/pkg/logger"
	"metadatatool/internal/pkg/storage"
	"metadatatool/internal/repository/cached"
	"metadatatool/internal/repository/postgres"
	"metadatatool/internal/usecase"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize logger
	log := logger.NewLogger()

	// Initialize error tracking
	errortracker := errortracking.NewErrorTracker()
	defer errortracker.Close()

	// Connect to PostgreSQL
	db, err := database.ConnectPostgres()
	if err != nil {
		log.Fatal(err)
	}

	// Connect to Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Host + ":" + cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer redisClient.Close()

	// Initialize repositories
	var trackRepo domain.TrackRepository = postgres.NewPostgresTrackRepository(db)
	var userRepo domain.UserRepository = postgres.NewUserRepository(db)

	// Add caching layer if needed
	if cfg.Redis.Enabled {
		trackRepo = cached.NewTrackRepository(redisClient, trackRepo)
		userRepo = cached.NewUserRepository(redisClient, userRepo)
	}

	// Initialize storage
	storageClient := storage.NewStorageClient(&cfg.Storage)

	// Initialize services with configurations
	authService := usecase.NewAuthService(&cfg.Auth, userRepo)
	audioService := usecase.NewAudioService(storageClient, trackRepo)
	aiService, err := usecase.NewOpenAIService(&cfg.AI)
	if err != nil {
		log.Fatal(err)
	}
	ddexService := usecase.NewDDEXService()

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService, userRepo)
	trackHandler := handler.NewTrackHandler(trackRepo, aiService)
	audioHandler := handler.NewAudioHandler(audioService)
	ddexHandler := handler.NewDDEXHandler(ddexService, trackRepo)
	healthHandler := handler.NewHealthHandler(db, redisClient)

	// Initialize Gin router
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.Default()

	// Add middleware
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// Initialize routes
	initRoutes(router, healthHandler, authHandler, trackHandler, audioHandler, ddexHandler)

	// Start server
	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Info("Server exiting")
}

func initRoutes(router *gin.Engine, healthHandler *handler.HealthHandler, authHandler *handler.AuthHandler, trackHandler *handler.TrackHandler, audioHandler *handler.AudioHandler, ddexHandler *handler.DDEXHandler) {
	// Health check
	router.GET("/health", healthHandler.Check)

	// Public routes
	auth := router.Group("/api/v1/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.RefreshToken)
	}

	// Protected routes
	api := router.Group("/api/v1")
	{
		// User routes
		api.POST("/api-key", authHandler.GenerateAPIKey)

		// Track routes
		tracks := api.Group("/tracks")
		{
			tracks.POST("", trackHandler.CreateTrack)
			tracks.GET("/:id", trackHandler.GetTrack)
			tracks.PUT("/:id", trackHandler.UpdateTrack)
			tracks.DELETE("/:id", trackHandler.DeleteTrack)
			tracks.GET("", trackHandler.ListTracks)
			tracks.POST("/enrich", trackHandler.EnrichTrack)
			tracks.POST("/validate", trackHandler.ValidateTrack)
		}

		// Audio routes
		audio := api.Group("/audio")
		{
			audio.POST("/upload", audioHandler.UploadAudio)
			audio.GET("/:id", audioHandler.GetAudioURL)
		}

		// DDEX routes
		ddex := api.Group("/ddex")
		{
			ddex.POST("/validate", ddexHandler.ValidateTrackDDEX)
			ddex.GET("/:id", ddexHandler.ExportTrackDDEX)
			ddex.POST("/batch", ddexHandler.BatchExportDDEX)
		}
	}
}
