package usecase

import (
	"context"
	"encoding/xml"
	"fmt"
	"math"
	"metadatatool/internal/pkg/domain"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ddexService struct {
	schemaValidator SchemaValidator
	config          *DDEXConfig
}

// DDEXConfig holds configuration for the DDEX service
type DDEXConfig struct {
	MessageSender    string
	MessageRecipient string
	SchemaPath       string
	ValidateSchema   bool
}

// SchemaValidator defines the interface for XML schema validation
type SchemaValidator interface {
	ValidateAgainstSchema(xmlData []byte, schemaPath string) error
}

// NewDDEXService creates a new DDEX service instance
func NewDDEXService(validator SchemaValidator, config *DDEXConfig) domain.DDEXService {
	if config == nil {
		config = &DDEXConfig{
			MessageSender:    "MetadataTool",
			MessageRecipient: "DSP",
			ValidateSchema:   false,
		}
	}
	return &ddexService{
		schemaValidator: validator,
		config:          config,
	}
}

// ValidateTrack validates track metadata against DDEX schema
func (s *ddexService) ValidateTrack(ctx context.Context, track *domain.Track) (bool, []string) {
	var validationErrors []string

	// Required fields validation
	if err := s.validateRequiredFields(track); err != nil {
		validationErrors = append(validationErrors, err...)
	}

	// Format validation
	if err := s.validateFormats(track); err != nil {
		validationErrors = append(validationErrors, err...)
	}

	// Business rules validation
	if err := s.validateBusinessRules(track); err != nil {
		validationErrors = append(validationErrors, err...)
	}

	// Schema validation if enabled
	if s.config.ValidateSchema {
		xmlData, err := s.ExportTrack(ctx, track)
		if err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("Failed to generate XML: %v", err))
		} else if err := s.schemaValidator.ValidateAgainstSchema([]byte(xmlData), s.config.SchemaPath); err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("Schema validation failed: %v", err))
		}
	}

	return len(validationErrors) == 0, validationErrors
}

func (s *ddexService) validateRequiredFields(track *domain.Track) []string {
	var errors []string

	// Required field checks
	if track.ISRC() == "" {
		errors = append(errors, "ISRC is required")
	}
	if track.Title() == "" {
		errors = append(errors, "Title is required")
	}
	if track.Artist() == "" {
		errors = append(errors, "Artist is required")
	}
	if track.Label() == "" {
		errors = append(errors, "Label is required")
	}
	if track.Territory() == "" {
		errors = append(errors, "Territory is required")
	}

	return errors
}

func (s *ddexService) validateFormats(track *domain.Track) []string {
	var errors []string

	// ISRC format (CC-XXX-YY-NNNNN)
	if isrc := track.ISRC(); isrc != "" {
		isrcPattern := regexp.MustCompile(`^[A-Z]{2}[A-Z0-9]{3}\d{7}$`)
		if !isrcPattern.MatchString(isrc) {
			errors = append(errors, "Invalid ISRC format (should be CC-XXX-YY-NNNNN)")
		}
	}

	// Territory format (ISO 3166-1 alpha-2)
	if territory := track.Territory(); territory != "" {
		territoryPattern := regexp.MustCompile(`^[A-Z]{2}$`)
		if !territoryPattern.MatchString(territory) {
			errors = append(errors, "Territory must be a valid ISO 3166-1 alpha-2 code")
		}
	}

	// Audio format validation
	if format := track.AudioFormat(); format != "" {
		validFormats := map[string]bool{
			"AAC":  true,
			"MP3":  true,
			"FLAC": true,
			"WAV":  true,
		}
		if !validFormats[strings.ToUpper(format)] {
			errors = append(errors, "Unsupported audio format")
		}
	}

	return errors
}

func (s *ddexService) validateBusinessRules(track *domain.Track) []string {
	var errors []string
	now := time.Now()

	// Year validation
	if year := track.Year(); year != 0 {
		if year < 1900 || year > now.Year()+1 {
			errors = append(errors, "Year must be between 1900 and next year")
		}
	}

	// Duration validation
	if duration := track.Duration(); duration > 0 {
		if duration < 1 || duration > 7200 { // Max 2 hours
			errors = append(errors, "Duration must be between 1 second and 2 hours")
		}
	}

	// Technical metadata validation
	if bitrate := track.Bitrate(); bitrate > 0 {
		if bitrate < 32 || bitrate > 1411 {
			errors = append(errors, "Bitrate must be between 32 and 1411 kbps")
		}
	}

	if sampleRate := track.SampleRate(); sampleRate > 0 {
		validRates := map[int]bool{
			44100: true,
			48000: true,
			88200: true,
			96000: true,
		}
		if !validRates[sampleRate] {
			errors = append(errors, "Invalid sample rate")
		}
	}

	return errors
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

	// Add XML header and schema references
	xmlHeader := []byte(xml.Header)
	schemaRef := []byte(`<?xml-model href="http://ddex.net/xml/ern/43/release-notification.xsd" type="application/xml" schematypens="http://purl.oclc.org/dsdl/schematron"?>`)
	result := append(xmlHeader, append(schemaRef, output...)...)

	return string(result), nil
}

// ExportTracks exports multiple tracks to DDEX format
func (s *ddexService) ExportTracks(ctx context.Context, tracks []*domain.Track) (string, error) {
	// Validate all tracks first
	for _, track := range tracks {
		if valid, errors := s.ValidateTrack(ctx, track); !valid {
			return "", fmt.Errorf("track %s validation failed: %v", track.ID, errors)
		}
	}

	// Create DDEX ERN 4.3 message
	message := s.createERNMessage(tracks)

	// Marshal to XML
	output, err := xml.MarshalIndent(message, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal ERN: %w", err)
	}

	// Add XML header and schema references
	xmlHeader := []byte(xml.Header)
	schemaRef := []byte(`<?xml-model href="http://ddex.net/xml/ern/43/release-notification.xsd" type="application/xml" schematypens="http://purl.oclc.org/dsdl/schematron"?>`)
	result := append(xmlHeader, append(schemaRef, output...)...)

	return string(result), nil
}

// createERNMessage creates a DDEX ERN 4.3 message from tracks
func (s *ddexService) createERNMessage(tracks []*domain.Track) *domain.ERNMessage {
	messageId := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	// Create message
	message := &domain.ERNMessage{
		MessageHeader: domain.MessageHeader{
			MessageID:              messageId,
			MessageSender:          s.config.MessageSender,
			MessageRecipient:       s.config.MessageRecipient,
			MessageCreatedDateTime: now,
		},
	}

	// Add resources
	var soundRecordings []domain.SoundRecording
	for _, track := range tracks {
		// Convert duration to integer seconds
		durationSecs := int(math.Round(track.Duration()))

		soundRecording := domain.SoundRecording{
			ISRC: track.ISRC(),
			Title: domain.Title{
				TitleText: track.Title(),
			},
			Duration: fmt.Sprintf("PT%dS", durationSecs),
			TechnicalDetails: domain.TechnicalDetails{
				TechnicalResourceDetailsReference: track.ID,
				Audio: domain.Audio{
					Format:     strings.ToUpper(track.AudioFormat()),
					BitRate:    track.Bitrate(),
					SampleRate: track.SampleRate(),
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
			ReleaseID: domain.ReleaseID{
				ICPN: track.ID,
			},
			ReferenceTitle: domain.Title{
				TitleText: track.Title(),
			},
			ReleaseType: "Single",
		}
		releases = append(releases, release)
	}
	message.ReleaseList.Releases = releases

	// Add deals
	var deals []domain.ReleaseDeal
	for _, track := range tracks {
		deal := domain.ReleaseDeal{
			DealReleaseReference: track.ID,
			Deal: domain.Deal{
				Territory: domain.Territory{
					TerritoryCode: track.Territory(),
				},
				DealTerms: domain.DealTerms{
					CommercialModelType: "PayAsYouGoModel",
					Usage: domain.Usage{
						UseType: "OnDemandStream",
					},
				},
			},
		}
		deals = append(deals, deal)
	}
	message.DealList.ReleaseDeals = deals

	return message
}
