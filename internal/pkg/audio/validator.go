package audio

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"

	"github.com/dhowden/tag"
)

// SupportedFormats defines the supported audio file formats and their MIME types
var SupportedFormats = map[string][]string{
	".mp3":  {"audio/mpeg", "audio/mp3"},
	".wav":  {"audio/wav", "audio/x-wav", "audio/wave"},
	".flac": {"audio/flac", "audio/x-flac"},
	".m4a":  {"audio/mp4", "audio/x-m4a"},
	".ogg":  {"audio/ogg", "application/ogg"},
	".aac":  {"audio/aac", "audio/aacp"},
}

// MaxFileSize defines the maximum allowed file size (100MB)
const MaxFileSize = 100 * 1024 * 1024

// ValidationError represents an audio file validation error
type ValidationError struct {
	Code    string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Common validation error codes
const (
	ErrUnsupportedFormat = "UNSUPPORTED_FORMAT"
	ErrFileTooLarge      = "FILE_TOO_LARGE"
	ErrInvalidContent    = "INVALID_CONTENT"
	ErrCorruptMetadata   = "CORRUPT_METADATA"
)

// ValidateFormat checks if the file extension and MIME type are supported
func ValidateFormat(filename, mimeType string) error {
	ext := filepath.Ext(filename)
	if ext == "" {
		return &ValidationError{
			Code:    ErrUnsupportedFormat,
			Message: "File has no extension",
		}
	}

	validMimeTypes, supported := SupportedFormats[ext]
	if !supported {
		return &ValidationError{
			Code:    ErrUnsupportedFormat,
			Message: fmt.Sprintf("Unsupported file format: %s", ext),
		}
	}

	// Check MIME type if provided
	if mimeType != "" {
		mimeValid := false
		for _, validType := range validMimeTypes {
			if mimeType == validType {
				mimeValid = true
				break
			}
		}
		if !mimeValid {
			return &ValidationError{
				Code:    ErrUnsupportedFormat,
				Message: fmt.Sprintf("Invalid MIME type for %s: %s", ext, mimeType),
			}
		}
	}

	return nil
}

// ValidateContent performs content-based validation of the audio file
func ValidateContent(content io.Reader, size int64) error {
	if size > MaxFileSize {
		return &ValidationError{
			Code:    ErrFileTooLarge,
			Message: fmt.Sprintf("File size exceeds maximum allowed size of %d bytes", MaxFileSize),
		}
	}

	// Read a sample of the file to validate format
	sample := make([]byte, 512)
	n, err := content.Read(sample)
	if err != nil && err != io.EOF {
		return &ValidationError{
			Code:    ErrInvalidContent,
			Message: "Failed to read file content",
		}
	}

	// Try to parse metadata to validate file integrity
	if _, err := tag.ReadFrom(bytes.NewReader(sample[:n])); err != nil {
		return &ValidationError{
			Code:    ErrCorruptMetadata,
			Message: "Failed to read audio metadata",
		}
	}

	return nil
}

// ExtractMetadata extracts metadata from the audio file
func ExtractMetadata(content io.Reader) (tag.Metadata, error) {
	// Read the entire content into a buffer to make it seekable
	data, err := io.ReadAll(content)
	if err != nil {
		return nil, &ValidationError{
			Code:    ErrInvalidContent,
			Message: fmt.Sprintf("Failed to read content: %v", err),
		}
	}

	// Create a seekable reader from the buffer
	reader := bytes.NewReader(data)
	metadata, err := tag.ReadFrom(reader)
	if err != nil {
		return nil, &ValidationError{
			Code:    ErrCorruptMetadata,
			Message: fmt.Sprintf("Failed to extract metadata: %v", err),
		}
	}
	return metadata, nil
}
