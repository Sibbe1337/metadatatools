package main

import (
	"context"
	"fmt"
	"metadatatool/internal/domain"
	"metadatatool/internal/handler"
	"metadatatool/internal/handler/middleware"
	"metadatatool/internal/pkg/analytics"
	pkgconfig "metadatatool/internal/pkg/config"
	"metadatatool/internal/pkg/converter"
	pkgdomain "metadatatool/internal/pkg/domain"
	"metadatatool/internal/pkg/errortracking"
	"metadatatool/internal/pkg/logger"
	"metadatatool/internal/pkg/metrics"
	"metadatatool/internal/pkg/validator"
	"metadatatool/internal/repository/ai"
	"metadatatool/internal/repository/base"
	queuepkg "metadatatool/internal/repository/queue"
	"metadatatool/internal/repository/redis"
	storagepkg "metadatatool/internal/repository/storage"
	"metadatatool/internal/usecase"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	goredis "github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Load .env file if it exists (optional)
	_ = godotenv.Load()

	// Initialize logger
	log := logger.NewLogger()

	// Load configuration
	cfg, err := pkgconfig.Load()
	if err != nil {
		log.Warnf("Failed to load config: %v", err)
		// Continue with defaults
	}

	// Initialize error tracking
	errorTracker := errortracking.NewErrorTracker()

	var redisClient *goredis.Client
	if os.Getenv("DISABLE_REDIS") != "true" {
		// Initialize Redis client
		redisClient = goredis.NewClient(&goredis.Options{
			Addr:     cfg.Redis.GetAddress(),
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})

		// Test Redis connection
		if err := redisClient.Ping(context.Background()).Err(); err != nil {
			log.Warn("Failed to connect to Redis:", err)
			redisClient = nil
		}
	} else {
		log.Info("Redis is disabled")
		redisClient = nil
	}

	// Initialize queue service (optional)
	var queueService *queuepkg.PubSubService
	if !cfg.Queue.Disabled && os.Getenv("DISABLE_QUEUE") != "true" {
		queueConfig := &queuepkg.PubSubConfig{
			ProjectID:          cfg.Queue.ProjectID,
			HighPriorityTopic:  cfg.Queue.HighPriorityTopic,
			LowPriorityTopic:   cfg.Queue.LowPriorityTopic,
			DeadLetterTopic:    cfg.Queue.DeadLetterTopic,
			SubscriptionPrefix: cfg.Queue.SubscriptionPrefix,
			MaxRetries:         cfg.Queue.MaxRetries,
			AckDeadline:        cfg.Queue.AckDeadline,
			RetentionDuration:  cfg.Queue.RetentionDuration,
		}

		queueMetrics := metrics.NewQueueMetrics()
		var err error
		queueService, err = queuepkg.NewPubSubService(context.Background(), queueConfig, queueMetrics)
		if err != nil {
			log.Warnf("Failed to initialize queue service: %v", err)
		} else {
			defer func() {
				if queueService != nil {
					queueService.Close()
				}
			}()
		}
	} else {
		log.Info("Queue service is disabled")
	}

	var db *gorm.DB
	if os.Getenv("DISABLE_DB") != "true" {
		// Initialize PostgreSQL connection
		dbConfig := postgres.Config{
			DSN: fmt.Sprintf(
				"postgresql://%s:%s@%s:%d/%s?sslmode=%s",
				cfg.Database.User, cfg.Database.Password, cfg.Database.Host,
				cfg.Database.Port, cfg.Database.DBName, cfg.Database.SSLMode,
			),
			PreferSimpleProtocol: true,
		}
		log.Printf("Database DSN: %s", dbConfig.DSN)
		db, err = gorm.Open(postgres.New(dbConfig), &gorm.Config{})
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}

		// Get underlying *sql.DB to close it later
		sqlDB, err := db.DB()
		if err != nil {
			log.Fatalf("Failed to get underlying *sql.DB: %v", err)
		}
		defer sqlDB.Close()

		// Test database connection
		if err := sqlDB.Ping(); err != nil {
			log.Fatalf("Failed to ping database: %v", err)
		}
	} else {
		log.Info("Database is disabled")
	}

	// Initialize analytics service (optional)
	var analyticsService *analytics.BigQueryService
	if os.Getenv("DISABLE_ANALYTICS") != "true" {
		var err error
		analyticsService, err = analytics.NewBigQueryService(cfg.Queue.ProjectID, cfg.Queue.HighPriorityTopic)
		if err != nil {
			log.Warnf("Failed to initialize analytics service: %v", err)
		}
	} else {
		log.Info("Analytics service is disabled")
	}

	// Initialize storage service (optional)
	var storageService pkgdomain.StorageService
	if os.Getenv("DISABLE_STORAGE") != "true" {
		var err error
		storageService, err = storagepkg.NewS3Storage(&cfg.Storage)
		if err != nil {
			log.Warnf("Failed to initialize storage service: %v", err)
		}
	} else {
		log.Info("Storage service is disabled")
	}

	// Initialize validator
	validatorService := validator.NewValidator()

	// Initialize AI service (optional)
	var pkgAIService pkgdomain.AIService
	if os.Getenv("DISABLE_AI") != "true" {
		// Create AI service config
		aiConfig := &ai.Config{
			EnableFallback:        true,
			TimeoutSeconds:        int(cfg.AI.Timeout.Seconds()),
			MinConfidence:         cfg.AI.MinConfidence,
			MaxConcurrentRequests: cfg.AI.BatchSize,
			RetryAttempts:         3,
			RetryBackoffSeconds:   2,
			OpenAIConfig: &pkgdomain.OpenAIConfig{
				APIKey:                cfg.AI.APIKey,
				Endpoint:              cfg.AI.BaseURL,
				TimeoutSeconds:        int(cfg.AI.Timeout.Seconds()),
				MinConfidence:         cfg.AI.MinConfidence,
				MaxConcurrentRequests: cfg.AI.BatchSize,
				RetryAttempts:         3,
				RetryBackoffSeconds:   2,
				RequestsPerSecond:     10,
			},
			Qwen2Config: &pkgdomain.Qwen2Config{
				APIKey:                cfg.AI.APIKey,
				Endpoint:              cfg.AI.BaseURL,
				TimeoutSeconds:        int(cfg.AI.Timeout.Seconds()),
				MinConfidence:         cfg.AI.MinConfidence,
				MaxConcurrentRequests: cfg.AI.BatchSize,
				RetryAttempts:         3,
				RetryBackoffSeconds:   2,
			},
		}

		// Create composite AI service
		compositeService, err := ai.NewCompositeAIService(aiConfig, analyticsService)
		if err != nil {
			log.Warnf("Failed to create composite AI service: %v", err)
		} else {
			pkgAIService = compositeService
		}
	} else {
		log.Info("AI service is disabled")
	}

	// Initialize repositories and stores
	var (
		baseTrackRepo   domain.TrackRepository
		pkgTrackRepo    pkgdomain.TrackRepository
		baseUserRepo    domain.UserRepository
		pkgUserRepo     pkgdomain.UserRepository
		sessionStore    domain.SessionStore
		sessionStorePkg pkgdomain.SessionStore
	)

	if db != nil {
		baseTrackRepo = base.NewTrackRepository(db)
		pkgTrackRepo = base.NewPkgTrackRepository(db)
		baseUserRepo = base.NewUserRepository(db)
		pkgUserRepo = base.NewPkgUserRepository(db)
	}

	var sessionStoreWrapper *converter.SessionStoreWrapper
	if redisClient != nil {
		sessionStore = redis.NewSessionStore(redisClient, configToDomainSession(cfg.Session))
		sessionStorePkg = redis.NewPkgSessionStore(redisClient, configToPkgSession(cfg.Session))
		sessionStoreWrapper = converter.NewSessionStoreWrapper(sessionStore, sessionStorePkg)
	} else {
		log.Info("Session store is disabled (Redis not available)")
		sessionStoreWrapper = converter.NewSessionStoreWrapper(nil, nil)
	}

	trackRepoWrapper := converter.NewTrackRepositoryWrapper(baseTrackRepo, pkgTrackRepo)
	userRepoWrapper := converter.NewUserRepositoryWrapper(baseUserRepo, pkgUserRepo)

	// Initialize auth service
	authService := usecase.NewAuthService(&cfg.Auth)
	authServiceWrapper := converter.NewAuthServiceWrapper(authService, nil)

	// Initialize use cases
	authUseCase := usecase.NewAuthUseCase(userRepoWrapper.Internal(), sessionStoreWrapper.Internal(), authServiceWrapper.Internal())
	userUseCase := usecase.NewUserUseCase(userRepoWrapper.Pkg())

	// Initialize handlers
	healthHandler := handler.NewHealthHandler(redisClient)
	var metricsHandler *handler.MetricsHandler
	if os.Getenv("DISABLE_METRICS") != "true" {
		metricsHandler = handler.NewMetricsHandler()
	}

	authHandler := handler.NewAuthHandler(authUseCase, userUseCase, sessionStoreWrapper.Internal())
	trackHandler := handler.NewTrackHandler(
		trackRepoWrapper.Pkg(),
		pkgAIService,
		storageService,
		validatorService,
		errorTracker,
	)

	// Initialize router with minimal middleware
	router := gin.New()
	router.Use(gin.Recovery())

	// Only add auth middleware if session store is available
	if sessionStoreWrapper.Pkg() != nil {
		router.Use(middleware.Auth(authServiceWrapper.Pkg()))
		router.Use(middleware.Session(sessionStoreWrapper.Pkg(), configToPkgSession(cfg.Session)))
	}

	// Register routes
	router.GET("/health", healthHandler.Check)
	if metricsHandler != nil {
		router.GET("/metrics", metricsHandler.PrometheusHandler())
	}

	// API routes
	api := router.Group("/api/v1")
	{
		// Auth routes
		auth := api.Group("/auth")
		if sessionStoreWrapper.Pkg() != nil {
			auth.Use(middleware.RequireSession(sessionStoreWrapper.Pkg()))
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.POST("/logout", authHandler.Logout)
		}

		// Track routes
		tracks := api.Group("/tracks")
		if sessionStoreWrapper.Pkg() != nil {
			tracks.Use(middleware.RequireSession(sessionStoreWrapper.Pkg()))
		}
		{
			tracks.POST("", trackHandler.CreateTrack)
			tracks.GET("/:id", trackHandler.GetTrack)
			tracks.PUT("/:id", trackHandler.UpdateTrack)
			tracks.DELETE("/:id", trackHandler.DeleteTrack)
			tracks.GET("", trackHandler.ListTracks)
			tracks.POST("/search", trackHandler.SearchTracks)
		}
	}

	// Get port from environment variable for Cloud Run compatibility
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port if not specified
	}

	// Start server
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Channel to signal when the server is ready to accept connections
	serverReady := make(chan struct{})
	serverErr := make(chan error, 1)

	// Create a timeout context for startup
	startupCtx, startupCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer startupCancel()

	// Start server in a goroutine
	go func() {
		// Create a listener first
		listener, err := net.Listen("tcp", srv.Addr)
		if err != nil {
			serverErr <- fmt.Errorf("failed to create listener: %v", err)
			return
		}

		// Signal that we're ready to accept connections
		close(serverReady)

		log.Printf("Starting server on port %s", port)
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Wait for either server error or interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Wait for the server to be ready to accept connections
	select {
	case err := <-serverErr:
		log.Fatalf("Server failed to start: %v", err)
	case <-startupCtx.Done():
		log.Fatal("Server failed to start within timeout")
	case <-serverReady:
		log.Info("Server is ready to accept connections")
	}

	// Try to connect to our own health endpoint to verify the server is up
	healthCheckCtx, healthCheckCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer healthCheckCancel()

	healthCheckDone := make(chan struct{})
	go func() {
		defer close(healthCheckDone)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-healthCheckCtx.Done():
				return
			case <-ticker.C:
				client := &http.Client{
					Timeout: 1 * time.Second,
				}
				resp, err := client.Get(fmt.Sprintf("http://localhost:%s/health", port))
				if err == nil {
					resp.Body.Close()
					if resp.StatusCode == http.StatusOK {
						log.Info("Health check passed")
						return
					}
					log.Warnf("Health check returned status %d", resp.StatusCode)
				} else {
					log.Warnf("Health check failed: %v", err)
				}
			}
		}
	}()

	// Wait for health check to complete or timeout
	select {
	case <-healthCheckDone:
		log.Info("Server is healthy and ready to serve traffic")
	case <-healthCheckCtx.Done():
		log.Fatal("Health check timed out")
	case err := <-serverErr:
		log.Fatalf("Server error during health check: %v", err)
	}

	// Wait for server error or signal
	select {
	case err := <-serverErr:
		log.Fatalf("Server error: %v", err)
	case <-quit:
		log.Info("Shutting down server...")
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Warnf("Server forced to shutdown: %v", err)
	}

	log.Info("Server exited properly")
}

func configToDomainSession(cfg pkgconfig.SessionConfig) domain.SessionConfig {
	return domain.SessionConfig{
		CookieName:         cfg.CookieName,
		CookieDomain:       cfg.CookieDomain,
		CookiePath:         cfg.CookiePath,
		CookieSecure:       cfg.CookieSecure,
		CookieHTTPOnly:     cfg.CookieHTTPOnly,
		CookieSameSite:     cfg.CookieSameSite,
		SessionDuration:    cfg.SessionDuration,
		CleanupInterval:    cfg.CleanupInterval,
		MaxSessionsPerUser: cfg.MaxSessionsPerUser,
	}
}

func configToPkgSession(cfg pkgconfig.SessionConfig) pkgdomain.SessionConfig {
	return pkgdomain.SessionConfig{
		CookieName:         cfg.CookieName,
		CookieDomain:       cfg.CookieDomain,
		CookiePath:         cfg.CookiePath,
		CookieSecure:       cfg.CookieSecure,
		CookieHTTPOnly:     cfg.CookieHTTPOnly,
		CookieSameSite:     cfg.CookieSameSite,
		SessionDuration:    cfg.SessionDuration,
		CleanupInterval:    cfg.CleanupInterval,
		MaxSessionsPerUser: cfg.MaxSessionsPerUser,
	}
}
