package handler

import (
	"fmt"
	"metadatatool/internal/pkg/domain"
	apperrors "metadatatool/internal/pkg/errors"
	"metadatatool/internal/pkg/errortracking"
	"metadatatool/internal/pkg/metrics"
	"metadatatool/internal/pkg/utils"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TrackHandler handles HTTP requests for track operations
type TrackHandler struct {
	trackRepo    domain.TrackRepository
	aiService    domain.AIService
	validator    domain.Validator
	errorTracker *errortracking.ErrorTracker
	storageRoot  string
}

// NewTrackHandler creates a new track handler
func NewTrackHandler(
	trackRepo domain.TrackRepository,
	aiService domain.AIService,
	validator domain.Validator,
	errorTracker *errortracking.ErrorTracker,
	storageRoot string,
) *TrackHandler {
	return &TrackHandler{
		trackRepo:    trackRepo,
		aiService:    aiService,
		validator:    validator,
		errorTracker: errorTracker,
		storageRoot:  storageRoot,
	}
}

// UploadTrack handles track file upload and metadata creation
// @Summary Upload new track
// @Description Upload an audio file and create track metadata
// @Tags tracks
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Audio file"
// @Param title formData string true "Track title"
// @Param artist formData string true "Artist name"
// @Success 201 {object} domain.Track
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /tracks/upload [post]
func (h *TrackHandler) UploadTrack(c *gin.Context) {
	start := time.Now()
	defer func() {
		metrics.DatabaseOperationsTotal.WithLabelValues("upload", "total").Inc()
		metrics.DatabaseQueryDuration.WithLabelValues("upload").Observe(time.Since(start).Seconds())
	}()

	file, err := c.FormFile("file")
	if err != nil {
		h.handleError(c, apperrors.NewValidationError("invalid file upload", err.Error()))
		return
	}

	if !utils.IsValidAudioFormat(file.Filename) {
		h.handleError(c, apperrors.NewValidationError("invalid audio format", "unsupported file format"))
		return
	}

	filename := fmt.Sprintf("%s%s", uuid.New().String(), filepath.Ext(file.Filename))
	filepath := filepath.Join(h.storageRoot, filename)

	if err := c.SaveUploadedFile(file, filepath); err != nil {
		h.handleError(c, apperrors.NewStorageError("failed to save file", err))
		return
	}

	track := &domain.Track{
		ID:          uuid.New().String(),
		StoragePath: filepath,
		FileSize:    file.Size,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Status:      domain.TrackStatusPending,
		Metadata: domain.CompleteTrackMetadata{
			BasicTrackMetadata: domain.BasicTrackMetadata{
				Title:     c.PostForm("title"),
				Artist:    c.PostForm("artist"),
				Album:     c.PostForm("album"),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Technical: domain.AudioTechnicalMetadata{
				Format:   domain.AudioFormat(utils.GetAudioFormat(file.Filename)),
				FileSize: file.Size,
			},
		},
	}

	// Validate track
	result := h.validator.Validate(track)
	if !result.Valid {
		details := make([]string, len(result.Issues))
		for i, issue := range result.Issues {
			details[i] = fmt.Sprintf("%s: %s", issue.Field, issue.Message)
		}
		h.handleError(c, apperrors.NewValidationError("invalid track data", strings.Join(details, "; ")))
		return
	}

	if err := h.trackRepo.Create(c, track); err != nil {
		h.handleError(c, apperrors.NewDatabaseError("failed to create track", err))
		return
	}

	// Trigger async AI processing
	go func() {
		if err := h.aiService.EnrichMetadata(c, track); err != nil {
			h.errorTracker.CaptureError(err, map[string]string{
				"operation": "ai_enrich",
				"track_id":  track.ID,
			})
		}
	}()

	c.JSON(http.StatusCreated, track)
}

// CreateTrack handles track creation requests
// @Summary Create track
// @Description Create a new track with metadata
// @Tags tracks
// @Accept json
// @Produce json
// @Param track body domain.Track true "Track object"
// @Success 201 {object} domain.Track
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /tracks [post]
func (h *TrackHandler) CreateTrack(c *gin.Context) {
	start := time.Now()
	defer func() {
		metrics.DatabaseOperationsTotal.WithLabelValues("create", "total").Inc()
		metrics.DatabaseQueryDuration.WithLabelValues("create").Observe(time.Since(start).Seconds())
	}()

	var track domain.Track
	if err := c.ShouldBindJSON(&track); err != nil {
		h.handleError(c, apperrors.NewValidationError("invalid request body", err.Error()))
		return
	}

	// Set default values
	track.ID = uuid.New().String()
	track.CreatedAt = time.Now()
	track.UpdatedAt = time.Now()
	track.Status = domain.TrackStatusPending

	// Validate track
	result := h.validator.Validate(&track)
	if !result.Valid {
		details := make([]string, len(result.Issues))
		for i, issue := range result.Issues {
			details[i] = fmt.Sprintf("%s: %s", issue.Field, issue.Message)
		}
		h.handleError(c, apperrors.NewValidationError("invalid track data", strings.Join(details, "; ")))
		return
	}

	if err := h.trackRepo.Create(c, &track); err != nil {
		h.handleError(c, apperrors.NewDatabaseError("failed to create track", err))
		return
	}

	c.JSON(http.StatusCreated, track)
}

// GetTrack retrieves a track by ID
// @Summary Get track
// @Description Get a track by ID
// @Tags tracks
// @Produce json
// @Param id path string true "Track ID"
// @Success 200 {object} domain.Track
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /tracks/{id} [get]
func (h *TrackHandler) GetTrack(c *gin.Context) {
	start := time.Now()
	defer func() {
		metrics.DatabaseOperationsTotal.WithLabelValues("get", "total").Inc()
		metrics.DatabaseQueryDuration.WithLabelValues("get").Observe(time.Since(start).Seconds())
	}()

	id := c.Param("id")
	if id == "" {
		h.handleError(c, apperrors.NewValidationError("missing track ID", "track ID is required"))
		return
	}

	track, err := h.trackRepo.GetByID(c, id)
	if err != nil {
		h.handleError(c, apperrors.NewDatabaseError("failed to get track", err))
		return
	}

	if track == nil {
		h.handleError(c, apperrors.NewNotFoundError("track not found"))
		return
	}

	c.JSON(http.StatusOK, track)
}

// UpdateTrack modifies an existing track
// @Summary Update track
// @Description Update an existing track
// @Tags tracks
// @Accept json
// @Produce json
// @Param id path string true "Track ID"
// @Param track body domain.Track true "Track object"
// @Success 200 {object} domain.Track
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /tracks/{id} [put]
func (h *TrackHandler) UpdateTrack(c *gin.Context) {
	start := time.Now()
	defer func() {
		metrics.DatabaseOperationsTotal.WithLabelValues("update", "total").Inc()
		metrics.DatabaseQueryDuration.WithLabelValues("update").Observe(time.Since(start).Seconds())
	}()

	id := c.Param("id")
	if id == "" {
		h.handleError(c, apperrors.NewValidationError("missing track ID", "track ID is required"))
		return
	}

	// Get existing track
	existingTrack, err := h.trackRepo.GetByID(c, id)
	if err != nil {
		h.handleError(c, apperrors.NewDatabaseError("failed to get track", err))
		return
	}

	if existingTrack == nil {
		h.handleError(c, apperrors.NewNotFoundError("track not found"))
		return
	}

	// Parse update data
	var updateData domain.Track
	if err := c.ShouldBindJSON(&updateData); err != nil {
		h.handleError(c, apperrors.NewValidationError("invalid request body", err.Error()))
		return
	}

	// Apply updates while preserving certain fields
	updateData.ID = id
	updateData.CreatedAt = existingTrack.CreatedAt
	updateData.UpdatedAt = time.Now()
	updateData.StoragePath = existingTrack.StoragePath
	updateData.FileSize = existingTrack.FileSize

	// Validate updated track
	result := h.validator.Validate(&updateData)
	if !result.Valid {
		details := make([]string, len(result.Issues))
		for i, issue := range result.Issues {
			details[i] = fmt.Sprintf("%s: %s", issue.Field, issue.Message)
		}
		h.handleError(c, apperrors.NewValidationError("invalid track data", strings.Join(details, "; ")))
		return
	}

	// Increment version
	updateData.Version = existingTrack.Version + 1
	updateData.PreviousID = existingTrack.ID

	if err := h.trackRepo.Update(c, &updateData); err != nil {
		h.handleError(c, apperrors.NewDatabaseError("failed to update track", err))
		return
	}

	c.JSON(http.StatusOK, updateData)
}

// DeleteTrack removes a track
// @Summary Delete track
// @Description Delete a track by ID
// @Tags tracks
// @Produce json
// @Param id path string true "Track ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /tracks/{id} [delete]
func (h *TrackHandler) DeleteTrack(c *gin.Context) {
	start := time.Now()
	defer func() {
		metrics.DatabaseOperationsTotal.WithLabelValues("delete", "total").Inc()
		metrics.DatabaseQueryDuration.WithLabelValues("delete").Observe(time.Since(start).Seconds())
	}()

	id := c.Param("id")
	if id == "" {
		h.handleError(c, apperrors.NewValidationError("missing track ID", "track ID is required"))
		return
	}

	// Get existing track
	existingTrack, err := h.trackRepo.GetByID(c, id)
	if err != nil {
		h.handleError(c, apperrors.NewDatabaseError("failed to get track", err))
		return
	}

	if existingTrack == nil {
		h.handleError(c, apperrors.NewNotFoundError("track not found"))
		return
	}

	if err := h.trackRepo.Delete(c, id); err != nil {
		h.handleError(c, apperrors.NewDatabaseError("failed to delete track", err))
		return
	}

	c.Status(http.StatusNoContent)
}

// ListTracks retrieves a paginated list of tracks
// @Summary List tracks
// @Description Get a paginated list of tracks
// @Tags tracks
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Success 200 {object} ListResponse
// @Failure 500 {object} ErrorResponse
// @Router /tracks [get]
func (h *TrackHandler) ListTracks(c *gin.Context) {
	start := time.Now()
	defer func() {
		metrics.DatabaseOperationsTotal.WithLabelValues("list", "total").Inc()
		metrics.DatabaseQueryDuration.WithLabelValues("list").Observe(time.Since(start).Seconds())
	}()

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit

	tracks, err := h.trackRepo.List(c, map[string]interface{}{}, offset, limit)
	if err != nil {
		h.handleError(c, apperrors.NewDatabaseError("failed to list tracks", err))
		return
	}

	c.JSON(http.StatusOK, ListResponse{
		Tracks: tracks,
		Page:   page,
		Limit:  limit,
	})
}

// SearchTracks searches tracks by metadata
// @Summary Search tracks
// @Description Search tracks by metadata fields
// @Tags tracks
// @Accept json
// @Produce json
// @Param query body SearchQuery true "Search query"
// @Success 200 {array} domain.Track
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /tracks/search [post]
func (h *TrackHandler) SearchTracks(c *gin.Context) {
	start := time.Now()
	defer func() {
		metrics.DatabaseOperationsTotal.WithLabelValues("search", "total").Inc()
		metrics.DatabaseQueryDuration.WithLabelValues("search").Observe(time.Since(start).Seconds())
	}()

	var query SearchQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		h.handleError(c, apperrors.NewValidationError("invalid search query", err.Error()))
		return
	}

	tracks, err := h.trackRepo.SearchByMetadata(c, query.toMap())
	if err != nil {
		h.handleError(c, apperrors.NewDatabaseError("failed to search tracks", err))
		return
	}

	c.JSON(http.StatusOK, tracks)
}

// BatchProcess processes multiple tracks
func (h *TrackHandler) BatchProcess(c *gin.Context) {
	start := time.Now()
	defer func() {
		metrics.DatabaseOperationsTotal.WithLabelValues("batch", "total").Inc()
		metrics.DatabaseQueryDuration.WithLabelValues("batch").Observe(time.Since(start).Seconds())
	}()

	var req BatchProcessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.handleError(c, apperrors.NewValidationError("invalid request body", err.Error()))
		return
	}

	var tracks []*domain.Track
	for _, id := range req.TrackIDs {
		track, err := h.trackRepo.GetByID(c, id)
		if err != nil {
			h.handleError(c, apperrors.NewDatabaseError("failed to get track", err))
			return
		}
		if track == nil {
			h.handleError(c, apperrors.NewNotFoundError(fmt.Sprintf("track not found: %s", id)))
			return
		}
		tracks = append(tracks, track)
	}

	// Process tracks in batch
	if err := h.aiService.BatchProcess(c, tracks); err != nil {
		h.handleError(c, apperrors.NewAIError("failed to process tracks", err))
		return
	}

	// Update tracks in database
	if err := h.trackRepo.BatchUpdate(c, tracks); err != nil {
		h.handleError(c, apperrors.NewDatabaseError("failed to update tracks", err))
		return
	}

	c.JSON(http.StatusOK, tracks)
}

// ExportTracks exports tracks in the specified format
// @Summary Export tracks
// @Description Export tracks in the specified format
// @Tags tracks
// @Accept json
// @Produce json
// @Param request body ExportRequest true "Export request"
// @Success 200 {object} ExportResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /tracks/export [post]
func (h *TrackHandler) ExportTracks(c *gin.Context) {
	var req ExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.handleError(c, apperrors.NewValidationError("invalid request body", err.Error()))
		return
	}

	// Get tracks
	var tracks []*domain.Track
	for _, id := range req.TrackIDs {
		track, err := h.trackRepo.GetByID(c, id)
		if err != nil {
			h.handleError(c, apperrors.NewDatabaseError("failed to get track", err))
			return
		}
		if track != nil {
			tracks = append(tracks, track)
		}
	}

	// Generate export data based on format
	var exportData interface{}
	switch req.Format {
	case "json":
		exportData = tracks
	case "csv":
		csvData := [][]string{
			{"ID", "Title", "Artist", "Album", "ISRC", "Duration", "Created At"},
		}
		for _, track := range tracks {
			row := []string{
				track.ID,
				track.Title(),
				track.Artist(),
				track.Album(),
				track.ISRC(),
				fmt.Sprintf("%d", int(track.Duration())),
				track.CreatedAt.Format(time.RFC3339),
			}
			csvData = append(csvData, row)
		}
		exportData = csvData
	default:
		h.handleError(c, apperrors.NewValidationError("unsupported export format", fmt.Sprintf("format '%s' is not supported", req.Format)))
		return
	}

	c.JSON(http.StatusOK, ExportResponse{
		Format: req.Format,
		Data:   exportData,
	})
}

// Helper functions and types

type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

type ListResponse struct {
	Tracks []*domain.Track `json:"tracks"`
	Page   int             `json:"page"`
	Limit  int             `json:"limit"`
}

type SearchQuery struct {
	Title       string    `json:"title,omitempty"`
	Artist      string    `json:"artist,omitempty"`
	Album       string    `json:"album,omitempty"`
	Genre       string    `json:"genre,omitempty"`
	Label       string    `json:"label,omitempty"`
	ISRC        string    `json:"isrc,omitempty"`
	ISWC        string    `json:"iswc,omitempty"`
	CreatedFrom time.Time `json:"created_from,omitempty"`
	CreatedTo   time.Time `json:"created_to,omitempty"`
	NeedsReview *bool     `json:"needs_review,omitempty"`
}

func (q *SearchQuery) toMap() map[string]interface{} {
	m := make(map[string]interface{})
	if q.Title != "" {
		m["title"] = q.Title
	}
	if q.Artist != "" {
		m["artist"] = q.Artist
	}
	if q.Album != "" {
		m["album"] = q.Album
	}
	if q.Genre != "" {
		m["genre"] = q.Genre
	}
	if q.Label != "" {
		m["label"] = q.Label
	}
	if q.ISRC != "" {
		m["isrc"] = q.ISRC
	}
	if q.ISWC != "" {
		m["iswc"] = q.ISWC
	}
	if !q.CreatedFrom.IsZero() {
		m["created_from"] = q.CreatedFrom
	}
	if !q.CreatedTo.IsZero() {
		m["created_to"] = q.CreatedTo
	}
	if q.NeedsReview != nil {
		m["needs_review"] = *q.NeedsReview
	}
	return m
}

func (h *TrackHandler) handleError(c *gin.Context, err *apperrors.AppError) {
	if h.errorTracker != nil {
		h.errorTracker.CaptureError(err, map[string]string{
			"handler": "track",
			"method":  c.Request.Method,
			"path":    c.Request.URL.Path,
		})
	}

	c.JSON(err.StatusCode, gin.H{
		"error": gin.H{
			"type":    err.Type,
			"message": err.Message,
			"details": err.Details,
		},
	})
}

func validateTrack(track *domain.Track) error {
	if track.Title() == "" {
		return fmt.Errorf("title is required")
	}
	if track.Artist() == "" {
		return fmt.Errorf("artist is required")
	}
	if track.ISRC() != "" && len(track.ISRC()) != 12 {
		return fmt.Errorf("ISRC must be 12 characters")
	}
	if track.ISWC() != "" && len(track.ISWC()) != 11 {
		return fmt.Errorf("ISWC must be 11 characters")
	}
	return nil
}

type BatchProcessRequest struct {
	TrackIDs []string `json:"track_ids" binding:"required"`
}

type ExportRequest struct {
	TrackIDs []string `json:"track_ids" binding:"required"`
	Format   string   `json:"format" binding:"required,oneof=json csv"`
}

type ExportResponse struct {
	Format string      `json:"format"`
	Data   interface{} `json:"data"`
}
