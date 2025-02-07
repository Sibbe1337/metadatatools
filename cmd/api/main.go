package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"metadatatool/internal/config"
	"metadatatool/internal/handler"
	"metadatatool/internal/handler/middleware"
	"metadatatool/internal/pkg/domain"
	"metadatatool/internal/pkg/errortracking"
	"metadatatool/internal/pkg/logger"
	"metadatatool/internal/pkg/storage"
	"metadatatool/internal/repository/base"
	"metadatatool/internal/repository/cached"
	redisrepo "metadatatool/internal/repository/redis"
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

	// Initialize session store
	sessionConfig := &domain.SessionConfig{
		SessionDuration:    24 * time.Hour, // Sessions last 24 hours
		CleanupInterval:    time.Hour,      // Cleanup every hour
		MaxSessionsPerUser: 5,              // Max 5 concurrent sessions per user
	}
	sessionStore := redisrepo.NewSessionStore(redisClient, sessionConfig)

	// Initialize AI service
	aiService, err := usecase.NewOpenAIService(&cfg.AI)
	if err != nil {
		log.Fatalf("Failed to initialize AI service: %v", err)
	}

	// Create track validator
	validator := domain.NewTrackValidator()

	// Create track handler
	trackHandler := handler.NewTrackHandler(
		trackRepo,
		aiService,
		validator,
		errorTracker,
		cfg.Storage.Bucket,
	)

	// Initialize handlers
	healthHandler := handler.NewHealthHandler(redisClient)
	authHandler := handler.NewAuthHandler(authService, userRepo, sessionStore)
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
		sessionStore,
		sessionConfig,
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
	sessionStore domain.SessionStore,
	sessionConfig *domain.SessionConfig,
) {
	// Add basic middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Metrics endpoints
	router.GET("/metrics", metricsHandler.PrometheusHandler())
	router.GET("/health", metricsHandler.HealthCheck)

	// Health check
	router.GET("/health", healthHandler.Check)

	// Public routes
	auth := router.Group("/api/v1/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", middleware.CreateSession(sessionStore, sessionConfig), authHandler.Login)
		auth.POST("/refresh", authHandler.RefreshToken)
		auth.POST("/logout", middleware.ClearSession(sessionStore), authHandler.Logout)

		// Session management routes (protected)
		sessions := auth.Group("")
		sessions.Use(middleware.Auth(authService))
		sessions.Use(middleware.Session(sessionStore, sessionConfig))
		{
			sessions.GET("/sessions", authHandler.GetActiveSessions)
			sessions.DELETE("/sessions/:id", authHandler.RevokeSession)
			sessions.DELETE("/sessions", authHandler.RevokeAllSessions)
		}
	}

	// Protected routes
	api := router.Group("/api/v1")
	api.Use(middleware.APIKeyAuth(userRepo))

	// Track routes
	tracks := api.Group("/tracks")
	{
		// Public endpoints
		tracks.GET("", trackHandler.ListTracks)
		tracks.GET("/:id", trackHandler.GetTrack)

		// Protected endpoints (require authentication)
		authenticated := tracks.Group("")
		authenticated.Use(middleware.Auth(authService))
		{
			authenticated.POST("", trackHandler.CreateTrack)
			authenticated.PUT("/:id", trackHandler.UpdateTrack)
			authenticated.DELETE("/:id", trackHandler.DeleteTrack)
		}

		// Admin-only endpoints
		admin := authenticated.Group("")
		admin.Use(middleware.RequireRole(domain.RoleAdmin))
		{
			admin.POST("/batch", trackHandler.BatchProcess)
			admin.POST("/export", trackHandler.ExportTracks)
		}
	}

	// Audio routes
	audio := api.Group("/audio")
	audio.Use(middleware.Auth(authService))
	{
		audio.POST("/upload", audioHandler.UploadAudio)
		audio.GET("/:id", audioHandler.GetAudioURL)
	}

	// DDEX routes (admin only)
	ddex := api.Group("/ddex")
	ddex.Use(middleware.Auth(authService))
	ddex.Use(middleware.RequireRole(domain.RoleAdmin))
	{
		ddex.POST("/validate", ddexHandler.ValidateERN)
		ddex.POST("/import", ddexHandler.ImportERN)
		ddex.POST("/export", ddexHandler.ExportERN)
	}
}
