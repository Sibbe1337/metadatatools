package handler

import (
	"fmt"
	"metadatatool/internal/pkg/domain"
	"path/filepath"
	"time"

	"metadatatool/internal/pkg/metrics"

	"github.com/gin-gonic/gin"
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
	timer := metrics.NewTimer(metrics.AudioOpDurations.WithLabelValues("upload"))
	defer timer.ObserveDuration()

	metrics.AudioOps.WithLabelValues("upload", "started").Inc()

	file, err := c.FormFile("file")
	if err != nil {
		metrics.AudioOpErrors.WithLabelValues("upload", "form_error").Inc()
		c.JSON(400, gin.H{"error": "No file provided"})
		return
	}

	// Generate storage key
	key := fmt.Sprintf("audio/%s/%s", time.Now().Format("2006/01/02"), file.Filename)

	// Open the uploaded file
	src, err := file.Open()
	if err != nil {
		metrics.AudioOpErrors.WithLabelValues("upload", "open_error").Inc()
		c.JSON(500, gin.H{"error": "Failed to open file"})
		return
	}
	defer src.Close()

	// Create storage file
	storageFile := &domain.StorageFile{
		Key:         key,
		Name:        file.Filename,
		Size:        file.Size,
		ContentType: file.Header.Get("Content-Type"),
		Content:     src,
	}

	// Upload to storage
	if err := h.storage.Upload(c, storageFile); err != nil {
		metrics.AudioOpErrors.WithLabelValues("upload", "storage_error").Inc()
		c.JSON(500, gin.H{"error": "Failed to upload file"})
		return
	}

	// Create track record
	track := &domain.Track{
		StoragePath: key,
		FileSize:    file.Size,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata: domain.CompleteTrackMetadata{
			BasicTrackMetadata: domain.BasicTrackMetadata{
				Title:     filepath.Base(file.Filename),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Technical: domain.AudioTechnicalMetadata{
				Format:   domain.AudioFormat(filepath.Ext(file.Filename)[1:]),
				FileSize: file.Size,
			},
		},
	}

	if err := h.trackRepo.Create(c, track); err != nil {
		metrics.AudioOpErrors.WithLabelValues("upload", "db_error").Inc()
		c.JSON(500, gin.H{"error": "Failed to create track record"})
		return
	}

	metrics.AudioOps.WithLabelValues("upload", "completed").Inc()
	c.JSON(200, gin.H{
		"id":  track.ID,
		"url": key,
	})
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
	timer := metrics.NewTimer(metrics.AudioOpDurations.WithLabelValues("get_url"))
	defer timer.ObserveDuration()

	metrics.AudioOps.WithLabelValues("get_url", "started").Inc()

	id := c.Param("id")
	track, err := h.trackRepo.GetByID(c, id)
	if err != nil {
		metrics.AudioOpErrors.WithLabelValues("get_url", "not_found").Inc()
		c.JSON(404, gin.H{"error": "Track not found"})
		return
	}

	url, err := h.storage.GetURL(c, track.StoragePath)
	if err != nil {
		metrics.AudioOpErrors.WithLabelValues("get_url", "storage_error").Inc()
		c.JSON(500, gin.H{"error": "Failed to generate URL"})
		return
	}

	metrics.AudioOps.WithLabelValues("get_url", "completed").Inc()
	c.JSON(200, gin.H{"url": url})
}

// Helper functions moved to audio_utils.go
