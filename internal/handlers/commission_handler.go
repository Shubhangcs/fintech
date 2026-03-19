package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/levionstudio/fintech/internal/models"
	"github.com/levionstudio/fintech/internal/store"
	"github.com/levionstudio/fintech/internal/utils"
)

type CommissionHandler struct {
	commissionStore store.CommissionStore
	logger          *slog.Logger
}

func NewCommissionHandler(commissionStore store.CommissionStore, logger *slog.Logger) *CommissionHandler {
	return &CommissionHandler{commissionStore: commissionStore, logger: logger}
}

func (ch *CommissionHandler) HandleCreateCommission(w http.ResponseWriter, r *http.Request) {
	var c models.CommissionModel
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		utils.BadRequest(w, ch.logger, "create commission", err)
		return
	}

	if err := c.Validate(); err != nil {
		utils.BadRequest(w, ch.logger, "create commission", err)
		return
	}

	if err := ch.commissionStore.CreateCommission(&c); err != nil {
		utils.ServerError(w, ch.logger, "create commission", err)
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"commission": c})
}

func (ch *CommissionHandler) HandleUpdateCommission(w http.ResponseWriter, r *http.Request) {
	id, err := readCommissionID(r)
	if err != nil {
		utils.BadRequest(w, ch.logger, "update commission", err)
		return
	}

	var c models.CommissionModel
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		utils.BadRequest(w, ch.logger, "update commission", err)
		return
	}

	if err := ch.commissionStore.UpdateCommission(id, &c); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, ch.logger, "update commission", errors.New("commission not found"))
			return
		}
		utils.ServerError(w, ch.logger, "update commission", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "commission updated successfully"})
}

func (ch *CommissionHandler) HandleDeleteCommission(w http.ResponseWriter, r *http.Request) {
	id, err := readCommissionID(r)
	if err != nil {
		utils.BadRequest(w, ch.logger, "delete commission", err)
		return
	}

	if err := ch.commissionStore.DeleteCommission(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, ch.logger, "delete commission", errors.New("commission not found"))
			return
		}
		utils.ServerError(w, ch.logger, "delete commission", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "commission deleted successfully"})
}

func (ch *CommissionHandler) HandleGetCommissions(w http.ResponseWriter, r *http.Request) {
	p := utils.ReadPaginationParams(r)

	commissions, err := ch.commissionStore.GetCommissions(p.Limit, p.Offset)
	if err != nil {
		utils.ServerError(w, ch.logger, "get commissions", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"commissions": commissions})
}

func readCommissionID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		return 0, errors.New("invalid commission id")
	}
	return id, nil
}
