package base

import (
	"context"
	"database/sql"
	"fmt"
	"metadatatool/internal/pkg/domain"
	"time"
)

// UserRepository implements domain.UserRepository with PostgreSQL
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sql.DB) domain.UserRepository {
	return &UserRepository{
		db: db,
	}
}

// Create inserts a new user
func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (
			email, password, name, role, company, api_key, plan,
			track_quota, tracks_used, quota_reset_date, last_login_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id`

	err := r.db.QueryRowContext(ctx, query,
		user.Email, user.Password, user.Name, user.Role, user.Company,
		user.APIKey, user.Plan, user.TrackQuota, user.TracksUsed,
		user.QuotaResetDate, user.LastLoginAt,
	).Scan(&user.ID)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `
		SELECT id, email, password, name, role, company, api_key,
			plan, track_quota, tracks_used, quota_reset_date, last_login_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL`

	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.Password, &user.Name, &user.Role,
		&user.Company, &user.APIKey, &user.Plan, &user.TrackQuota,
		&user.TracksUsed, &user.QuotaResetDate, &user.LastLoginAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, password, name, role, company, api_key,
			plan, track_quota, tracks_used, quota_reset_date, last_login_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL`

	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.Password, &user.Name, &user.Role,
		&user.Company, &user.APIKey, &user.Plan, &user.TrackQuota,
		&user.TracksUsed, &user.QuotaResetDate, &user.LastLoginAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

// GetByAPIKey retrieves a user by API key
func (r *UserRepository) GetByAPIKey(ctx context.Context, apiKey string) (*domain.User, error) {
	query := `
		SELECT id, email, password, name, role, company, api_key,
			plan, track_quota, tracks_used, quota_reset_date, last_login_at
		FROM users
		WHERE api_key = $1 AND deleted_at IS NULL`

	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, apiKey).Scan(
		&user.ID, &user.Email, &user.Password, &user.Name, &user.Role,
		&user.Company, &user.APIKey, &user.Plan, &user.TrackQuota,
		&user.TracksUsed, &user.QuotaResetDate, &user.LastLoginAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get user by API key: %w", err)
	}

	return user, nil
}

// Update modifies an existing user
func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE users
		SET email = $1, password = $2, name = $3, role = $4,
			company = $5, api_key = $6, plan = $7, track_quota = $8,
			tracks_used = $9, quota_reset_date = $10, last_login_at = $11
		WHERE id = $12 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query,
		user.Email, user.Password, user.Name, user.Role,
		user.Company, user.APIKey, user.Plan, user.TrackQuota,
		user.TracksUsed, user.QuotaResetDate, user.LastLoginAt,
		user.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// Delete soft deletes a user
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	query := `
		UPDATE users
		SET deleted_at = $1
		WHERE id = $2 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// List retrieves a paginated list of users
func (r *UserRepository) List(ctx context.Context, offset, limit int) ([]*domain.User, error) {
	query := `
		SELECT id, email, password, name, role, company, api_key,
			plan, track_quota, tracks_used, quota_reset_date, last_login_at
		FROM users
		WHERE deleted_at IS NULL
		ORDER BY last_login_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		user := &domain.User{}
		err := rows.Scan(
			&user.ID, &user.Email, &user.Password, &user.Name, &user.Role,
			&user.Company, &user.APIKey, &user.Plan, &user.TrackQuota,
			&user.TracksUsed, &user.QuotaResetDate, &user.LastLoginAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}
