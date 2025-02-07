package base

import (
	"context"
	"database/sql"
	"fmt"
	"metadatatool/internal/pkg/domain"
	"time"
)

// TrackRepository implements domain.TrackRepository with PostgreSQL
type TrackRepository struct {
	db *sql.DB
}

// NewTrackRepository creates a new track repository
func NewTrackRepository(db *sql.DB) domain.TrackRepository {
	return &TrackRepository{
		db: db,
	}
}

// Create inserts a new track
func (r *TrackRepository) Create(ctx context.Context, track *domain.Track) error {
	query := `
		INSERT INTO tracks (
			title, artist, album, file_path, audio_format, file_size,
			isrc, iswc, genre, label, needs_review, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id`

	err := r.db.QueryRowContext(ctx, query,
		track.Title, track.Artist, track.Album, track.FilePath,
		track.AudioFormat, track.FileSize, track.ISRC, track.ISWC,
		track.Genre, track.Label, track.NeedsReview,
		time.Now(), time.Now(),
	).Scan(&track.ID)

	if err != nil {
		return fmt.Errorf("failed to create track: %w", err)
	}

	return nil
}

// GetByID retrieves a track by ID
func (r *TrackRepository) GetByID(ctx context.Context, id string) (*domain.Track, error) {
	query := `
		SELECT id, title, artist, album, file_path, audio_format,
			file_size, isrc, iswc, genre, label, needs_review,
			created_at, updated_at
		FROM tracks
		WHERE id = $1 AND deleted_at IS NULL`

	track := &domain.Track{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&track.ID, &track.Title, &track.Artist, &track.Album,
		&track.FilePath, &track.AudioFormat, &track.FileSize,
		&track.ISRC, &track.ISWC, &track.Genre, &track.Label,
		&track.NeedsReview, &track.CreatedAt, &track.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get track: %w", err)
	}

	return track, nil
}

// SearchByMetadata searches tracks by metadata fields
func (r *TrackRepository) SearchByMetadata(ctx context.Context, query map[string]interface{}) ([]*domain.Track, error) {
	// Build dynamic query based on search parameters
	baseQuery := `
		SELECT id, title, artist, album, file_path, audio_format,
			file_size, isrc, iswc, genre, label, needs_review,
			created_at, updated_at
		FROM tracks
		WHERE deleted_at IS NULL`

	var conditions []string
	var args []interface{}
	argNum := 1

	for key, value := range query {
		conditions = append(conditions, fmt.Sprintf("%s = $%d", key, argNum))
		args = append(args, value)
		argNum++
	}

	if len(conditions) > 0 {
		baseQuery += " AND " + conditions[0]
		for _, cond := range conditions[1:] {
			baseQuery += " AND " + cond
		}
	}

	rows, err := r.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search tracks: %w", err)
	}
	defer rows.Close()

	var tracks []*domain.Track
	for rows.Next() {
		track := &domain.Track{}
		err := rows.Scan(
			&track.ID, &track.Title, &track.Artist, &track.Album,
			&track.FilePath, &track.AudioFormat, &track.FileSize,
			&track.ISRC, &track.ISWC, &track.Genre, &track.Label,
			&track.NeedsReview, &track.CreatedAt, &track.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan track: %w", err)
		}
		tracks = append(tracks, track)
	}

	return tracks, nil
}

// Update modifies an existing track
func (r *TrackRepository) Update(ctx context.Context, track *domain.Track) error {
	query := `
		UPDATE tracks
		SET title = $1, artist = $2, album = $3, file_path = $4,
			audio_format = $5, file_size = $6, isrc = $7, iswc = $8,
			genre = $9, label = $10, needs_review = $11, updated_at = $12
		WHERE id = $13 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query,
		track.Title, track.Artist, track.Album, track.FilePath,
		track.AudioFormat, track.FileSize, track.ISRC, track.ISWC,
		track.Genre, track.Label, track.NeedsReview,
		time.Now(), track.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update track: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("track not found")
	}

	return nil
}

// Delete soft deletes a track
func (r *TrackRepository) Delete(ctx context.Context, id string) error {
	query := `
		UPDATE tracks
		SET deleted_at = $1
		WHERE id = $2 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to delete track: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("track not found")
	}

	return nil
}

// List retrieves a paginated list of tracks
func (r *TrackRepository) List(ctx context.Context, offset, limit int) ([]*domain.Track, error) {
	query := `
		SELECT id, title, artist, album, file_path, audio_format,
			file_size, isrc, iswc, genre, label, needs_review,
			created_at, updated_at
		FROM tracks
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list tracks: %w", err)
	}
	defer rows.Close()

	var tracks []*domain.Track
	for rows.Next() {
		track := &domain.Track{}
		err := rows.Scan(
			&track.ID, &track.Title, &track.Artist, &track.Album,
			&track.FilePath, &track.AudioFormat, &track.FileSize,
			&track.ISRC, &track.ISWC, &track.Genre, &track.Label,
			&track.NeedsReview, &track.CreatedAt, &track.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan track: %w", err)
		}
		tracks = append(tracks, track)
	}

	return tracks, nil
}
