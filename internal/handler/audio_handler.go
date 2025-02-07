package handler

import (
	"fmt"
	"metadatatool/internal/pkg/domain"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AudioHandler struct {
	storage   domain.StorageService
	trackRepo domain.TrackRepository
}

// NewAudioHandler creates a new audio handler
func NewAudioHandler(storage domain.StorageService, trackRepo domain.TrackRepository) *AudioHandler {
	return &AudioHandler{
		storage:   storage,
		trackRepo: trackRepo,
	}
}

// UploadAudio handles audio file upload
// @Summary Upload audio file
// @Description Upload an audio file and store it in cloud storage
// @Tags audio
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Audio file"
// @Success 201 {object} domain.Track
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /audio/upload [post]
func (h *AudioHandler) UploadAudio(c *gin.Context) {
	// Get the file
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid file upload",
		})
		return
	}

	// Validate file type
	if !isValidAudioFormat(file.Filename) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid audio format",
		})
		return
	}

	// Generate unique filename
	filename := fmt.Sprintf("%s%s", uuid.New().String(), filepath.Ext(file.Filename))
	key := filepath.Join("audio", filename)

	// Open the file
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to read file",
		})
		return
	}
	defer src.Close()

	// Upload to storage
	if err := h.storage.Upload(c, key, src); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to upload file",
		})
		return
	}

	// Create track record
	track := &domain.Track{
		Title:       file.Filename,
		FilePath:    key,
		AudioFormat: getAudioFormat(file.Filename),
		FileSize:    file.Size,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := h.trackRepo.Create(c, track); err != nil {
		// Try to clean up the uploaded file
		_ = h.storage.Delete(c, key)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create track record",
		})
		return
	}

	c.JSON(http.StatusCreated, track)
}

// GetAudioURL generates a pre-signed URL for audio download
// @Summary Get audio download URL
// @Description Get a pre-signed URL for downloading an audio file
// @Tags audio
// @Produce json
// @Param id path string true "Track ID"
// @Success 200 {object} map[string]string
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /audio/{id} [get]
func (h *AudioHandler) GetAudioURL(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "missing track ID",
		})
		return
	}

	// Get track
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

	// Generate pre-signed URL
	url, err := h.storage.GetSignedURL(c, track.FilePath, domain.SignedURLDownload, 15*time.Minute)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to generate download URL",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url": url,
	})
}

// Helper functions moved to audio_utils.go
