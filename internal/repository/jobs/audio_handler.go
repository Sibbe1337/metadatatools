package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"metadatatool/internal/pkg/domain"
	"metadatatool/internal/pkg/metrics"
)

// AudioProcessPayload represents the payload for audio processing jobs
type AudioProcessPayload struct {
	TrackID     string `json:"track_id"`
	StoragePath string `json:"storage_path"`
	Format      string `json:"format,omitempty"`
}

// AudioProcessHandler handles audio processing jobs
type AudioProcessHandler struct {
	audioProcessor domain.AudioProcessor
	trackRepo      domain.TrackRepository
	storageClient  domain.StorageClient
}

// NewAudioProcessHandler creates a new audio process handler
func NewAudioProcessHandler(
	audioProcessor domain.AudioProcessor,
	trackRepo domain.TrackRepository,
	storageClient domain.StorageClient,
) *AudioProcessHandler {
	return &AudioProcessHandler{
		audioProcessor: audioProcessor,
		trackRepo:      trackRepo,
		storageClient:  storageClient,
	}
}

// JobType returns the type of job this handler processes
func (h *AudioProcessHandler) JobType() domain.JobType {
	return domain.JobTypeAudioProcess
}

// HandleJob processes an audio processing job
func (h *AudioProcessHandler) HandleJob(ctx context.Context, job *domain.Job) error {
	start := time.Now()
	defer func() {
		metrics.JobProcessingDuration.WithLabelValues(string(job.Type)).Observe(time.Since(start).Seconds())
	}()

	// Parse payload
	var payload AudioProcessPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Validate payload
	if payload.TrackID == "" || payload.StoragePath == "" {
		return fmt.Errorf("invalid payload: missing required fields")
	}

	// Get track from repository
	track, err := h.trackRepo.GetByID(ctx, payload.TrackID)
	if err != nil {
		return fmt.Errorf("failed to get track: %w", err)
	}

	// Download audio file
	audioData, err := h.storageClient.Download(ctx, payload.StoragePath)
	if err != nil {
		return fmt.Errorf("failed to download audio: %w", err)
	}

	// Convert StorageFile to ProcessingAudioFile
	processingFile := &domain.ProcessingAudioFile{
		Name:    audioData.Name,
		Path:    audioData.Key,
		Size:    audioData.Size,
		Format:  domain.AudioFormat(payload.Format),
		Content: audioData.Content,
	}

	// Process audio file
	result, err := h.audioProcessor.Process(ctx, processingFile, &domain.AudioProcessOptions{
		FilePath:        processingFile.Path,
		Format:          processingFile.Format,
		AnalyzeAudio:    true,
		ExtractMetadata: true,
	})
	if err != nil {
		return fmt.Errorf("failed to process audio: %w", err)
	}

	// Update track with extracted metadata
	track.SetTitle(result.Metadata.Title)
	track.SetArtist(result.Metadata.Artist)
	track.SetAlbum(result.Metadata.Album)
	track.SetYear(result.Metadata.Year)
	track.SetDuration(result.Metadata.Duration)
	track.SetBPM(result.Metadata.Musical.BPM)
	track.SetKey(result.Metadata.Musical.Key)
	track.SetISRC(result.Metadata.ISRC)
	track.SetAudioFormat(string(result.Metadata.Technical.Format))
	track.SetSampleRate(result.Metadata.Technical.SampleRate)
	track.SetBitrate(result.Metadata.Technical.Bitrate)
	track.SetChannels(result.Metadata.Technical.Channels)

	// Save updated track
	if err := h.trackRepo.Update(ctx, track); err != nil {
		return fmt.Errorf("failed to update track: %w", err)
	}

	return nil
}
