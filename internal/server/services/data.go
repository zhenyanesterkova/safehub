package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/zhenyanesterkova/safehub/internal/models"
	"github.com/zhenyanesterkova/safehub/internal/server/storage"
)

var (
	ErrDataNotFound         = errors.New("data not found")
	ErrDataAccessDenied     = errors.New("access denied to data")
	ErrInvalidDataType      = errors.New("invalid data type")
	ErrDataValidationFailed = errors.New("data validation failed")
)

// DataService предоставляет сервисы для работы с пользовательскими данными
type DataService struct {
	dataRepo storage.DataRepository
	syncRepo storage.SyncRepository
	log      *LogService
}

// NewDataService создает новый экземпляр DataService
func NewDataService(
	dataRepo storage.DataRepository,
	syncRepo storage.SyncRepository,
	log *LogService,
) *DataService {
	return &DataService{
		dataRepo: dataRepo,
		syncRepo: syncRepo,
		log:      log,
	}
}

// CreateDataRequest представляет запрос на создание данных
type CreateDataRequest struct {
	Name     string          `json:"name" validate:"required,min=1,max=255"`
	Type     models.DataType `json:"type" validate:"required,oneof=credentials text binary card"`
	Data     map[string]any  `json:"data" validate:"required"`
	Metadata string          `json:"metadata,omitempty"`
	ClientID string          `json:"client_id,omitempty"`
}

// UpdateDataRequest представляет запрос на обновление данных
type UpdateDataRequest struct {
	Name     string         `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Data     map[string]any `json:"data,omitempty"`
	Metadata string         `json:"metadata,omitempty"`
	ClientID string         `json:"client_id,omitempty"`
}

// DataResponse представляет ответ с данными
type DataResponse struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Type      models.DataType `json:"type"`
	Data      map[string]any  `json:"data"`
	Metadata  string          `json:"metadata"`
	Version   int64           `json:"version"`
	CreatedAt time.Time       `json:"created_at"`
}

// ListDataRequest представляет запрос на получение списка данных
type ListDataRequest struct {
	Type *models.DataType `json:"type,omitempty"`
}

// Create создает новые данные пользователя
func (s *DataService) Create(ctx context.Context, userID string, req CreateDataRequest) (*DataResponse, error) {
	if !s.isValidDataType(req.Type) {
		return nil, ErrInvalidDataType
	}

	if err := s.validateDataStructure(req.Type, req.Data); err != nil {
		return nil, fmt.Errorf("data validation failed: %w", err)
	}

	uID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	jsonData, err := json.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed marshal data map to json: %w", err)
	}

	jsonMetadata, err := json.Marshal(req.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed marshal metadata map to json: %w", err)
	}

	dataItem := &models.DataItem{
		UserID:    uID,
		Name:      req.Name,
		Type:      req.Type,
		Data:      jsonData,
		Metadata:  jsonMetadata,
		Version:   int64(1),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	dataItem, err = s.dataRepo.Create(ctx, dataItem)
	if err != nil {
		return nil, fmt.Errorf("failed to create data: %w", err)
	}

	syncEvent := &models.SyncEvent{
		UserID:   uID,
		DataID:   dataItem.ID,
		Action:   models.EventTypeCreate,
		Version:  int64(1),
		Created:  time.Now(),
		ClientID: req.ClientID,
	}

	if err := s.syncRepo.CreateEvent(ctx, syncEvent); err != nil {
		s.log.Log.Warningf("Warning: failed to create sync event: %v", err)
	}

	resp, err := s.modelToResponse(dataItem)
	if err != nil {
		return nil, fmt.Errorf("failed to convert model to response: %w", err)
	}

	return resp, nil
}

// GetByID получает данные по ID
func (s *DataService) GetByID(ctx context.Context, userID, dataID string) (*DataResponse, error) {
	uID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	dID, err := uuid.Parse(dataID)
	if err != nil {
		return nil, fmt.Errorf("invalid data ID: %w", err)
	}

	dataItem, err := s.dataRepo.GetByID(ctx, dID)
	if err != nil {
		if errors.Is(err, models.ErrDataNotFound) {
			return nil, ErrDataNotFound
		}
		return nil, fmt.Errorf("failed to get data: %w", err)
	}

	if dataItem.UserID != uID {
		return nil, ErrDataAccessDenied
	}

	resp, err := s.modelToResponse(dataItem)
	if err != nil {
		return nil, fmt.Errorf("failed to convert model to response: %w", err)
	}

	return resp, nil
}

// List получает список данных пользователя
func (s *DataService) List(ctx context.Context, userID string, req ListDataRequest) ([]*DataResponse, error) {
	uID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	dataItems, err := s.dataRepo.GetByUserID(ctx, uID)
	if err != nil {
		return nil, fmt.Errorf("failed to list data: %w", err)
	}

	responses := make([]*DataResponse, len(dataItems))
	for i, item := range dataItems {
		responses[i], err = s.modelToResponse(item)
		if err != nil {
			return nil, fmt.Errorf("failed to convert model to response: %w", err)
		}
	}

	return responses, nil
}

// Update обновляет данные пользователя
func (s *DataService) Update(ctx context.Context, userID, dataID string, req UpdateDataRequest) (*DataResponse, error) {
	uID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	dID, err := uuid.Parse(dataID)
	if err != nil {
		return nil, fmt.Errorf("invalid data ID: %w", err)
	}

	dataItem, err := s.dataRepo.GetByID(ctx, dID)
	if err != nil {
		if errors.Is(err, models.ErrDataNotFound) {
			return nil, ErrDataNotFound
		}
		return nil, fmt.Errorf("failed to get data: %w", err)
	}

	if dataItem.UserID != uID {
		return nil, ErrDataAccessDenied
	}

	if req.Name != "" {
		dataItem.Name = req.Name
	}
	if req.Data != nil {
		if err := s.validateDataStructure(dataItem.Type, req.Data); err != nil {
			return nil, fmt.Errorf("data validation failed: %w", err)
		}

		jsonData, err := json.Marshal(req.Data)
		if err != nil {
			return nil, fmt.Errorf("failed marshal data map to json: %w", err)
		}

		dataItem.Data = jsonData
	}

	if req.Metadata != "" {
		dataItem.Metadata = []byte(req.Metadata)
	}

	dataItem.Version++
	dataItem.UpdatedAt = time.Now()

	if err := s.dataRepo.Update(ctx, dataItem); err != nil {
		return nil, fmt.Errorf("failed to update data: %w", err)
	}

	syncEvent := &models.SyncEvent{
		UserID:   uID,
		DataID:   dID,
		Action:   models.EventTypeUpdate,
		Version:  dataItem.Version,
		Created:  time.Now(),
		ClientID: req.ClientID,
	}

	if err := s.syncRepo.CreateEvent(ctx, syncEvent); err != nil {
		s.log.Log.Warningf("Warning: failed to create sync event: %v", err)
	}

	resp, err := s.modelToResponse(dataItem)
	if err != nil {
		return nil, fmt.Errorf("failed to convert model to response: %w", err)
	}

	return resp, nil
}

// Delete удаляет данные пользователя
func (s *DataService) Delete(ctx context.Context, userID, dataID string) error {
	uID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	dID, err := uuid.Parse(dataID)
	if err != nil {
		return fmt.Errorf("invalid data ID: %w", err)
	}

	dataItem, err := s.dataRepo.GetByID(ctx, dID)
	if err != nil {
		if errors.Is(err, models.ErrDataNotFound) {
			return ErrDataNotFound
		}
		return fmt.Errorf("failed to get data: %w", err)
	}

	if dataItem.UserID != uID {
		return ErrDataAccessDenied
	}

	if err := s.dataRepo.Delete(ctx, dID); err != nil {
		return fmt.Errorf("failed to delete data: %w", err)
	}

	syncEvent := &models.SyncEvent{
		UserID:  uID,
		DataID:  dID,
		Action:  models.EventTypeDelete,
		Version: dataItem.Version + 1,
		Created: time.Now(),
	}

	if err := s.syncRepo.CreateEvent(ctx, syncEvent); err != nil {
		s.log.Log.Warningf("Warning: failed to create sync event: %v", err)
	}

	return nil
}

// GetStats возвращает статистику по данным пользователя
func (s *DataService) GetStats(ctx context.Context, userID string) (int64, error) {
	uID, err := uuid.Parse(userID)
	if err != nil {
		return 0, fmt.Errorf("invalid user ID: %w", err)
	}

	count, err := s.dataRepo.Count(ctx, uID)
	if err != nil {
		return 0, fmt.Errorf("failed to get count for type: %w", err)
	}

	return count, nil
}

// isValidDataType проверяет валидность типа данных
func (s *DataService) isValidDataType(dataType models.DataType) bool {
	switch dataType {
	case models.DataTypeCredentials, models.DataTypeText, models.DataTypeBinary, models.DataTypeCard:
		return true
	default:
		return false
	}
}

// validateDataStructure валидирует структуру данных в зависимости от типа
func (s *DataService) validateDataStructure(dataType models.DataType, data map[string]any) error {
	switch dataType {
	case models.DataTypeCredentials:
		return s.validateCredentials(data)
	case models.DataTypeText:
		return s.validateText(data)
	case models.DataTypeBinary:
		return s.validateBinary(data)
	case models.DataTypeCard:
		return s.validateCard(data)
	default:
		return ErrInvalidDataType
	}
}

// validateCredentials валидирует структуру данных для типа "credentials"
func (s *DataService) validateCredentials(data map[string]any) error {
	username, hasUsername := data["username"]
	password, hasPassword := data["password"]

	if !hasUsername || !hasPassword {
		return errors.New("credentials must contain username and password")
	}

	if _, ok := username.(string); !ok {
		return errors.New("username must be a string")
	}

	if _, ok := password.(string); !ok {
		return errors.New("password must be a string")
	}

	return nil
}

// validateText валидирует структуру данных для типа "text"
func (s *DataService) validateText(data map[string]any) error {
	content, hasContent := data["content"]
	if !hasContent {
		return errors.New("text data must contain content field")
	}

	if _, ok := content.(string); !ok {
		return errors.New("content must be a string")
	}

	return nil
}

// validateBinary валидирует структуру данных для типа "binary"
func (s *DataService) validateBinary(data map[string]any) error {
	content, hasContent := data["content"]
	filename, hasFilename := data["filename"]

	if !hasContent {
		return errors.New("binary data must contain content field")
	}

	if !hasFilename {
		return errors.New("binary data must contain filename field")
	}

	if _, ok := content.(string); !ok {
		return errors.New("content must be a string (base64 encoded)")
	}

	if _, ok := filename.(string); !ok {
		return errors.New("filename must be a string")
	}

	return nil
}

// validateCard валидирует структуру данных для типа "card"
func (s *DataService) validateCard(data map[string]any) error {
	number, hasNumber := data["number"]
	holder, hasHolder := data["holder"]
	expiry, hasExpiry := data["expiry"]
	cvv, hasCVV := data["cvv"]

	if !hasNumber || !hasHolder || !hasExpiry || !hasCVV {
		return errors.New("card data must contain number, holder, expiry, and cvv")
	}

	if _, ok := number.(string); !ok {
		return errors.New("card number must be a string")
	}

	if _, ok := holder.(string); !ok {
		return errors.New("card holder must be a string")
	}

	if _, ok := expiry.(string); !ok {
		return errors.New("card expiry must be a string")
	}

	if _, ok := cvv.(string); !ok {
		return errors.New("card cvv must be a string")
	}

	return nil
}

// modelToResponse преобразует модель в ответ
func (s *DataService) modelToResponse(item *models.DataItem) (*DataResponse, error) {
	var data map[string]any
	err := json.Unmarshal(item.Data, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return &DataResponse{
		ID:        item.ID.String(),
		Name:      item.Name,
		Type:      item.Type,
		Data:      data,
		Metadata:  string(item.Metadata),
		Version:   item.Version,
		CreatedAt: item.CreatedAt,
	}, nil
}
