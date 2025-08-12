package storage

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zhenyanesterkova/safehub/internal/models"
)

func TestNewPostgresStorage(t *testing.T) {
	// Test with invalid URL
	storage, err := NewPostgresStorage("invalid-url")
	assert.Error(t, err)
	assert.Nil(t, storage)
}

func TestUserRepository_Create(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := &userRepository{db: mock}
	user := &models.User{
		Username:     "username",
		Email:        "test@example.com",
		PasswordHash: "hash",
		Salt:         "salt",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	expectedID := uuid.New()
	rows := pgxmock.NewRows([]string{"id"}).AddRow(expectedID)

	mock.ExpectQuery("INSERT INTO users").
		WithArgs(user.Username, user.Email, user.PasswordHash, user.Salt, user.CreatedAt, user.UpdatedAt).
		WillReturnRows(rows)

	err = repo.Create(context.Background(), user)
	assert.NoError(t, err)
	assert.Equal(t, expectedID, user.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetByID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := &userRepository{db: mock}

	t.Run("successful get", func(t *testing.T) {
		expectedUser := &models.User{
			ID:           uuid.New(),
			Username:     "username",
			Email:        "test@example.com",
			PasswordHash: "hash",
			Salt:         "salt",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			LastLoginAt:  time.Now(),
		}

		lastLoginTime := pgtype.Timestamptz{
			Time:  expectedUser.LastLoginAt,
			Valid: true,
		}

		rows := pgxmock.NewRows([]string{
			"id", "username", "email", "password_hash", "salt", "created_at", "updated_at", "last_login_at",
		}).AddRow(
			expectedUser.ID,
			expectedUser.Username,
			expectedUser.Email,
			expectedUser.PasswordHash,
			expectedUser.Salt,
			expectedUser.CreatedAt,
			expectedUser.UpdatedAt,
			lastLoginTime,
		)

		mock.ExpectQuery("SELECT (.+) FROM users").
			WithArgs(expectedUser.ID).
			WillReturnRows(rows)

		user, err := repo.GetByID(context.Background(), expectedUser.ID)
		assert.NoError(t, err)
		assert.Equal(t, expectedUser, user)
	})

	t.Run("user not found", func(t *testing.T) {
		userID := uuid.New()

		mock.ExpectQuery("SELECT (.+) FROM users").
			WithArgs(userID).
			WillReturnError(pgx.ErrNoRows)

		user, err := repo.GetByID(context.Background(), userID)
		assert.Error(t, err)
		assert.Equal(t, models.ErrUserNotFound, err)
		assert.Nil(t, user)
	})

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Update(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := &userRepository{db: mock}

	t.Run("successful update", func(t *testing.T) {
		user := &models.User{
			ID:           uuid.New(),
			Username:     "username",
			Email:        "updated@example.com",
			PasswordHash: "newhash",
			Salt:         "newsalt",
			UpdatedAt:    time.Now(),
		}

		mock.ExpectExec("UPDATE users").
			WithArgs(
				user.ID,
				user.Username,
				user.Email,
				user.PasswordHash,
				user.Salt,
				user.UpdatedAt,
			).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := repo.Update(context.Background(), user)
		assert.NoError(t, err)
	})

	t.Run("user not found", func(t *testing.T) {
		user := &models.User{
			ID:           uuid.New(),
			Username:     "username",
			Email:        "notfound@example.com",
			PasswordHash: "hash",
			Salt:         "salt",
			UpdatedAt:    time.Now(),
		}

		mock.ExpectExec("UPDATE users").
			WithArgs(
				user.ID,
				user.Username,
				user.Email,
				user.PasswordHash,
				user.Salt,
				user.UpdatedAt,
			).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		err := repo.Update(context.Background(), user)
		assert.Equal(t, models.ErrUserNotFound, err)
	})

	t.Run("database error", func(t *testing.T) {
		user := &models.User{
			ID:           uuid.New(),
			Username:     "username",
			Email:        "error@example.com",
			PasswordHash: "hash",
			Salt:         "salt",
			UpdatedAt:    time.Now(),
		}

		mock.ExpectExec("UPDATE users").
			WithArgs(
				user.ID,
				user.Username,
				user.Email,
				user.PasswordHash,
				user.Salt,
				user.UpdatedAt,
			).
			WillReturnError(fmt.Errorf("database error"))

		err := repo.Update(context.Background(), user)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update user")
	})

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetByEmail(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := &userRepository{db: mock}
	expectedUser := &models.User{
		ID:           uuid.New(),
		Username:     "username",
		Email:        "test@example.com",
		PasswordHash: "hash",
		Salt:         "salt",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		LastLoginAt:  time.Now(),
	}

	lastLoginTime := pgtype.Timestamptz{
		Time:  expectedUser.LastLoginAt,
		Valid: true,
	}

	rows := pgxmock.NewRows([]string{
		"id", "username", "email", "password_hash", "salt", "created_at", "updated_at", "last_login_at",
	}).AddRow(
		expectedUser.ID,
		expectedUser.Username,
		expectedUser.Email,
		expectedUser.PasswordHash,
		expectedUser.Salt,
		expectedUser.CreatedAt,
		expectedUser.UpdatedAt,
		lastLoginTime,
	)

	mock.ExpectQuery("SELECT (.+) FROM users").
		WithArgs(expectedUser.Email).
		WillReturnRows(rows)

	user, err := repo.GetByEmail(context.Background(), expectedUser.Email)
	assert.NoError(t, err)
	assert.Equal(t, expectedUser, user)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetByUsername(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := &userRepository{db: mock}
	expectedUser := &models.User{
		ID:           uuid.New(),
		Username:     "username",
		Email:        "test@example.com",
		PasswordHash: "hash",
		Salt:         "salt",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		LastLoginAt:  time.Now(),
	}

	lastLoginTime := pgtype.Timestamptz{
		Time:  expectedUser.LastLoginAt,
		Valid: true,
	}

	rows := pgxmock.NewRows([]string{
		"id", "username", "email", "password_hash", "salt", "created_at", "updated_at", "last_login_at",
	}).AddRow(
		expectedUser.ID,
		expectedUser.Username,
		expectedUser.Email,
		expectedUser.PasswordHash,
		expectedUser.Salt,
		expectedUser.CreatedAt,
		expectedUser.UpdatedAt,
		lastLoginTime,
	)

	mock.ExpectQuery("SELECT (.+) FROM users").
		WithArgs(expectedUser.Username).
		WillReturnRows(rows)

	user, err := repo.GetByUsername(context.Background(), expectedUser.Username)
	assert.NoError(t, err)
	assert.Equal(t, expectedUser, user)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_UpdateLastLoginAt(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := &userRepository{db: mock}

	t.Run("successful update", func(t *testing.T) {
		userID := uuid.New()
		lastLogin := time.Now()

		mock.ExpectExec("UPDATE users SET last_login_at = \\$2 WHERE id = \\$1").
			WithArgs(userID, lastLogin).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := repo.UpdateLastLoginAt(context.Background(), userID, lastLogin)
		assert.NoError(t, err)
	})

	t.Run("user not found", func(t *testing.T) {
		userID := uuid.New()
		lastLogin := time.Now()

		mock.ExpectExec("UPDATE users SET last_login_at = \\$2 WHERE id = \\$1").
			WithArgs(userID, lastLogin).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		err := repo.UpdateLastLoginAt(context.Background(), userID, lastLogin)
		assert.Equal(t, models.ErrUserNotFound, err)
	})

	t.Run("database error", func(t *testing.T) {
		userID := uuid.New()
		lastLogin := time.Now()

		mock.ExpectExec("UPDATE users SET last_login_at = \\$2 WHERE id = \\$1").
			WithArgs(userID, lastLogin).
			WillReturnError(fmt.Errorf("database error"))

		err := repo.UpdateLastLoginAt(context.Background(), userID, lastLogin)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update last login")
	})

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Delete(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := &userRepository{db: mock}
	userID := uuid.New()

	// Test successful deletion
	mock.ExpectExec("DELETE FROM users").
		WithArgs(userID).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err = repo.Delete(context.Background(), userID)
	assert.NoError(t, err)

	// Test user not found
	mock.ExpectExec("DELETE FROM users").
		WithArgs(userID).
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	err = repo.Delete(context.Background(), userID)
	assert.Equal(t, models.ErrUserNotFound, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDataRepository_Create(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := &dataRepository{db: mock}
	data := &models.DataItem{
		UserID:    uuid.New(),
		Name:      "test",
		Type:      models.DataTypeText,
		Data:      []byte("data"),
		Metadata:  []byte("metadata"),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Version:   1,
	}

	expectedID := uuid.New()
	rows := pgxmock.NewRows([]string{"id"}).AddRow(expectedID)

	mock.ExpectQuery("INSERT INTO data_items").
		WithArgs(
			data.UserID,
			data.Name,
			data.Type,
			data.Data,
			data.Metadata,
			data.CreatedAt,
			data.UpdatedAt,
			data.Version,
		).
		WillReturnRows(rows)

	err = repo.Create(context.Background(), data)
	assert.NoError(t, err)
	assert.Equal(t, expectedID, data.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDataRepository_GetByID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := &dataRepository{db: mock}

	t.Run("successful get", func(t *testing.T) {
		expectedData := &models.DataItem{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			Name:      "test-data",
			Type:      models.DataTypeText,
			Data:      []byte("encrypted data"),
			Metadata:  []byte("metadata"),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Version:   1,
		}

		rows := pgxmock.NewRows([]string{
			"id", "user_id", "name", "type", "encrypted_data",
			"metadata", "created_at", "updated_at", "version",
		}).AddRow(
			expectedData.ID,
			expectedData.UserID,
			expectedData.Name,
			expectedData.Type,
			expectedData.Data,
			expectedData.Metadata,
			expectedData.CreatedAt,
			expectedData.UpdatedAt,
			expectedData.Version,
		)

		mock.ExpectQuery("SELECT (.+) FROM data_items").
			WithArgs(expectedData.ID).
			WillReturnRows(rows)

		data, err := repo.GetByID(context.Background(), expectedData.ID)
		assert.NoError(t, err)
		assert.Equal(t, expectedData, data)
	})

	t.Run("data not found", func(t *testing.T) {
		id := uuid.New()

		mock.ExpectQuery("SELECT (.+) FROM data_items").
			WithArgs(id).
			WillReturnError(pgx.ErrNoRows)

		data, err := repo.GetByID(context.Background(), id)
		assert.Error(t, err)
		assert.Equal(t, models.ErrDataNotFound, err)
		assert.Nil(t, data)
	})

	t.Run("database error", func(t *testing.T) {
		id := uuid.New()

		mock.ExpectQuery("SELECT (.+) FROM data_items").
			WithArgs(id).
			WillReturnError(fmt.Errorf("database error"))

		data, err := repo.GetByID(context.Background(), id)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get data item by ID")
		assert.Nil(t, data)
	})

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDataRepository_GetByUserID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := &dataRepository{db: mock}

	t.Run("successful get", func(t *testing.T) {
		userID := uuid.New()
		expectedItems := []*models.DataItem{
			{
				ID:        uuid.New(),
				UserID:    userID,
				Name:      "test-data-1",
				Type:      models.DataTypeText,
				Data:      []byte("encrypted data 1"),
				Metadata:  []byte("metadata 1"),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Version:   1,
			},
			{
				ID:        uuid.New(),
				UserID:    userID,
				Name:      "test-data-2",
				Type:      models.DataTypeText,
				Data:      []byte("encrypted data 2"),
				Metadata:  []byte("metadata 2"),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Version:   1,
			},
		}

		rows := pgxmock.NewRows([]string{
			"id", "user_id", "name", "type", "encrypted_data",
			"metadata", "created_at", "updated_at", "version",
		})

		for _, item := range expectedItems {
			rows.AddRow(
				item.ID,
				item.UserID,
				item.Name,
				item.Type,
				item.Data,
				item.Metadata,
				item.CreatedAt,
				item.UpdatedAt,
				item.Version,
			)
		}

		mock.ExpectQuery("SELECT (.+) FROM data_items").
			WithArgs(userID).
			WillReturnRows(rows)

		items, err := repo.GetByUserID(context.Background(), userID)
		assert.NoError(t, err)
		assert.Equal(t, expectedItems, items)
	})

	t.Run("empty result", func(t *testing.T) {
		userID := uuid.New()

		rows := pgxmock.NewRows([]string{
			"id", "user_id", "name", "type", "encrypted_data",
			"metadata", "created_at", "updated_at", "version",
		})

		mock.ExpectQuery("SELECT (.+) FROM data_items").
			WithArgs(userID).
			WillReturnRows(rows)

		items, err := repo.GetByUserID(context.Background(), userID)
		assert.NoError(t, err)
		assert.Empty(t, items)
	})

	t.Run("database error", func(t *testing.T) {
		userID := uuid.New()

		mock.ExpectQuery("SELECT (.+) FROM data_items").
			WithArgs(userID).
			WillReturnError(fmt.Errorf("database error"))

		items, err := repo.GetByUserID(context.Background(), userID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get data items by user ID")
		assert.Nil(t, items)
	})

	t.Run("scan error", func(t *testing.T) {
		userID := uuid.New()

		rows := pgxmock.NewRows([]string{
			"id", "user_id", "name", "type", "encrypted_data",
			"metadata", "created_at", "updated_at", "version",
		}).AddRow(
			"invalid-uuid", // Некорректные данные для сканирования
			userID,
			"test",
			models.DataTypeText,
			[]byte("data"),
			[]byte("metadata"),
			time.Now(),
			time.Now(),
			1,
		)

		mock.ExpectQuery("SELECT (.+) FROM data_items").
			WithArgs(userID).
			WillReturnRows(rows)

		items, err := repo.GetByUserID(context.Background(), userID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to scan data item")
		assert.Nil(t, items)
	})

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDataRepository_GetByUserIDAndType(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := &dataRepository{db: mock}

	t.Run("successful get", func(t *testing.T) {
		userID := uuid.New()
		dataType := models.DataTypeText
		expectedItems := []*models.DataItem{
			{
				ID:        uuid.New(),
				UserID:    userID,
				Name:      "test-data-1",
				Type:      dataType,
				Data:      []byte("encrypted data 1"),
				Metadata:  []byte("metadata 1"),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Version:   1,
			},
			{
				ID:        uuid.New(),
				UserID:    userID,
				Name:      "test-data-2",
				Type:      dataType,
				Data:      []byte("encrypted data 2"),
				Metadata:  []byte("metadata 2"),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Version:   1,
			},
		}

		rows := pgxmock.NewRows([]string{
			"id", "user_id", "name", "type", "encrypted_data",
			"metadata", "created_at", "updated_at", "version",
		})

		for _, item := range expectedItems {
			rows.AddRow(
				item.ID,
				item.UserID,
				item.Name,
				item.Type,
				item.Data,
				item.Metadata,
				item.CreatedAt,
				item.UpdatedAt,
				item.Version,
			)
		}

		mock.ExpectQuery("SELECT (.+) FROM data_items").
			WithArgs(userID, dataType).
			WillReturnRows(rows)

		items, err := repo.GetByUserIDAndType(context.Background(), userID, dataType)
		assert.NoError(t, err)
		assert.Equal(t, expectedItems, items)
	})

	t.Run("empty result", func(t *testing.T) {
		userID := uuid.New()
		dataType := models.DataTypeText

		rows := pgxmock.NewRows([]string{
			"id", "user_id", "name", "type", "encrypted_data",
			"metadata", "created_at", "updated_at", "version",
		})

		mock.ExpectQuery("SELECT (.+) FROM data_items").
			WithArgs(userID, dataType).
			WillReturnRows(rows)

		items, err := repo.GetByUserIDAndType(context.Background(), userID, dataType)
		assert.NoError(t, err)
		assert.Empty(t, items)
	})

	t.Run("database error", func(t *testing.T) {
		userID := uuid.New()
		dataType := models.DataTypeText

		mock.ExpectQuery("SELECT (.+) FROM data_items").
			WithArgs(userID, dataType).
			WillReturnError(fmt.Errorf("database error"))

		items, err := repo.GetByUserIDAndType(context.Background(), userID, dataType)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get data items by user ID and type")
		assert.Nil(t, items)
	})

	t.Run("scan error", func(t *testing.T) {
		userID := uuid.New()
		dataType := models.DataTypeText

		rows := pgxmock.NewRows([]string{
			"id", "user_id", "name", "type", "encrypted_data",
			"metadata", "created_at", "updated_at", "version",
		}).AddRow(
			"invalid-uuid", // Некорректные данные для сканирования
			userID,
			"test",
			dataType,
			[]byte("data"),
			[]byte("metadata"),
			time.Now(),
			time.Now(),
			1,
		)

		mock.ExpectQuery("SELECT (.+) FROM data_items").
			WithArgs(userID, dataType).
			WillReturnRows(rows)

		items, err := repo.GetByUserIDAndType(context.Background(), userID, dataType)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to scan data item")
		assert.Nil(t, items)
	})

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDataRepository_Update(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := &dataRepository{db: mock}

	t.Run("successful update", func(t *testing.T) {
		data := &models.DataItem{
			ID:        uuid.New(),
			Name:      "updated-name",
			Data:      []byte("updated encrypted data"),
			Metadata:  []byte("updated metadata"),
			UpdatedAt: time.Now(),
			Version:   2,
		}

		mock.ExpectExec("UPDATE data_items").
			WithArgs(
				data.ID,
				data.Name,
				data.Data,
				data.Metadata,
				data.UpdatedAt,
				data.Version,
			).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := repo.Update(context.Background(), data)
		assert.NoError(t, err)
	})

	t.Run("data not found", func(t *testing.T) {
		data := &models.DataItem{
			ID:        uuid.New(),
			Name:      "non-existent",
			Data:      []byte("data"),
			Metadata:  []byte("metadata"),
			UpdatedAt: time.Now(),
			Version:   1,
		}

		mock.ExpectExec("UPDATE data_items").
			WithArgs(
				data.ID,
				data.Name,
				data.Data,
				data.Metadata,
				data.UpdatedAt,
				data.Version,
			).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		err := repo.Update(context.Background(), data)
		assert.Equal(t, models.ErrDataNotFound, err)
	})

	t.Run("database error", func(t *testing.T) {
		data := &models.DataItem{
			ID:        uuid.New(),
			Name:      "error-data",
			Data:      []byte("data"),
			Metadata:  []byte("metadata"),
			UpdatedAt: time.Now(),
			Version:   1,
		}

		mock.ExpectExec("UPDATE data_items").
			WithArgs(
				data.ID,
				data.Name,
				data.Data,
				data.Metadata,
				data.UpdatedAt,
				data.Version,
			).
			WillReturnError(fmt.Errorf("database error"))

		err := repo.Update(context.Background(), data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update data item")
	})

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDataRepository_Delete(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := &dataRepository{db: mock}

	t.Run("successful delete", func(t *testing.T) {
		id := uuid.New()

		mock.ExpectExec("UPDATE data_items SET deleted_at = NOW\\(\\) WHERE id = \\$1 AND deleted_at IS NULL").
			WithArgs(id).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := repo.Delete(context.Background(), id)
		assert.NoError(t, err)
	})

	t.Run("data not found", func(t *testing.T) {
		id := uuid.New()

		mock.ExpectExec("UPDATE data_items SET deleted_at = NOW\\(\\) WHERE id = \\$1 AND deleted_at IS NULL").
			WithArgs(id).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		err := repo.Delete(context.Background(), id)
		assert.Equal(t, models.ErrDataNotFound, err)
	})

	t.Run("database error", func(t *testing.T) {
		id := uuid.New()

		mock.ExpectExec("UPDATE data_items SET deleted_at = NOW\\(\\) WHERE id = \\$1 AND deleted_at IS NULL").
			WithArgs(id).
			WillReturnError(fmt.Errorf("database error"))

		err := repo.Delete(context.Background(), id)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete data item")
	})

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDataRepository_Count(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := &dataRepository{db: mock}

	t.Run("successful count", func(t *testing.T) {
		userID := uuid.New()
		expectedCount := int64(5)

		rows := pgxmock.NewRows([]string{"count"}).AddRow(expectedCount)

		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM data_items WHERE user_id = \\$1 AND deleted_at IS NULL").
			WithArgs(userID).
			WillReturnRows(rows)

		count, err := repo.Count(context.Background(), userID)
		assert.NoError(t, err)
		assert.Equal(t, expectedCount, count)
	})

	t.Run("empty result", func(t *testing.T) {
		userID := uuid.New()
		expectedCount := int64(0)

		rows := pgxmock.NewRows([]string{"count"}).AddRow(expectedCount)

		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM data_items WHERE user_id = \\$1 AND deleted_at IS NULL").
			WithArgs(userID).
			WillReturnRows(rows)

		count, err := repo.Count(context.Background(), userID)
		assert.NoError(t, err)
		assert.Equal(t, expectedCount, count)
	})

	t.Run("database error", func(t *testing.T) {
		userID := uuid.New()

		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM data_items WHERE user_id = \\$1 AND deleted_at IS NULL").
			WithArgs(userID).
			WillReturnError(fmt.Errorf("database error"))

		count, err := repo.Count(context.Background(), userID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to count data items")
		assert.Equal(t, int64(0), count)
	})

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDataRepository_GetModifiedAfter(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := &dataRepository{db: mock}
	userID := uuid.New()
	after := time.Now()

	expectedItems := []*models.DataItem{
		{
			ID:        uuid.New(),
			UserID:    userID,
			Name:      "test1",
			Type:      models.DataTypeText,
			Data:      []byte("data1"),
			Metadata:  []byte("metadata1"),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Version:   1,
		},
		{
			ID:        uuid.New(),
			UserID:    userID,
			Name:      "test2",
			Type:      models.DataTypeText,
			Data:      []byte("data2"),
			Metadata:  []byte("metadata2"),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Version:   2,
		},
	}

	rows := pgxmock.NewRows([]string{
		"id", "user_id", "name", "type", "encrypted_data", "metadata",
		"created_at", "updated_at", "version",
	})

	for _, item := range expectedItems {
		rows.AddRow(
			item.ID, item.UserID, item.Name, item.Type, item.Data,
			item.Metadata, item.CreatedAt, item.UpdatedAt, item.Version,
		)
	}

	mock.ExpectQuery("SELECT (.+) FROM data_items").
		WithArgs(userID, after).
		WillReturnRows(rows)

	items, err := repo.GetModifiedAfter(context.Background(), userID, after)
	assert.NoError(t, err)
	assert.Equal(t, expectedItems, items)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSyncRepository_CreateEvent(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := &syncRepository{db: mock}
	event := &models.SyncEvent{
		UserID:  uuid.New(),
		DataID:  uuid.New(),
		Action:  "add",
		Created: time.Now(),
		Version: 1,
	}

	expectedID := int64(1)
	rows := pgxmock.NewRows([]string{"id"}).AddRow(expectedID)

	mock.ExpectQuery("INSERT INTO sync_events").
		WithArgs(
			event.UserID,
			event.DataID,
			event.Action,
			event.Created,
			event.Version,
		).
		WillReturnRows(rows)

	err = repo.CreateEvent(context.Background(), event)
	assert.NoError(t, err)
	assert.Equal(t, expectedID, event.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSyncRepository_GetEventsAfter(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := &syncRepository{db: mock}

	t.Run("successful get events", func(t *testing.T) {
		userID := uuid.New()
		afterEventID := uuid.New()
		expectedEvents := []*models.SyncEvent{
			{
				ID:      int64(1),
				UserID:  userID,
				DataID:  uuid.New(),
				Action:  "add",
				Created: time.Now(),
				Version: 1,
			},
			{
				ID:      int64(2),
				UserID:  userID,
				DataID:  uuid.New(),
				Action:  "update",
				Created: time.Now().Add(time.Hour),
				Version: 2,
			},
		}

		rows := pgxmock.NewRows([]string{
			"id", "user_id", "data_id", "event_type",
			"timestamp", "data_version",
		})

		for _, event := range expectedEvents {
			rows.AddRow(
				event.ID,
				event.UserID,
				event.DataID,
				event.Action,
				event.Created,
				event.Version,
			)
		}

		mock.ExpectQuery("SELECT (.+) FROM sync_events").
			WithArgs(userID, afterEventID).
			WillReturnRows(rows)

		events, err := repo.GetEventsAfter(context.Background(), userID, afterEventID)
		assert.NoError(t, err)
		assert.Equal(t, expectedEvents, events)
	})

	t.Run("empty result", func(t *testing.T) {
		userID := uuid.New()
		afterEventID := uuid.New()

		rows := pgxmock.NewRows([]string{
			"id", "user_id", "data_id", "event_type",
			"timestamp", "data_version",
		})

		mock.ExpectQuery("SELECT (.+) FROM sync_events").
			WithArgs(userID, afterEventID).
			WillReturnRows(rows)

		events, err := repo.GetEventsAfter(context.Background(), userID, afterEventID)
		assert.NoError(t, err)
		assert.Empty(t, events)
	})

	t.Run("database error", func(t *testing.T) {
		userID := uuid.New()
		afterEventID := uuid.New()

		mock.ExpectQuery("SELECT (.+) FROM sync_events").
			WithArgs(userID, afterEventID).
			WillReturnError(fmt.Errorf("database error"))

		events, err := repo.GetEventsAfter(context.Background(), userID, afterEventID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get sync events")
		assert.Nil(t, events)
	})

	t.Run("scan error", func(t *testing.T) {
		userID := uuid.New()
		afterEventID := uuid.New()

		rows := pgxmock.NewRows([]string{
			"id", "user_id", "data_id", "event_type",
			"timestamp", "data_version",
		}).AddRow(
			"invalid-uuid", // Некорректные данные для сканирования
			userID,
			uuid.New(),
			"add",
			time.Now(),
			1,
		)

		mock.ExpectQuery("SELECT (.+) FROM sync_events").
			WithArgs(userID, afterEventID).
			WillReturnRows(rows)

		events, err := repo.GetEventsAfter(context.Background(), userID, afterEventID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to scan sync event")
		assert.Nil(t, events)
	})

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSyncRepository_GetLastEventID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := &syncRepository{db: mock}

	t.Run("successful get last event ID", func(t *testing.T) {
		userID := uuid.New()
		expectedLastID := uuid.New()

		rows := pgxmock.NewRows([]string{"id"}).
			AddRow(expectedLastID)

		mock.ExpectQuery("SELECT id FROM sync_events WHERE user_id = \\$1").
			WithArgs(userID).
			WillReturnRows(rows)

		lastID, err := repo.GetLastEventID(context.Background(), userID)
		assert.NoError(t, err)
		assert.Equal(t, expectedLastID, lastID)
	})

	t.Run("no events found", func(t *testing.T) {
		userID := uuid.New()

		mock.ExpectQuery("SELECT id FROM sync_events WHERE user_id = \\$1").
			WithArgs(userID).
			WillReturnError(pgx.ErrNoRows)

		lastID, err := repo.GetLastEventID(context.Background(), userID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get last event ID")
		assert.Equal(t, uuid.Nil, lastID)
	})

	t.Run("database error", func(t *testing.T) {
		userID := uuid.New()

		mock.ExpectQuery("SELECT id FROM sync_events WHERE user_id = \\$1").
			WithArgs(userID).
			WillReturnError(fmt.Errorf("database error"))

		lastID, err := repo.GetLastEventID(context.Background(), userID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get last event ID")
		assert.Equal(t, uuid.Nil, lastID)
	})

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSyncRepository_DeleteOldEvents(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := &syncRepository{db: mock}

	t.Run("successful delete", func(t *testing.T) {
		beforeTime := time.Now().Add(-24 * time.Hour)

		mock.ExpectExec("DELETE FROM sync_events WHERE timestamp < \\$1").
			WithArgs(beforeTime).
			WillReturnResult(pgxmock.NewResult("DELETE", 5))

		err := repo.DeleteOldEvents(context.Background(), beforeTime)
		assert.NoError(t, err)
	})

	t.Run("no events to delete", func(t *testing.T) {
		beforeTime := time.Now().Add(-24 * time.Hour)

		mock.ExpectExec("DELETE FROM sync_events WHERE timestamp < \\$1").
			WithArgs(beforeTime).
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		err := repo.DeleteOldEvents(context.Background(), beforeTime)
		assert.NoError(t, err)
	})

	t.Run("database error", func(t *testing.T) {
		beforeTime := time.Now().Add(-24 * time.Hour)

		mock.ExpectExec("DELETE FROM sync_events WHERE timestamp < \\$1").
			WithArgs(beforeTime).
			WillReturnError(fmt.Errorf("database error"))

		err := repo.DeleteOldEvents(context.Background(), beforeTime)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete old sync events")
	})

	assert.NoError(t, mock.ExpectationsWereMet())
}
func TestSyncRepository_GetEventsByDataID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := &syncRepository{db: mock}

	t.Run("successful get", func(t *testing.T) {
		userID := uuid.New()
		dataID := uuid.New()
		expectedEvents := []*models.SyncEvent{
			{
				ID:      int64(1),
				UserID:  userID,
				DataID:  dataID,
				Action:  "update",
				Created: time.Now(),
				Version: 1,
			},
			{
				ID:      int64(2),
				UserID:  userID,
				DataID:  dataID,
				Action:  "add",
				Created: time.Now().Add(time.Hour),
				Version: 2,
			},
		}

		rows := pgxmock.NewRows([]string{
			"id", "user_id", "data_id", "event_type", "timestamp", "data_version",
		})

		for _, event := range expectedEvents {
			rows.AddRow(
				event.ID,
				event.UserID,
				event.DataID,
				event.Action,
				event.Created,
				event.Version,
			)
		}

		mock.ExpectQuery("SELECT (.+) FROM sync_events").
			WithArgs(userID, dataID).
			WillReturnRows(rows)

		events, err := repo.GetEventsByDataID(context.Background(), userID, dataID)
		assert.NoError(t, err)
		assert.Equal(t, expectedEvents, events)
	})

	t.Run("empty result", func(t *testing.T) {
		userID := uuid.New()
		dataID := uuid.New()

		rows := pgxmock.NewRows([]string{
			"id", "user_id", "data_id", "event_type", "timestamp", "data_version",
		})

		mock.ExpectQuery("SELECT (.+) FROM sync_events").
			WithArgs(userID, dataID).
			WillReturnRows(rows)

		events, err := repo.GetEventsByDataID(context.Background(), userID, dataID)
		assert.NoError(t, err)
		assert.Empty(t, events)
	})

	t.Run("database error", func(t *testing.T) {
		userID := uuid.New()
		dataID := uuid.New()

		mock.ExpectQuery("SELECT (.+) FROM sync_events").
			WithArgs(userID, dataID).
			WillReturnError(fmt.Errorf("database error"))

		events, err := repo.GetEventsByDataID(context.Background(), userID, dataID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get sync events by data ID")
		assert.Nil(t, events)
	})

	t.Run("scan error", func(t *testing.T) {
		userID := uuid.New()
		dataID := uuid.New()

		rows := pgxmock.NewRows([]string{
			"id", "user_id", "data_id", "event_type", "timestamp", "data_version",
		}).AddRow(
			"invalid-uuid", // Invalid UUID that will cause scan error
			userID,
			dataID,
			"add",
			time.Now(),
			1,
		)

		mock.ExpectQuery("SELECT (.+) FROM sync_events").
			WithArgs(userID, dataID).
			WillReturnRows(rows)

		events, err := repo.GetEventsByDataID(context.Background(), userID, dataID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to scan sync event")
		assert.Nil(t, events)
	})

	assert.NoError(t, mock.ExpectationsWereMet())
}
