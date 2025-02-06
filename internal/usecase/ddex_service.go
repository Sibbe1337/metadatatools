package usecase

import (
	"context"
	"encoding/xml"
	"fmt"
	"metadatatool/internal/pkg/domain"
	"time"

	"github.com/google/uuid"
)

type ddexService struct {
	// Configuration could be added here if needed
}

// NewDDEXService creates a new DDEX service instance
func NewDDEXService() domain.DDEXService {
	return &ddexService{}
}

// ValidateTrack validates track metadata against DDEX schema
func (s *ddexService) ValidateTrack(ctx context.Context, track *domain.Track) (bool, []string) {
	var validationErrors []string

	// Validate required fields
	if track.ISRC == "" {
		validationErrors = append(validationErrors, "ISRC is required")
	} else if len(track.ISRC) != 12 {
		validationErrors = append(validationErrors, "ISRC must be 12 characters")
	}

	if track.Title == "" {
		validationErrors = append(validationErrors, "Title is required")
	}

	if track.Artist == "" {
		validationErrors = append(validationErrors, "Artist is required")
	}

	if track.Label == "" {
		validationErrors = append(validationErrors, "Label is required")
	}

	if track.Territory == "" {
		validationErrors = append(validationErrors, "Territory is required")
	} else if len(track.Territory) != 2 {
		validationErrors = append(validationErrors, "Territory must be a 2-letter ISO country code")
	}

	// Validate format-specific fields
	if track.Year < 1900 || track.Year > time.Now().Year() {
		validationErrors = append(validationErrors, "Invalid release year")
	}

	return len(validationErrors) == 0, validationErrors
}

// ExportTrack exports a single track to DDEX format
func (s *ddexService) ExportTrack(ctx context.Context, track *domain.Track) (string, error) {
	// Create DDEX ERN 4.3 message
	message := s.createERNMessage([]*domain.Track{track})

	// Marshal to XML
	output, err := xml.MarshalIndent(message, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal ERN: %w", err)
	}

	// Add XML header
	xmlHeader := []byte(xml.Header)
	result := append(xmlHeader, output...)
	return string(result), nil
}

// ExportTracks exports multiple tracks to DDEX format
func (s *ddexService) ExportTracks(ctx context.Context, tracks []*domain.Track) (string, error) {
	// Create DDEX ERN 4.3 message
	message := s.createERNMessage(tracks)

	// Marshal to XML
	output, err := xml.MarshalIndent(message, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal ERN: %w", err)
	}

	// Add XML header
	xmlHeader := []byte(xml.Header)
	result := append(xmlHeader, output...)
	return string(result), nil
}

// createERNMessage creates a DDEX ERN 4.3 message from tracks
func (s *ddexService) createERNMessage(tracks []*domain.Track) *domain.DDEXERN43Message {
	messageId := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	// Create message
	message := &domain.DDEXERN43Message{
		XMLNs:                  "http://ddex.net/xml/ern/43",
		XMLNsErn:               "http://ddex.net/xml/ern/43",
		MessageSchemaVersionId: "ern/43",
		MessageHeader: domain.MessageHeader{
			MessageID:          messageId,
			MessageSender:      "MetadataTool",
			MessageRecipient:   "DSP",
			MessageCreatedDate: now,
			MessageControlType: "LiveMessage",
		},
	}

	// Add resources
	var soundRecordings []domain.SoundRecording
	for _, track := range tracks {
		soundRecording := domain.SoundRecording{
			ResourceId: domain.ResourceId{
				ISRC: track.ISRC,
			},
			Title: domain.Title{
				TitleText: track.Title,
			},
			Duration: fmt.Sprintf("PT%dS", int(track.Duration)),
			TechnicalDetails: domain.TechnicalDetails{
				TechnicalResourceDetailsReference: track.ID,
				Audio: domain.Audio{
					Format:     "AAC",
					BitRate:    320,
					SampleRate: 44100,
				},
			},
			SoundRecordingType: "MusicalWorkSoundRecording",
			ResourceReference:  track.ID,
		}
		soundRecordings = append(soundRecordings, soundRecording)
	}
	message.ResourceList.SoundRecordings = soundRecordings

	// Add releases
	var releases []domain.Release
	for _, track := range tracks {
		release := domain.Release{
			ReleaseId: domain.ReleaseId{
				ICPN: track.ID, // Use track ID as ICPN for now
			},
			ReferenceTitle: domain.Title{
				TitleText: track.Title,
			},
			ReleaseType: "Single",
		}
		releases = append(releases, release)
	}
	message.ReleaseList.Releases = releases

	// Add deals
	var releaseDeals []domain.ReleaseDeal
	for _, track := range tracks {
		releaseDeal := domain.ReleaseDeal{
			DealReleaseReference: track.ID,
			Deal: domain.Deal{
				Territory: domain.Territory{
					TerritoryCode: track.Territory,
				},
				DealTerms: domain.DealTerms{
					CommercialModelType: "PayAsYouGoModel",
					Usage: domain.Usage{
						UseType: "OnDemandStream",
					},
				},
			},
		}
		releaseDeals = append(releaseDeals, releaseDeal)
	}
	message.DealList.ReleaseDeals = releaseDeals

	return message
}
