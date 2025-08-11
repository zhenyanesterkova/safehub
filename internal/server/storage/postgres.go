package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zhenyanesterkova/safehub/internal/models"
)

// PostgresStorage представляет PostgreSQL реализацию хранилища
type PostgresStorage struct {
	pool *pgxpool.Pool
}

// NewPostgresStorage создает новое PostgreSQL хранилище
func NewPostgresStorage(databaseURL string) (*PostgresStorage, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Настройка пула соединений
	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = 5 * time.Minute
	config.MaxConnIdleTime = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	return &PostgresStorage{pool: pool}, nil
}

// Users возвращает репозиторий пользователей
func (s *PostgresStorage) Users() UserRepository {
	return &userRepository{db: s.pool}
}

// Data возвращает репозиторий данных
func (s *PostgresStorage) Data() DataRepository {
	return &dataRepository{db: s.pool}
}

// Sync возвращает репозиторий синхронизации
func (s *PostgresStorage) Sync() SyncRepository {
	return &syncRepository{db: s.pool}
}

// Close закрывает пул соединений
func (s *PostgresStorage) Close() error {
	s.pool.Close()
	return nil
}

// Ping проверяет соединение с базой данных
func (s *PostgresStorage) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// BeginTx начинает новую транзакцию
func (s *PostgresStorage) BeginTx(ctx context.Context) (Transaction, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return &postgresTransaction{tx: tx}, nil
}

// postgresTransaction представляет транзакцию PostgreSQL
type postgresTransaction struct {
	tx pgx.Tx
}

// Users возвращает репозиторий пользователей в рамках транзакции
func (t *postgresTransaction) Users() UserRepository {
	return &userRepository{db: t.tx}
}

// Data возвращает репозиторий данных в рамках транзакции
func (t *postgresTransaction) Data() DataRepository {
	return &dataRepository{db: t.tx}
}

// Sync возвращает репозиторий синхронизации в рамках транзакции
func (t *postgresTransaction) Sync() SyncRepository {
	return &syncRepository{db: t.tx}
}

// Commit фиксирует транзакцию
func (t *postgresTransaction) Commit() error {
	return t.tx.Commit(context.Background())
}

// Rollback откатывает транзакцию
func (t *postgresTransaction) Rollback() error {
	return t.tx.Rollback(context.Background())
}

// Интерфейс для работы с базой данных (поддерживает как pgxpool.Pool, так и pgx.Tx)
type dbInterface interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// userRepository реализует UserRepository для PostgreSQL
type userRepository struct {
	db dbInterface
}

func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (email, password_hash, salt, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`

	err := r.db.QueryRow(ctx, query,
		user.Email,
		user.PasswordHash,
		user.Salt,
		user.CreatedAt,
		user.UpdatedAt,
	).Scan(&user.ID)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, email, password_hash, salt, created_at, updated_at, last_login_at
		FROM users 
		WHERE email = $1`

	var lastLoginAt pgtype.Timestamptz
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Salt,
		&user.CreatedAt,
		&user.UpdatedAt,
		&lastLoginAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	if lastLoginAt.Valid {
		user.LastLoginAt = lastLoginAt.Time
	}

	return user, nil
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, email, password_hash, salt, created_at, updated_at, last_login_at
		FROM users 
		WHERE id = $1`

	var lastLoginAt pgtype.Timestamptz
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Salt,
		&user.CreatedAt,
		&user.UpdatedAt,
		&lastLoginAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, models.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	if lastLoginAt.Valid {
		user.LastLoginAt = lastLoginAt.Time
	}

	return user, nil
}

func (r *userRepository) Update(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users 
		SET email = $2, password_hash = $3, salt = $4, updated_at = $5
		WHERE id = $1`

	cmdTag, err := r.db.Exec(ctx, query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.Salt,
		user.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return models.ErrUserNotFound
	}

	return nil
}

func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`

	cmdTag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return models.ErrUserNotFound
	}

	return nil
}

func (r *userRepository) UpdateLastLoginAt(ctx context.Context, id uuid.UUID, lastLogin time.Time) error {
	query := `UPDATE users SET last_login_at = $2 WHERE id = $1`

	cmdTag, err := r.db.Exec(ctx, query, id, lastLogin)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return models.ErrUserNotFound
	}

	return nil
}

// dataRepository реализует DataRepository для PostgreSQL
type dataRepository struct {
	db dbInterface
}

func (r *dataRepository) Create(ctx context.Context, data *models.DataItem) error {
	query := `
		INSERT INTO data_items (user_id, name, type, metadata, created_at, updated_at, version)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`

	err := r.db.QueryRow(ctx, query,
		data.UserID,
		data.Name,
		data.Type,
		data.Data,
		data.Metadata,
		data.CreatedAt,
		data.UpdatedAt,
		data.Version,
	).Scan(&data.ID)

	if err != nil {
		return fmt.Errorf("failed to create data item: %w", err)
	}

	return nil
}

func (r *dataRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.DataItem, error) {
	data := &models.DataItem{}
	query := `
		SELECT id, user_id, name, type, encrypted_data, metadata, created_at, updated_at, version
		FROM data_items 
		WHERE id = $1 AND deleted_at IS NULL`

	err := r.db.QueryRow(ctx, query, id).Scan(
		&data.ID,
		&data.UserID,
		&data.Name,
		&data.Type,
		&data.Data,
		&data.Metadata,
		&data.CreatedAt,
		&data.UpdatedAt,
		&data.Version,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, models.ErrDataNotFound
		}
		return nil, fmt.Errorf("failed to get data item by ID: %w", err)
	}

	return data, nil
}

func (r *dataRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.DataItem, error) {
	query := `
		SELECT id, user_id, name, type, encrypted_data, metadata, created_at, updated_at, version
		FROM data_items 
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get data items by user ID: %w", err)
	}
	defer rows.Close()

	var items []*models.DataItem
	for rows.Next() {
		data := &models.DataItem{}
		err := rows.Scan(
			&data.ID,
			&data.UserID,
			&data.Name,
			&data.Type,
			&data.Data,
			&data.Metadata,
			&data.CreatedAt,
			&data.UpdatedAt,
			&data.Version,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan data item: %w", err)
		}
		items = append(items, data)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over data items: %w", err)
	}

	return items, nil
}

func (r *dataRepository) GetByUserIDAndType(ctx context.Context, userID uuid.UUID, dataType models.DataType) ([]*models.DataItem, error) {
	query := `
		SELECT id, user_id, name, type, encrypted_data, metadata, created_at, updated_at, version
		FROM data_items 
		WHERE user_id = $1 AND type = $2 AND deleted_at IS NULL
		ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, userID, dataType)
	if err != nil {
		return nil, fmt.Errorf("failed to get data items by user ID and type: %w", err)
	}
	defer rows.Close()

	var items []*models.DataItem
	for rows.Next() {
		data := &models.DataItem{}
		err := rows.Scan(
			&data.ID,
			&data.UserID,
			&data.Name,
			&data.Type,
			&data.Data,
			&data.Metadata,
			&data.CreatedAt,
			&data.UpdatedAt,
			&data.Version,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan data item: %w", err)
		}
		items = append(items, data)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over data items: %w", err)
	}

	return items, nil
}

func (r *dataRepository) Update(ctx context.Context, data *models.DataItem) error {
	query := `
		UPDATE data_items 
		SET name = $2, encrypted_data = $3, metadata = $4, updated_at = $5, version = $6
		WHERE id = $1 AND deleted_at IS NULL`

	cmdTag, err := r.db.Exec(ctx, query,
		data.ID,
		data.Name,
		data.Data,
		data.Metadata,
		data.UpdatedAt,
		data.Version,
	)

	if err != nil {
		return fmt.Errorf("failed to update data item: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return models.ErrDataNotFound
	}

	return nil
}

func (r *dataRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE data_items SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`

	cmdTag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete data item: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return models.ErrDataNotFound
	}

	return nil
}

func (r *dataRepository) GetModifiedAfter(ctx context.Context, userID uuid.UUID, after time.Time) ([]*models.DataItem, error) {
	query := `
		SELECT id, user_id, name, type, encrypted_data, metadata, created_at, updated_at, version
		FROM data_items 
		WHERE user_id = $1 AND updated_at > $2 AND deleted_at IS NULL
		ORDER BY updated_at ASC`

	rows, err := r.db.Query(ctx, query, userID, after)
	if err != nil {
		return nil, fmt.Errorf("failed to get modified data items: %w", err)
	}
	defer rows.Close()

	var items []*models.DataItem
	for rows.Next() {
		data := &models.DataItem{}
		err := rows.Scan(
			&data.ID,
			&data.UserID,
			&data.Name,
			&data.Type,
			&data.Data,
			&data.Metadata,
			&data.CreatedAt,
			&data.UpdatedAt,
			&data.Version,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan data item: %w", err)
		}
		items = append(items, data)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over data items: %w", err)
	}

	return items, nil
}

func (r *dataRepository) Count(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM data_items WHERE user_id = $1 AND deleted_at IS NULL`

	err := r.db.QueryRow(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count data items: %w", err)
	}

	return count, nil
}

// syncRepository реализует SyncRepository для PostgreSQL
type syncRepository struct {
	db dbInterface
}

func (r *syncRepository) CreateEvent(ctx context.Context, event *models.SyncEvent) error {
	query := `
		INSERT INTO sync_events (user_id, data_id, event_type, timestamp, data_version)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`

	err := r.db.QueryRow(ctx, query,
		event.UserID,
		event.DataID,
		event.Action,
		event.Created,
		event.Version,
	).Scan(&event.ID)

	if err != nil {
		return fmt.Errorf("failed to create sync event: %w", err)
	}

	return nil
}

func (r *syncRepository) GetEventsAfter(ctx context.Context, userID uuid.UUID, afterEventID uuid.UUID) ([]*models.SyncEvent, error) {
	query := `
		SELECT id, user_id, data_id, event_type, timestamp, data_version
		FROM sync_events 
		WHERE user_id = $1 AND id > $2
		ORDER BY timestamp ASC`

	rows, err := r.db.Query(ctx, query, userID, afterEventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sync events: %w", err)
	}
	defer rows.Close()

	var events []*models.SyncEvent
	for rows.Next() {
		event := &models.SyncEvent{}

		err := rows.Scan(
			&event.ID,
			&event.UserID,
			&event.DataID,
			&event.Action,
			&event.Created,
			&event.Version,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan sync event: %w", err)
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over sync events: %w", err)
	}

	return events, nil
}

func (r *syncRepository) GetLastEventID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var lastID uuid.UUID
	query := `SELECT id FROM sync_events WHERE user_id = $1 ORDER BY timestamp DESC LIMIT 1`

	err := r.db.QueryRow(ctx, query, userID).Scan(&lastID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get last event ID: %w", err)
	}

	return lastID, nil
}

func (r *syncRepository) DeleteOldEvents(ctx context.Context, before time.Time) error {
	query := `DELETE FROM sync_events WHERE timestamp < $1`

	_, err := r.db.Exec(ctx, query, before)
	if err != nil {
		return fmt.Errorf("failed to delete old sync events: %w", err)
	}

	return nil
}

func (r *syncRepository) GetEventsByDataID(ctx context.Context, userID uuid.UUID, dataID uuid.UUID) ([]*models.SyncEvent, error) {
	query := `
		SELECT id, user_id, data_id, event_type, timestamp, data_version
		FROM sync_events 
		WHERE user_id = $1 AND data_id = $2
		ORDER BY timestamp ASC`

	rows, err := r.db.Query(ctx, query, userID, dataID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sync events by data ID: %w", err)
	}
	defer rows.Close()

	var events []*models.SyncEvent
	for rows.Next() {
		event := &models.SyncEvent{}

		err := rows.Scan(
			&event.ID,
			&event.UserID,
			&event.DataID,
			&event.Action,
			&event.Created,
			&event.Version,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan sync event: %w", err)
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over sync events: %w", err)
	}

	return events, nil
}
