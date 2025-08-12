package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/zhenyanesterkova/safehub/internal/models"
	"github.com/zhenyanesterkova/safehub/internal/server/services"
)

type DataHandler struct {
	dataService *services.DataService
	logger      *services.LogService
}

func NewDataHandler(
	dataService *services.DataService,
	logger *services.LogService,
) *DataHandler {
	return &DataHandler{
		dataService: dataService,
		logger:      logger,
	}
}

// CreateData создает новые данные пользователя
// POST /api/v1/data
func (h *DataHandler) CreateData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user, ok := ctx.Value("user").(*models.User)
	if !ok {
		h.logger.Log.Error("failed get user from context when CreateData handler")
		http.Error(w, StatusInternalServerError, http.StatusInternalServerError)
		return
	}

	var req services.CreateDataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	response, err := h.dataService.Create(ctx, user.ID.String(), req)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidDataType):
			http.Error(w, "Invalid data type", http.StatusBadRequest)
			return
		case errors.Is(err, services.ErrDataValidationFailed):
			http.Error(w, "Invalid data", http.StatusBadRequest)
			return
		default:
			h.logger.Log.Errorf("failed create data: %v", err)
			http.Error(w, StatusInternalServerError, http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		h.logger.Log.Errorf("Error encoding response: %v", err)
		http.Error(w, StatusInternalServerError, http.StatusInternalServerError)
		return
	}
}

// GetData получает данные по ID
// GET /api/v1/data/{id}
func (h *DataHandler) GetData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user, ok := ctx.Value("user").(*models.User)
	if !ok {
		h.logger.Log.Error("failed get user from context when GetData handler")
		http.Error(w, StatusInternalServerError, http.StatusInternalServerError)
		return
	}

	dataID := chi.URLParam(r, "id")
	if dataID == "" {
		http.Error(w, "Data ID is required", http.StatusBadRequest)
		return
	}

	response, err := h.dataService.GetByID(ctx, user.ID.String(), dataID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrDataNotFound):
			http.Error(w, "Data not found", http.StatusNotFound)
			return
		case errors.Is(err, services.ErrDataAccessDenied):
			http.Error(w, "Access denied to this data", http.StatusForbidden)
			return
		default:
			h.logger.Log.Errorf("Error get data by id: %v", err)
			http.Error(w, StatusInternalServerError, http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		h.logger.Log.Errorf("Error encoding response: %v", err)
		http.Error(w, StatusInternalServerError, http.StatusInternalServerError)
		return
	}
}

// ListData получает список данных пользователя
// GET /api/v1/data
func (h *DataHandler) ListData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user, ok := ctx.Value("user").(*models.User)
	if !ok {
		h.logger.Log.Error("failed get user from context when GetData handler")
		http.Error(w, StatusInternalServerError, http.StatusInternalServerError)
		return
	}

	responses, err := h.dataService.List(ctx, user.ID.String())
	if err != nil {
		h.logger.Log.Errorf("failed get list data: %v", err)
		http.Error(w, StatusInternalServerError, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(responses)
	if err != nil {
		h.logger.Log.Errorf("Error encoding response: %v", err)
		http.Error(w, StatusInternalServerError, http.StatusInternalServerError)
		return
	}
}

// UpdateData обновляет данные пользователя
// PUT /api/v1/data/{id}
func (h *DataHandler) UpdateData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user, ok := ctx.Value("user").(*models.User)
	if !ok {
		h.logger.Log.Error("failed get user from context when GetData handler")
		http.Error(w, StatusInternalServerError, http.StatusInternalServerError)
		return
	}

	dataID := chi.URLParam(r, "id")
	if dataID == "" {
		http.Error(w, "Data ID is required", http.StatusBadRequest)
		return
	}

	var req services.UpdateDataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	response, err := h.dataService.Update(ctx, user.ID.String(), dataID, req)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrDataNotFound):
			http.Error(w, "Data not found", http.StatusNotFound)
			return
		case errors.Is(err, services.ErrDataAccessDenied):
			http.Error(w, "Access denied to this data", http.StatusNotFound)
			return
		case errors.Is(err, services.ErrDataValidationFailed):
			http.Error(w, "Invalid data", http.StatusBadRequest)
			return
		default:
			h.logger.Log.Errorf("failed update data: %v", err)
			http.Error(w, StatusInternalServerError, http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		h.logger.Log.Errorf("Error encoding response: %v", err)
		http.Error(w, StatusInternalServerError, http.StatusInternalServerError)
		return
	}
}

// DeleteData удаляет данные пользователя
// DELETE /api/v1/data/{id}
func (h *DataHandler) DeleteData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user, ok := ctx.Value("user").(*models.User)
	if !ok {
		h.logger.Log.Error("failed get user from context when GetData handler")
		http.Error(w, StatusInternalServerError, http.StatusInternalServerError)
		return
	}

	dataID := chi.URLParam(r, "id")
	if dataID == "" {
		http.Error(w, "Data ID is required", http.StatusBadRequest)
		return
	}

	err := h.dataService.Delete(ctx, user.ID.String(), dataID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrDataNotFound):
			http.Error(w, "Data not found", http.StatusNotFound)
			return
		case errors.Is(err, services.ErrDataAccessDenied):
			http.Error(w, "Access denied to this data", http.StatusNotFound)
			return
		case errors.Is(err, services.ErrDataValidationFailed):
			http.Error(w, "Invalid data", http.StatusBadRequest)
			return
		default:
			h.logger.Log.Errorf("failed update data: %v", err)
			http.Error(w, StatusInternalServerError, http.StatusInternalServerError)
			return
		}
	}
}

// GetDataStats получает статистику по данным пользователя
// GET /api/v1/data/stats
func (h *DataHandler) GetDataStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user, ok := ctx.Value("user").(*models.User)
	if !ok {
		h.logger.Log.Error("failed get user from context when GetData handler")
		http.Error(w, StatusInternalServerError, http.StatusInternalServerError)
		return
	}

	stats, err := h.dataService.GetStats(ctx, user.ID.String())
	if err != nil {
		h.logger.Log.Errorf("failed get statistic data: %v", err)
		http.Error(w, StatusInternalServerError, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(strconv.FormatInt(stats, 10)))
	if err != nil {
		h.logger.Log.Errorf("Error encoding response: %v", err)
		http.Error(w, StatusInternalServerError, http.StatusInternalServerError)
		return
	}
}

// GetDataTypes возвращает список доступных типов данных
// GET /api/v1/data/types
func (h *DataHandler) GetDataTypes(w http.ResponseWriter, r *http.Request) {
	types := []map[string]interface{}{
		{
			"type":        models.DataTypeCredentials,
			"name":        "Credentials",
			"description": "Login and password pairs",
		},
		{
			"type":        models.DataTypeText,
			"name":        "Text",
			"description": "Arbitrary text data",
		},
		{
			"type":        models.DataTypeBinary,
			"name":        "Binary",
			"description": "Arbitrary binary data",
		},
		{
			"type":        models.DataTypeCard,
			"name":        "Card",
			"description": "Bank card information",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err := json.NewEncoder(w).Encode(types)
	if err != nil {
		h.logger.Log.Errorf("Error encoding response: %v", err)
		http.Error(w, StatusInternalServerError, http.StatusInternalServerError)
		return
	}
}
