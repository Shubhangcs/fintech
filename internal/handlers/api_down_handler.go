package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/levionstudio/fintech/internal/store"
	"github.com/levionstudio/fintech/internal/utils"
)

type ApiDownHandler struct {
	apiDownStore store.ApiDownStore
	logger       *slog.Logger
}

func NewApiDownHandler(apiDownStore store.ApiDownStore, logger *slog.Logger) *ApiDownHandler {
	return &ApiDownHandler{apiDownStore: apiDownStore, logger: logger}
}

func (ah *ApiDownHandler) HandleGetAllServiceStatuses(w http.ResponseWriter, r *http.Request) {
	statuses, err := ah.apiDownStore.GetAllServiceStatuses()
	if err != nil {
		utils.ServerError(w, ah.logger, "get all service statuses", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"services": statuses})
}

func (ah *ApiDownHandler) HandleUpdateServiceStatus(w http.ResponseWriter, r *http.Request) {
	serviceName := chi.URLParam(r, "service_name")
	if serviceName == "" {
		utils.BadRequest(w, ah.logger, "update service status", errors.New("service_name is required"))
		return
	}

	var req struct {
		Status bool `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, ah.logger, "update service status", err)
		return
	}

	if err := ah.apiDownStore.UpdateServiceStatus(serviceName, req.Status); err != nil {
		utils.ServerError(w, ah.logger, "update service status", err)
		return
	}

	msg := "service is now up"
	if req.Status {
		msg = "service is now down"
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": msg, "service": serviceName, "status": req.Status})
}
