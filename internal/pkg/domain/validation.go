package domain

import (
	"unicode"
)

// ValidationResult represents the result of a validation check
type ValidationResult struct {
	IsValid bool              `json:"is_valid"`
	Errors  []ValidationError `json:"errors,omitempty"`
}

// ValidationError represents a single validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Validator defines the interface for track validation
type Validator interface {
	// Validate validates a track and returns the validation result
	Validate(track *Track) ValidationResult
}

// ValidationIssue represents a validation issue found during validation
type ValidationIssue struct {
	Field       string `json:"field"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
}

// ValidationSuggestion represents a metadata improvement suggestion
type ValidationSuggestion struct {
	Field          string `json:"field"`
	CurrentValue   string `json:"current_value"`
	SuggestedValue string `json:"suggested_value"`
	Reason         string `json:"reason"`
}

// ValidationResponse represents the complete validation response from an AI service
type ValidationResponse struct {
	ConfidenceScores struct {
		TitleArtistMatch     float64 `json:"title_artist_match"`
		GenreAccuracy        float64 `json:"genre_accuracy"`
		MusicalConsistency   float64 `json:"musical_consistency"`
		MetadataCompleteness float64 `json:"metadata_completeness"`
		Overall              float64 `json:"overall"`
	} `json:"confidence_scores"`
	Issues      []ValidationIssue      `json:"issues"`
	Suggestions []ValidationSuggestion `json:"suggestions"`
	Analysis    string                 `json:"analysis"`
}

// TrackValidator implements the Validator interface
type TrackValidator struct{}

// NewTrackValidator creates a new track validator
func NewTrackValidator() *TrackValidator {
	return &TrackValidator{}
}

// Validate validates a track and returns the validation result
func (v *TrackValidator) Validate(track *Track) ValidationResult {
	var errors []ValidationError

	// Required fields validation
	if track.Title() == "" {
		errors = append(errors, ValidationError{
			Field:   "title",
			Message: "Title is required",
		})
	}

	if track.Artist() == "" {
		errors = append(errors, ValidationError{
			Field:   "artist",
			Message: "Artist is required",
		})
	}

	if track.ISRC() != "" && !isValidISRC(track.ISRC()) {
		errors = append(errors, ValidationError{
			Field:   "isrc",
			Message: "Invalid ISRC format",
		})
	}

	return ValidationResult{
		IsValid: len(errors) == 0,
		Errors:  errors,
	}
}

// Helper validation functions

func isValidISRC(isrc string) bool {
	// ISRC format: CC-XXX-YY-NNNNN
	// CC: Country Code (2 chars)
	// XXX: Registrant Code (3 chars)
	// YY: Year (2 digits)
	// NNNNN: Designation Code (5 digits)
	if len(isrc) != 12 {
		return false
	}

	// Check country code (first two characters must be letters)
	if !isAlpha(isrc[0:2]) {
		return false
	}

	// Check registrant code (next three characters must be alphanumeric)
	if !isAlphanumeric(isrc[2:5]) {
		return false
	}

	// Check year code (next two characters must be digits)
	if !isNumeric(isrc[5:7]) {
		return false
	}

	// Check designation code (last five characters must be digits)
	if !isNumeric(isrc[7:]) {
		return false
	}

	return true
}

func isValidAudioFormat(format string) bool {
	validFormats := map[string]bool{
		string(AudioFormatMP3):  true,
		string(AudioFormatWAV):  true,
		string(AudioFormatFLAC): true,
		string(AudioFormatM4A):  true,
		string(AudioFormatAAC):  true,
		string(AudioFormatOGG):  true,
	}
	return validFormats[format]
}

func isValidBitrate(bitrate int) bool {
	return bitrate >= 32 && bitrate <= 1411
}

func isValidSampleRate(sampleRate int) bool {
	validRates := map[int]bool{
		8000:   true,
		11025:  true,
		16000:  true,
		22050:  true,
		32000:  true,
		44100:  true,
		48000:  true,
		88200:  true,
		96000:  true,
		192000: true,
	}
	return validRates[sampleRate]
}

func isValidBPM(bpm float64) bool {
	return bpm >= 20 && bpm <= 400
}

// String helper functions

func isAlpha(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

func isNumeric(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func isAlphanumeric(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
