package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"metadatatool/internal/config"
	"metadatatool/internal/handler"
	"metadatatool/internal/pkg/domain"
	"metadatatool/internal/pkg/errortracking"
	"metadatatool/internal/pkg/logger"
	"metadatatool/internal/pkg/middleware"
	"metadatatool/internal/pkg/storage"
	"metadatatool/internal/repository/base"
	"metadatatool/internal/repository/cached"
	"metadatatool/internal/usecase"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	log := logger.NewLogger()

	// Initialize error tracking
	errorTracker := errortracking.NewErrorTracker()

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Initialize PostgreSQL connection
	dbDSN := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User,
		cfg.Database.Password, cfg.Database.DBName,
	)
	db, err := sql.Open("postgres", dbDSN)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Initialize repositories
	baseTrackRepo := base.NewTrackRepository(db)
	trackRepo := cached.NewTrackRepository(redisClient, baseTrackRepo)
	baseUserRepo := base.NewUserRepository(db)
	userRepo := cached.NewUserRepository(redisClient, baseUserRepo)

	// Initialize services
	authService := usecase.NewAuthService(&cfg.Auth, userRepo)
	storageService, err := storage.NewStorageService(cfg.Storage)
	if err != nil {
		log.Fatalf("Failed to initialize storage service: %v", err)
	}

	// Initialize AI service
	aiService, err := usecase.NewOpenAIService(&cfg.AI)
	if err != nil {
		log.Fatalf("Failed to initialize AI service: %v", err)
	}

	// Initialize handlers
	healthHandler := handler.NewHealthHandler(redisClient)
	authHandler := handler.NewAuthHandler(authService, userRepo)
	trackHandler := handler.NewTrackHandler(trackRepo, aiService, errorTracker, cfg.Storage.Bucket)
	audioHandler := handler.NewAudioHandler(storageService, trackRepo)
	ddexHandler := handler.NewDDEXHandler(trackRepo)
	metricsHandler := handler.NewMetricsHandler()

	// Initialize router
	router := gin.New()
	router.Use(gin.Recovery())

	// Initialize routes
	initRoutes(
		router,
		healthHandler,
		authHandler,
		trackHandler,
		audioHandler,
		ddexHandler,
		metricsHandler,
		authService,
		userRepo,
		redisClient,
	)

	// Start server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
}

func initRoutes(
	router *gin.Engine,
	healthHandler *handler.HealthHandler,
	authHandler *handler.AuthHandler,
	trackHandler *handler.TrackHandler,
	audioHandler *handler.AudioHandler,
	ddexHandler *handler.DDEXHandler,
	metricsHandler *handler.MetricsHandler,
	authService domain.AuthService,
	userRepo domain.UserRepository,
	redisClient *redis.Client,
) {
	// Add metrics middleware to all routes
	router.Use(middleware.MetricsMiddleware())
	router.Use(middleware.DatabaseMetricsMiddleware())
	router.Use(middleware.AIMetricsMiddleware())
	router.Use(middleware.CacheMetricsMiddleware())

	// Metrics endpoints
	router.GET("/metrics", metricsHandler.PrometheusHandler())
	router.GET("/health", metricsHandler.HealthCheck)

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
	api.Use(middleware.APIKeyAuth(userRepo))

	// Configure rate limiting
	rateLimitCfg := middleware.RateLimitConfig{
		RequestsPerMinute: 60, // Adjust based on your needs
		BurstSize:         10,
		RedisClient:       redisClient,
	}

	// Track routes with rate limiting
	tracks := api.Group("/tracks")
	tracks.Use(middleware.RateLimit(rateLimitCfg))
	{
		// Public endpoints (still rate limited)
		tracks.GET("", trackHandler.ListTracks)
		tracks.GET("/:id", trackHandler.GetTrack)

		// Protected endpoints (require authentication)
		authenticated := tracks.Group("")
		authenticated.Use(middleware.AuthMiddleware(authService))
		{
			authenticated.POST("", trackHandler.CreateTrack)
			authenticated.PUT("/:id", trackHandler.UpdateTrack)
			authenticated.DELETE("/:id", trackHandler.DeleteTrack)
		}

		// Admin-only endpoints
		admin := authenticated.Group("")
		admin.Use(middleware.RoleGuard(domain.RoleAdmin))
		{
			admin.POST("/batch", trackHandler.BatchProcess)
			admin.POST("/export", trackHandler.ExportTracks)
		}
	}

	// Audio routes
	audio := api.Group("/audio")
	audio.Use(middleware.RateLimit(rateLimitCfg))
	audio.Use(middleware.AuthMiddleware(authService))
	{
		audio.POST("/upload", audioHandler.UploadAudio)
		audio.GET("/:id", audioHandler.GetAudioURL)
	}

	// DDEX routes (admin only)
	ddex := api.Group("/ddex")
	ddex.Use(middleware.RateLimit(rateLimitCfg))
	ddex.Use(middleware.AuthMiddleware(authService))
	ddex.Use(middleware.RoleGuard(domain.RoleAdmin))
	{
		ddex.POST("/validate", ddexHandler.ValidateERN)
		ddex.POST("/import", ddexHandler.ImportERN)
		ddex.POST("/export", ddexHandler.ExportERN)
	}
}
