package utils

import "path/filepath"

// IsValidAudioFormat checks if the file extension is a supported audio format
func IsValidAudioFormat(filename string) bool {
	ext := filepath.Ext(filename)
	switch ext {
	case ".mp3", ".wav", ".flac", ".m4a", ".aac":
		return true
	default:
		return false
	}
}

// GetAudioFormat returns the audio format from the file extension
func GetAudioFormat(filename string) string {
	return filepath.Ext(filename)[1:] // Remove the dot
}
