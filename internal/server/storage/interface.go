package storage

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/zhenyanesterkova/safehub/internal/models"
)

// UserRepository определяет интерфейс для работы с пользователями
type UserRepository interface {
	// Create создает нового пользователя
	Create(ctx context.Context, user *models.User) error

	// GetByEmail получает пользователя по email
	GetByEmail(ctx context.Context, email string) (*models.User, error)

	// GetByID получает пользователя по ID
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)

	// Update обновляет данные пользователя
	Update(ctx context.Context, user *models.User) error

	// Delete удаляет пользователя
	Delete(ctx context.Context, id uuid.UUID) error

	// UpdateLastLoginAt обновляет время последнего входа
	UpdateLastLoginAt(ctx context.Context, id uuid.UUID, lastLogin time.Time) error
}

// DataRepository определяет интерфейс для работы с данными пользователей
type DataRepository interface {
	// Create создает новый элемент данных
	Create(ctx context.Context, data *models.DataItem) error

	// GetByID получает элемент данных по ID
	GetByID(ctx context.Context, id uuid.UUID) (*models.DataItem, error)

	// GetByUserID получает все данные пользователя
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.DataItem, error)

	// GetByUserIDAndType получает данные пользователя по типу
	GetByUserIDAndType(ctx context.Context, userID uuid.UUID, dataType models.DataType) ([]*models.DataItem, error)

	// Update обновляет элемент данных
	Update(ctx context.Context, data *models.DataItem) error

	// Delete удаляет элемент данных
	Delete(ctx context.Context, id uuid.UUID) error

	// GetModifiedAfter получает данные измененные после указанного времени
	GetModifiedAfter(ctx context.Context, userID uuid.UUID, after time.Time) ([]*models.DataItem, error)

	// Count возвращает количество элементов данных пользователя
	Count(ctx context.Context, userID uuid.UUID) (int64, error)
}

// SyncRepository определяет интерфейс для работы с синхронизацией
type SyncRepository interface {
	// CreateEvent создает событие синхронизации
	CreateEvent(ctx context.Context, event *models.SyncEvent) error

	// GetEventsAfter получает события после указанного ID
	GetEventsAfter(ctx context.Context, userID uuid.UUID, afterEventID uuid.UUID) ([]*models.SyncEvent, error)

	// GetLastEventID получает ID последнего события для пользователя
	GetLastEventID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)

	// DeleteOldEvents удаляет старые события (для очистки)
	DeleteOldEvents(ctx context.Context, before time.Time) error

	// GetEventsByDataID получает события по ID данных
	GetEventsByDataID(ctx context.Context, userID uuid.UUID, dataID uuid.UUID) ([]*models.SyncEvent, error)
}

// Storage объединяет все репозитории
type Storage interface {
	Users() UserRepository
	Data() DataRepository
	Sync() SyncRepository
	Close() error
	Ping(ctx context.Context) error
	BeginTx(ctx context.Context) (Transaction, error)
}

// Transaction представляет транзакцию базы данных
type Transaction interface {
	Users() UserRepository
	Data() DataRepository
	Sync() SyncRepository
	Commit() error
	Rollback() error
}

// Migrations определяет интерфейс для миграций
type Migrations interface {
	Up(ctx context.Context) error
	Down(ctx context.Context) error
	Version() int
}
