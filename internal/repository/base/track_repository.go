package base

import (
	"context"
	"database/sql"
	"encoding/json"
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
			id, storage_path, file_size, metadata, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`

	metadata, err := json.Marshal(track.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	err = r.db.QueryRowContext(ctx, query,
		track.ID, track.StoragePath, track.FileSize,
		metadata, track.CreatedAt, track.UpdatedAt,
	).Scan(&track.ID)

	if err != nil {
		return fmt.Errorf("failed to create track: %w", err)
	}

	return nil
}

// GetByID retrieves a track by ID
func (r *TrackRepository) GetByID(ctx context.Context, id string) (*domain.Track, error) {
	query := `
		SELECT id, storage_path, file_size, metadata, created_at, updated_at, deleted_at
		FROM tracks
		WHERE id = $1 AND deleted_at IS NULL`

	track := &domain.Track{}
	var metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&track.ID, &track.StoragePath, &track.FileSize,
		&metadataJSON, &track.CreatedAt, &track.UpdatedAt, &track.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get track: %w", err)
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &track.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return track, nil
}

// List retrieves tracks based on filters with pagination
func (r *TrackRepository) List(ctx context.Context, filters map[string]interface{}, offset, limit int) ([]*domain.Track, error) {
	query := `
		SELECT id, storage_path, file_size, metadata, created_at, updated_at, deleted_at
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
		var metadataJSON []byte

		err := rows.Scan(
			&track.ID, &track.StoragePath, &track.FileSize,
			&metadataJSON, &track.CreatedAt, &track.UpdatedAt, &track.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan track: %w", err)
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &track.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		tracks = append(tracks, track)
	}

	return tracks, nil
}

// SearchByMetadata searches tracks by metadata fields
func (r *TrackRepository) SearchByMetadata(ctx context.Context, query map[string]interface{}) ([]*domain.Track, error) {
	// Build query dynamically based on search criteria
	sqlQuery := `
		SELECT id, storage_path, file_size, metadata, created_at, updated_at, deleted_at
		FROM tracks
		WHERE deleted_at IS NULL`

	var params []interface{}
	var conditions []string

	// Add search conditions
	for key, value := range query {
		params = append(params, value)
		conditions = append(conditions, fmt.Sprintf("metadata->>'%s' = $%d", key, len(params)))
	}

	if len(conditions) > 0 {
		sqlQuery += " AND " + conditions[0]
		for i := 1; i < len(conditions); i++ {
			sqlQuery += " AND " + conditions[i]
		}
	}

	sqlQuery += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, sqlQuery, params...)
	if err != nil {
		return nil, fmt.Errorf("failed to search tracks: %w", err)
	}
	defer rows.Close()

	var tracks []*domain.Track
	for rows.Next() {
		track := &domain.Track{}
		var metadataJSON []byte

		err := rows.Scan(
			&track.ID, &track.StoragePath, &track.FileSize,
			&metadataJSON, &track.CreatedAt, &track.UpdatedAt, &track.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan track: %w", err)
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &track.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		tracks = append(tracks, track)
	}

	return tracks, nil
}

// Update updates an existing track
func (r *TrackRepository) Update(ctx context.Context, track *domain.Track) error {
	query := `
		UPDATE tracks
		SET storage_path = $1,
			file_size = $2,
			metadata = $3,
			updated_at = $4
		WHERE id = $5 AND deleted_at IS NULL`

	metadata, err := json.Marshal(track.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	result, err := r.db.ExecContext(ctx, query,
		track.StoragePath, track.FileSize,
		metadata, time.Now(), track.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update track: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("track not found")
	}

	return nil
}

// Delete soft-deletes a track
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
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("track not found")
	}

	return nil
}

// GetByISRC retrieves a track by ISRC
func (r *TrackRepository) GetByISRC(ctx context.Context, isrc string) (*domain.Track, error) {
	query := `
		SELECT id, storage_path, file_size, metadata, created_at, updated_at, deleted_at
		FROM tracks
		WHERE metadata->>'isrc' = $1 AND deleted_at IS NULL`

	track := &domain.Track{}
	var metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, isrc).Scan(
		&track.ID, &track.StoragePath, &track.FileSize,
		&metadataJSON, &track.CreatedAt, &track.UpdatedAt, &track.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get track by ISRC: %w", err)
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &track.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return track, nil
}

// BatchUpdate updates multiple tracks in a single transaction
func (r *TrackRepository) BatchUpdate(ctx context.Context, tracks []*domain.Track) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		UPDATE tracks
		SET storage_path = $1,
			file_size = $2,
			metadata = $3,
			updated_at = $4
		WHERE id = $5 AND deleted_at IS NULL`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, track := range tracks {
		metadata, err := json.Marshal(track.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}

		result, err := stmt.ExecContext(ctx,
			track.StoragePath, track.FileSize,
			metadata, time.Now(), track.ID,
		)
		if err != nil {
			return fmt.Errorf("failed to update track %s: %w", track.ID, err)
		}

		rows, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get affected rows: %w", err)
		}

		if rows == 0 {
			return fmt.Errorf("track %s not found", track.ID)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
