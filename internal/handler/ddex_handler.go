package handler

import (
	"encoding/xml"
	"metadatatool/internal/pkg/domain"
	"net/http"

	"github.com/gin-gonic/gin"
)

type DDEXHandler struct {
	trackRepo domain.TrackRepository
}

// NewDDEXHandler creates a new DDEX handler
func NewDDEXHandler(trackRepo domain.TrackRepository) *DDEXHandler {
	return &DDEXHandler{
		trackRepo: trackRepo,
	}
}

// ValidateERN validates a DDEX ERN file
// @Summary Validate DDEX ERN
// @Description Validate a DDEX ERN XML file
// @Tags ddex
// @Accept xml
// @Produce json
// @Param file formData file true "DDEX ERN XML file"
// @Success 200 {object} ValidationResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /ddex/validate [post]
func (h *DDEXHandler) ValidateERN(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid file upload",
		})
		return
	}

	// Open and read the file
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to read file",
		})
		return
	}
	defer src.Close()

	// Parse XML
	var ern domain.ERNMessage
	if err := xml.NewDecoder(src).Decode(&ern); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid ERN XML format",
		})
		return
	}

	// Validate ERN
	valid, errors := validateERN(&ern)
	c.JSON(http.StatusOK, ValidationResponse{
		Valid:  valid,
		Errors: errors,
	})
}

// ImportERN imports a DDEX ERN file
// @Summary Import DDEX ERN
// @Description Import tracks from a DDEX ERN XML file
// @Tags ddex
// @Accept xml
// @Produce json
// @Param file formData file true "DDEX ERN XML file"
// @Success 201 {array} domain.Track
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /ddex/import [post]
func (h *DDEXHandler) ImportERN(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid file upload",
		})
		return
	}

	// Open and read the file
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to read file",
		})
		return
	}
	defer src.Close()

	// Parse XML
	var ern domain.ERNMessage
	if err := xml.NewDecoder(src).Decode(&ern); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid ERN XML format",
		})
		return
	}

	// Convert ERN to tracks
	tracks := convertERNToTracks(&ern)

	// Save tracks
	var savedTracks []*domain.Track
	for _, track := range tracks {
		if err := h.trackRepo.Create(c, track); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to save track",
			})
			return
		}
		savedTracks = append(savedTracks, track)
	}

	c.JSON(http.StatusCreated, savedTracks)
}

// ExportERN exports tracks as a DDEX ERN file
// @Summary Export DDEX ERN
// @Description Export tracks as a DDEX ERN XML file
// @Tags ddex
// @Produce xml
// @Success 200 {string} string "ERN XML file"
// @Failure 500 {object} ErrorResponse
// @Router /ddex/export [post]
func (h *DDEXHandler) ExportERN(c *gin.Context) {
	// Get all tracks
	tracks, err := h.trackRepo.List(c, 0, 1000) // TODO: Add pagination
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get tracks",
		})
		return
	}

	// Convert tracks to ERN
	ern := convertTracksToERN(tracks)

	// Marshal to XML
	xmlData, err := xml.MarshalIndent(ern, "", "  ")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to generate XML",
		})
		return
	}

	// Set headers
	c.Header("Content-Type", "application/xml")
	c.Header("Content-Disposition", "attachment; filename=export.xml")
	c.String(http.StatusOK, xml.Header+string(xmlData))
}

type ValidationResponse struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}

func validateERN(ern *domain.ERNMessage) (bool, []string) {
	var errors []string

	// Validate MessageHeader
	if ern.MessageHeader.MessageID == "" {
		errors = append(errors, "missing MessageID")
	}
	if ern.MessageHeader.MessageSender == "" {
		errors = append(errors, "missing MessageSender")
	}
	if ern.MessageHeader.MessageRecipient == "" {
		errors = append(errors, "missing MessageRecipient")
	}

	// Validate ResourceList
	for _, track := range ern.ResourceList.SoundRecordings {
		if track.ISRC == "" {
			errors = append(errors, "missing ISRC")
		}
		if track.Title.TitleText == "" {
			errors = append(errors, "missing Title")
		}
	}

	return len(errors) == 0, errors
}

func convertERNToTracks(ern *domain.ERNMessage) []*domain.Track {
	var tracks []*domain.Track

	for _, recording := range ern.ResourceList.SoundRecordings {
		track := &domain.Track{
			Title: recording.Title.TitleText,
			ISRC:  recording.ISRC,
			// Add more fields as needed
		}
		tracks = append(tracks, track)
	}

	return tracks
}

func convertTracksToERN(tracks []*domain.Track) *domain.ERNMessage {
	ern := &domain.ERNMessage{
		MessageHeader: domain.MessageHeader{
			MessageID:              "MSG001", // TODO: Generate unique ID
			MessageSender:          "YourCompany",
			MessageRecipient:       "DSP",
			MessageCreatedDateTime: "2024-04-20T12:00:00Z",
		},
	}

	// Add resources
	for _, track := range tracks {
		recording := domain.SoundRecording{
			ISRC:  track.ISRC,
			Title: domain.Title{TitleText: track.Title},
			// Add more fields as needed
		}
		ern.ResourceList.SoundRecordings = append(ern.ResourceList.SoundRecordings, recording)
	}

	return ern
}
