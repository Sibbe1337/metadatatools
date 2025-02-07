package domain

import (
	"time"
	"unicode"
)

// ValidationResult represents the result of validating track metadata.
type ValidationResult struct {
	// Valid indicates if the metadata is valid
	Valid bool

	// Confidence is the AI model's confidence in the validation (0-1)
	Confidence float64

	// Issues contains any validation issues found
	Issues []ValidationIssue
}

// ValidationIssue represents a single validation issue found.
type ValidationIssue struct {
	// Field is the name of the field with the issue
	Field string

	// Message describes the validation issue
	Message string

	// Severity indicates how serious the issue is (1-5)
	Severity int
}

// Validator defines the interface for track validation
type Validator interface {
	// Validate validates a track and returns the validation result
	Validate(track *Track) ValidationResult
}

// TrackValidator implements the Validator interface for tracks
type TrackValidator struct{}

// NewTrackValidator creates a new TrackValidator instance
func NewTrackValidator() *TrackValidator {
	return &TrackValidator{}
}

// Validate implements comprehensive track validation
func (v *TrackValidator) Validate(track *Track) ValidationResult {
	var issues []ValidationIssue

	// Required fields validation
	if track.Title() == "" {
		issues = append(issues, ValidationIssue{
			Field:    "title",
			Message:  "Title is required",
			Severity: 5,
		})
	}

	if track.Artist() == "" {
		issues = append(issues, ValidationIssue{
			Field:    "artist",
			Message:  "Artist is required",
			Severity: 5,
		})
	}

	// ISRC validation
	if isrc := track.ISRC(); isrc != "" && !isValidISRC(isrc) {
		issues = append(issues, ValidationIssue{
			Field:    "isrc",
			Message:  "Invalid ISRC format",
			Severity: 4,
		})
	}

	// Duration validation
	if duration := track.Duration(); duration <= 0 {
		issues = append(issues, ValidationIssue{
			Field:    "duration",
			Message:  "Duration must be positive",
			Severity: 4,
		})
	}

	// Year validation
	if year := track.Year(); year != 0 {
		currentYear := time.Now().Year()
		if year < 1900 || year > currentYear+1 {
			issues = append(issues, ValidationIssue{
				Field:    "year",
				Message:  "Year must be between 1900 and next year",
				Severity: 3,
			})
		}
	}

	// Technical metadata validation
	if format := track.AudioFormat(); format != "" && !isValidAudioFormat(format) {
		issues = append(issues, ValidationIssue{
			Field:    "format",
			Message:  "Unsupported audio format",
			Severity: 4,
		})
	}

	if bitrate := track.Bitrate(); bitrate != 0 && !isValidBitrate(bitrate) {
		issues = append(issues, ValidationIssue{
			Field:    "bitrate",
			Message:  "Invalid bitrate",
			Severity: 3,
		})
	}

	if sampleRate := track.SampleRate(); sampleRate != 0 && !isValidSampleRate(sampleRate) {
		issues = append(issues, ValidationIssue{
			Field:    "sampleRate",
			Message:  "Invalid sample rate",
			Severity: 3,
		})
	}

	// BPM validation
	if bpm := track.BPM(); bpm != 0 && !isValidBPM(bpm) {
		issues = append(issues, ValidationIssue{
			Field:    "bpm",
			Message:  "BPM must be between 20 and 400",
			Severity: 2,
		})
	}

	return ValidationResult{
		Valid:      len(issues) == 0,
		Confidence: 1.0, // For non-AI validation
		Issues:     issues,
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
