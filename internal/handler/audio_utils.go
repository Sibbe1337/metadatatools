package handler

import "path/filepath"

// isValidAudioFormat checks if the file extension is a supported audio format
func isValidAudioFormat(filename string) bool {
	ext := filepath.Ext(filename)
	switch ext {
	case ".mp3", ".wav", ".flac", ".m4a", ".aac":
		return true
	default:
		return false
	}
}

// getAudioFormat returns the audio format from the file extension
func getAudioFormat(filename string) string {
	return filepath.Ext(filename)[1:] // Remove the dot
}
