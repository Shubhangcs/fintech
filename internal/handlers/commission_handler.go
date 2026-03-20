package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/levionstudio/fintech/internal/models"
	"github.com/levionstudio/fintech/internal/store"
	"github.com/levionstudio/fintech/internal/utils"
)

type CommisionHandler struct {
	commisionStore store.CommisionStore
	logger         *slog.Logger
}

func NewCommisionHandler(commisionStore store.CommisionStore, logger *slog.Logger) *CommisionHandler {
	return &CommisionHandler{commisionStore: commisionStore, logger: logger}
}

// Create Commision Handler
func (ch *CommisionHandler) HandleCreateCommision(w http.ResponseWriter, r *http.Request) {
	var c models.CommisionModel
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		utils.BadRequest(w, ch.logger, "create commision", err)
		return
	}

	if err := c.Validate(); err != nil {
		utils.BadRequest(w, ch.logger, "create commision", err)
		return
	}

	if err := ch.commisionStore.CreateCommision(&c); err != nil {
		utils.ServerError(w, ch.logger, "create commision", err)
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"message": "commision created successfully", "commision": c})
}

// Update Commision Handler
func (ch *CommisionHandler) HandleUpdateCommision(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamIDInt(r)
	if err != nil {
		utils.BadRequest(w, ch.logger, "update commision", err)
		return
	}

	var c models.CommisionModel
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		utils.BadRequest(w, ch.logger, "update commision", err)
		return
	}

	c.CommisionID = id
	if err := ch.commisionStore.UpdateCommision(&c); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, ch.logger, "update commision", errors.New("commision not found"))
			return
		}
		utils.ServerError(w, ch.logger, "update commision", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "commision updated successfully"})
}

// Delete Commision Handler
func (ch *CommisionHandler) HandleDeleteCommision(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamIDInt(r)
	if err != nil {
		utils.BadRequest(w, ch.logger, "delete commision", err)
		return
	}

	if err := ch.commisionStore.DeleteCommision(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, ch.logger, "delete commision", errors.New("commision not found"))
			return
		}
		utils.ServerError(w, ch.logger, "delete commision", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "commision deleted successfully"})
}

// Get Commisions Handler
func (ch *CommisionHandler) HandleGetCommisions(w http.ResponseWriter, r *http.Request) {
	p := utils.ReadPaginationParams(r)

	commisions, err := ch.commisionStore.GetAllCommisions(p.Limit, p.Offset)
	if err != nil {
		utils.ServerError(w, ch.logger, "get commisions", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "commisions fetched successfully", "commisions": commisions})
}
