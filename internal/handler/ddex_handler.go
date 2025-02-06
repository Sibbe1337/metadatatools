package handler

import (
	"metadatatool/internal/pkg/domain"
	"net/http"

	"github.com/gin-gonic/gin"
)

type DDEXHandler struct {
	ddexService domain.DDEXService
	trackRepo   domain.TrackRepository
}

func NewDDEXHandler(ddexService domain.DDEXService, trackRepo domain.TrackRepository) *DDEXHandler {
	return &DDEXHandler{
		ddexService: ddexService,
		trackRepo:   trackRepo,
	}
}

// ValidateTrackDDEX validates a track's metadata against DDEX schema
func (h *DDEXHandler) ValidateTrackDDEX(c *gin.Context) {
	var track domain.Track
	if err := c.ShouldBindJSON(&track); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	valid, errors := h.ddexService.ValidateTrack(c, &track)
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{
			"valid":  false,
			"errors": errors,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid": true,
	})
}

// ExportTrackDDEX exports a track to DDEX format
func (h *DDEXHandler) ExportTrackDDEX(c *gin.Context) {
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

	ddex, err := h.ddexService.ExportTrack(c, track)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to export track to DDEX",
		})
		return
	}

	c.Header("Content-Type", "application/xml")
	c.String(http.StatusOK, ddex)
}

// BatchExportDDEX exports multiple tracks to DDEX format
func (h *DDEXHandler) BatchExportDDEX(c *gin.Context) {
	var trackIDs []string
	if err := c.ShouldBindJSON(&trackIDs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	tracks := make([]*domain.Track, 0, len(trackIDs))
	for _, id := range trackIDs {
		track, err := h.trackRepo.GetByID(c, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to get track",
			})
			return
		}
		if track != nil {
			tracks = append(tracks, track)
		}
	}

	if len(tracks) == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "no tracks found",
		})
		return
	}

	ddex, err := h.ddexService.ExportTracks(c, tracks)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to export tracks to DDEX",
		})
		return
	}

	c.Header("Content-Type", "application/xml")
	c.String(http.StatusOK, ddex)
}
