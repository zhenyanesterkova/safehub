package handlers

import (
	"encoding/json"

	"github.com/zhenyanesterkova/safehub/internal/models"
	"github.com/zhenyanesterkova/safehub/internal/server/storage"
	"github.com/zhenyanesterkova/safehub/internal/shared/crypto"
)

type DataHandler struct {
	dataRepo  storage.DataRepository
	syncRepo  storage.SyncRepository
	cryptoSvc *crypto.CryptoService
}

func NewDataHandler(dataRepo storage.DataRepository, syncRepo storage.SyncRepository, cryptoSvc *crypto.CryptoService) *DataHandler {
	return &DataHandler{
		dataRepo:  dataRepo,
		syncRepo:  syncRepo,
		cryptoSvc: cryptoSvc,
	}
}

type CreateDataRequest struct {
	Type     models.DataType `json:"type" binding:"required"`
	Name     string          `json:"name" binding:"required"`
	Data     json.RawMessage `json:"data" binding:"required"`
	Metadata string          `json:"metadata,omitempty"`
}

// func (h *DataHandler) Create(c *gin.Context) {
// 	userID, exists := c.Get("user_id")
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
// 		return
// 	}

// 	var req CreateDataRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	// Валидируем тип данных
// 	if !isValidDataType(req.Type) {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid data type"})
// 		return
// 	}

// 	// Создаем элемент данных
// 	item := &models.DataItem{
// 		ID:        uuid.New(),
// 		UserID:    userID.(uuid.UUID),
// 		Type:      req.Type,
// 		Name:      req.Name,
// 		Data:      req.Data, // Данные уже з
