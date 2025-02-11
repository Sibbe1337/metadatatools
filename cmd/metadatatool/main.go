// Package main implements a command-line interface for the metadata tool.
//
// The metadatatool CLI provides functionality to:
//   - Enrich track metadata using AI services
//   - Validate track metadata against quality standards
//   - Export tracks in various formats (JSON, DDEX)
//
// Usage:
//
//	metadatatool -action=enrich -track=<track_id>
//	metadatatool -action=validate -track=<track_id>
//	metadatatool -action=export -track=<track_id> -format=[json|ddex]
//	metadatatool -action=export -batch=<file> -format=[json|ddex]
//
// Environment Variables:
//   - DB_HOST: PostgreSQL host
//   - DB_PORT: PostgreSQL port
//   - DB_USER: PostgreSQL user
//   - DB_PASSWORD: PostgreSQL password
//   - DB_NAME: PostgreSQL database name
//   - AI_API_KEY: API key for AI services
//   - AI_BASE_URL: Base URL for AI services
//   - BIGQUERY_PROJECT: Google Cloud project ID
//   - BIGQUERY_DATASET: BigQuery dataset name
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"metadatatool/internal/pkg/analytics"
	"metadatatool/internal/pkg/config"
	"metadatatool/internal/pkg/ddex"
	"metadatatool/internal/pkg/domain"
	"metadatatool/internal/repository/ai"
	"metadatatool/internal/repository/base"
	"metadatatool/internal/usecase"
	"os"
	"strings"

	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Command line flags
type flags struct {
	trackID   *string // ID of the track to process
	action    *string // Action to perform (enrich, validate, export)
	format    *string // Export format (json, ddex)
	batchFile *string // File containing list of track IDs to process
}

// parseFlags parses and validates command line flags
func parseFlags() *flags {
	f := &flags{
		trackID:   flag.String("track", "", "Track ID to process"),
		action:    flag.String("action", "", "Action to perform (enrich, validate, export)"),
		format:    flag.String("format", "json", "Export format (json, ddex)"),
		batchFile: flag.String("batch", "", "File containing list of track IDs to process"),
	}
	flag.Parse()
	return f
}

func main() {
	flags := parseFlags()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize services
	services, err := initializeServices(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize services: %v", err)
	}
	defer services.cleanup()

	// Process command
	ctx := context.Background()
	if err := processCommand(ctx, flags, services); err != nil {
		log.Fatalf("Command failed: %v", err)
	}
}

// services holds all the service dependencies
type services struct {
	analytics *analytics.BigQueryService
	ai        domain.AIService
	tracks    domain.TrackRepository
	ddex      domain.DDEXService
	db        *sql.DB
}

// cleanup performs cleanup of all services
func (s *services) cleanup() {
	if s.analytics != nil {
		s.analytics.Close()
	}
	if s.db != nil {
		s.db.Close()
	}
}

// initializeServices initializes all required services
func initializeServices(cfg *config.AppConfig) (*services, error) {
	// Initialize BigQuery analytics
	analyticsService, err := analytics.NewBigQueryService(
		cfg.Queue.ProjectID,
		cfg.Queue.HighPriorityTopic,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize BigQuery: %w", err)
	}

	// Initialize AI service
	aiConfig := &ai.Config{
		EnableFallback:        cfg.AI.Experiment.EnableFallback,
		TimeoutSeconds:        int(cfg.AI.Timeout.Seconds()),
		MinConfidence:         cfg.AI.MinConfidence,
		MaxConcurrentRequests: cfg.AI.BatchSize,
		RetryAttempts:         3,
		RetryBackoffSeconds:   5,
		OpenAIConfig: &domain.OpenAIConfig{
			APIKey:                cfg.AI.APIKey,
			Endpoint:              cfg.AI.BaseURL,
			TimeoutSeconds:        int(cfg.AI.Timeout.Seconds()),
			MinConfidence:         cfg.AI.MinConfidence,
			MaxConcurrentRequests: cfg.AI.BatchSize,
			RetryAttempts:         3,
			RetryBackoffSeconds:   5,
			RequestsPerSecond:     10,
		},
		Qwen2Config: &domain.Qwen2Config{
			APIKey:                cfg.AI.APIKey,
			Endpoint:              cfg.AI.BaseURL,
			TimeoutSeconds:        int(cfg.AI.Timeout.Seconds()),
			MinConfidence:         cfg.AI.MinConfidence,
			MaxConcurrentRequests: cfg.AI.BatchSize,
			RetryAttempts:         3,
			RetryBackoffSeconds:   5,
		},
	}

	aiService, err := ai.NewCompositeAIService(aiConfig, analyticsService)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize AI service: %w", err)
	}

	// Initialize database connection
	dbDSN := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DBName,
	)
	db, err := gorm.Open(postgres.Open(dbDSN), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying *sql.DB for cleanup
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying *sql.DB: %w", err)
	}

	// Initialize repositories and services
	pkgTrackRepo := base.NewPkgTrackRepository(db)

	// Initialize DDEX service
	schemaValidator := ddex.NewXMLSchemaValidator()
	ddexConfig := &usecase.DDEXConfig{
		MessageSender:    "MetadataTool",
		MessageRecipient: "DSP",
		SchemaPath:       "schemas/ddex/ern/4.3/release-notification.xsd",
		ValidateSchema:   true,
	}
	ddexService := usecase.NewDDEXService(schemaValidator, ddexConfig)

	return &services{
		analytics: analyticsService,
		ai:        aiService,
		tracks:    pkgTrackRepo,
		ddex:      ddexService,
		db:        sqlDB,
	}, nil
}

// processCommand processes the CLI command based on the provided flags
func processCommand(ctx context.Context, f *flags, s *services) error {
	switch *f.action {
	case "enrich":
		if *f.trackID == "" {
			return fmt.Errorf("track ID is required for enrich action")
		}
		return enrichTrack(ctx, *f.trackID, s.tracks, s.ai)

	case "validate":
		if *f.trackID == "" {
			return fmt.Errorf("track ID is required for validate action")
		}
		return validateTrack(ctx, *f.trackID, s.tracks, s.ai)

	case "export":
		if *f.trackID == "" && *f.batchFile == "" {
			return fmt.Errorf("either track ID or batch file is required for export action")
		}
		return exportTracks(ctx, *f.trackID, *f.batchFile, *f.format, s.tracks, s.ddex)

	default:
		printUsage()
		return fmt.Errorf("invalid action: %s", *f.action)
	}
}

// printUsage prints the CLI usage information
func printUsage() {
	fmt.Println("Available commands:")
	fmt.Println("  metadatatool -action=enrich -track=<track_id>")
	fmt.Println("  metadatatool -action=validate -track=<track_id>")
	fmt.Println("  metadatatool -action=export -track=<track_id> -format=[json|ddex]")
	fmt.Println("  metadatatool -action=export -batch=<file> -format=[json|ddex]")
}

// enrichTrack enriches a track's metadata using AI services
func enrichTrack(ctx context.Context, trackID string, repo domain.TrackRepository, ai domain.AIService) error {
	track, err := repo.GetByID(ctx, trackID)
	if err != nil {
		return fmt.Errorf("failed to get track: %w", err)
	}
	return ai.EnrichMetadata(ctx, track)
}

// validateTrack validates a track's metadata using AI services
func validateTrack(ctx context.Context, trackID string, repo domain.TrackRepository, ai domain.AIService) error {
	track, err := repo.GetByID(ctx, trackID)
	if err != nil {
		return fmt.Errorf("failed to get track: %w", err)
	}
	confidence, err := ai.ValidateMetadata(ctx, track)
	if err != nil {
		return err
	}
	fmt.Printf("Track validation confidence: %.2f\n", confidence)
	return nil
}

// exportTracks exports tracks in the specified format
func exportTracks(ctx context.Context, trackID, batchFile, format string, repo domain.TrackRepository, ddex domain.DDEXService) error {
	var tracks []*domain.Track

	if trackID != "" {
		track, err := repo.GetByID(ctx, trackID)
		if err != nil {
			return fmt.Errorf("failed to get track: %w", err)
		}
		tracks = append(tracks, track)
	} else if batchFile != "" {
		// Read track IDs from batch file
		content, err := os.ReadFile(batchFile)
		if err != nil {
			return fmt.Errorf("failed to read batch file: %w", err)
		}

		// Parse track IDs (assuming one ID per line)
		trackIDs := strings.Split(strings.TrimSpace(string(content)), "\n")
		for _, id := range trackIDs {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			track, err := repo.GetByID(ctx, id)
			if err != nil {
				return fmt.Errorf("failed to get track %s: %w", id, err)
			}
			if track != nil {
				tracks = append(tracks, track)
			}
		}
	} else {
		return fmt.Errorf("either track ID or batch file is required")
	}

	if len(tracks) == 0 {
		return fmt.Errorf("no tracks found to export")
	}

	if format == "ddex" {
		output, err := ddex.ExportTracks(ctx, tracks)
		if err != nil {
			return fmt.Errorf("failed to export tracks to DDEX: %w", err)
		}
		fmt.Println(output)
	} else if format == "json" {
		// Implement JSON export
		jsonData, err := json.MarshalIndent(tracks, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal tracks to JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		return fmt.Errorf("unsupported format: %s", format)
	}

	return nil
}
