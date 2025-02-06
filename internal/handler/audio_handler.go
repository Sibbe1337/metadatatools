package handler

import (
	"metadatatool/internal/pkg/domain"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AudioHandler struct {
	audioService domain.AudioService
}

func NewAudioHandler(audioService domain.AudioService) *AudioHandler {
	return &AudioHandler{
		audioService: audioService,
	}
}

// UploadAudio handles audio file uploads
func (h *AudioHandler) UploadAudio(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "no file uploaded",
		})
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to read file",
		})
		return
	}
	defer src.Close()

	audioFile := &domain.File{
		Name:        file.Filename,
		Size:        file.Size,
		ContentType: file.Header.Get("Content-Type"),
		Content:     src,
	}

	url, err := h.audioService.Upload(c, audioFile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to upload file",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url": url,
	})
}

// GetAudioURL retrieves a pre-signed URL for an audio file
func (h *AudioHandler) GetAudioURL(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "missing file ID",
		})
		return
	}

	url, err := h.audioService.GetURL(c, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get file URL",
		})
		return
	}

	if url == "" {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "file not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url": url,
	})
}
