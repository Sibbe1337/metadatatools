package handler

import (
	"metadatatool/internal/pkg/domain"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type TrackHandler struct {
	trackRepo domain.TrackRepository
	aiService domain.AIService
}

func NewTrackHandler(trackRepo domain.TrackRepository, aiService domain.AIService) *TrackHandler {
	return &TrackHandler{
		trackRepo: trackRepo,
		aiService: aiService,
	}
}

// CreateTrack handles track creation requests
func (h *TrackHandler) CreateTrack(c *gin.Context) {
	var track domain.Track
	if err := c.ShouldBindJSON(&track); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	// Create the track
	if err := h.trackRepo.Create(c, &track); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create track",
		})
		return
	}

	c.JSON(http.StatusCreated, track)
}

// GetTrack retrieves a track by ID
func (h *TrackHandler) GetTrack(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "missing track ID",
		})
		return
	}

	track, err := h.trackRepo.GetByID(c, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get track",
		})
		return
	}

	if track == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "track not found",
		})
		return
	}

	c.JSON(http.StatusOK, track)
}

// UpdateTrack modifies an existing track
func (h *TrackHandler) UpdateTrack(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "missing track ID",
		})
		return
	}

	var track domain.Track
	if err := c.ShouldBindJSON(&track); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	track.ID = id

	if err := h.trackRepo.Update(c, &track); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to update track",
		})
		return
	}

	c.JSON(http.StatusOK, track)
}

// DeleteTrack removes a track
func (h *TrackHandler) DeleteTrack(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "missing track ID",
		})
		return
	}

	if err := h.trackRepo.Delete(c, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to delete track",
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// ListTracks retrieves a paginated list of tracks
func (h *TrackHandler) ListTracks(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit

	tracks, err := h.trackRepo.List(c, offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to list tracks",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tracks": tracks,
		"page":   page,
		"limit":  limit,
	})
}

// EnrichTrack enriches track metadata using AI
func (h *TrackHandler) EnrichTrack(c *gin.Context) {
	var track domain.Track
	if err := c.ShouldBindJSON(&track); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	// Enrich metadata using AI service
	if err := h.aiService.EnrichMetadata(c, &track); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to enrich metadata",
		})
		return
	}

	// Save the track
	if err := h.trackRepo.Create(c, &track); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to save track",
		})
		return
	}

	c.JSON(http.StatusOK, track)
}

// ValidateTrack handles metadata validation for a single track
func (h *TrackHandler) ValidateTrack(c *gin.Context) {
	var track domain.Track
	if err := c.ShouldBindJSON(&track); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	// Validate metadata using AI
	confidence, err := h.aiService.ValidateMetadata(c, &track)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to validate metadata",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"track":        track,
		"confidence":   confidence,
		"needs_review": confidence < 0.85,
	})
}

// BatchProcess handles batch processing of tracks
func (h *TrackHandler) BatchProcess(c *gin.Context) {
	var tracks []*domain.Track
	if err := c.ShouldBindJSON(&tracks); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	// Process tracks in parallel
	if err := h.aiService.BatchProcess(c, tracks); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to process tracks",
		})
		return
	}

	// Save all tracks
	for _, track := range tracks {
		if err := h.trackRepo.Create(c, track); err != nil {
			// Log error but continue processing
			// In a production system, we might want to implement a retry mechanism
			continue
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Successfully processed tracks",
		"tracks":  tracks,
	})
}
