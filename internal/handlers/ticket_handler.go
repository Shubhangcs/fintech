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

type TicketHandler struct {
	ticketStore store.TicketStore
	logger      *slog.Logger
}

func NewTicketHandler(ticketStore store.TicketStore, logger *slog.Logger) *TicketHandler {
	return &TicketHandler{ticketStore: ticketStore, logger: logger}
}

func (th *TicketHandler) HandleCreateTicket(w http.ResponseWriter, r *http.Request) {
	var t models.TicketModel
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		utils.BadRequest(w, th.logger, "create ticket", err)
		return
	}

	if err := t.Validate(); err != nil {
		utils.BadRequest(w, th.logger, "create ticket", err)
		return
	}

	if err := th.ticketStore.CreateTicket(&t); err != nil {
		utils.ServerError(w, th.logger, "create ticket", err)
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"ticket": t})
}

func (th *TicketHandler) HandleUpdateTicket(w http.ResponseWriter, r *http.Request) {
	id, err := readTicketID(r)
	if err != nil {
		utils.BadRequest(w, th.logger, "update ticket", err)
		return
	}

	var t models.TicketModel
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		utils.BadRequest(w, th.logger, "update ticket", err)
		return
	}

	if err := th.ticketStore.UpdateTicket(id, &t); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, th.logger, "update ticket", errors.New("ticket not found"))
			return
		}
		utils.ServerError(w, th.logger, "update ticket", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "ticket updated successfully"})
}

func (th *TicketHandler) HandleDeleteTicket(w http.ResponseWriter, r *http.Request) {
	id, err := readTicketID(r)
	if err != nil {
		utils.BadRequest(w, th.logger, "delete ticket", err)
		return
	}

	if err := th.ticketStore.DeleteTicket(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, th.logger, "delete ticket", errors.New("ticket not found"))
			return
		}
		utils.ServerError(w, th.logger, "delete ticket", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "ticket deleted successfully"})
}

func (th *TicketHandler) HandleUpdateTicketClearStatus(w http.ResponseWriter, r *http.Request) {
	id, err := readTicketID(r)
	if err != nil {
		utils.BadRequest(w, th.logger, "update ticket clear status", err)
		return
	}

	var body struct {
		IsTicketCleared bool `json:"is_ticket_cleared"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.BadRequest(w, th.logger, "update ticket clear status", err)
		return
	}

	if err := th.ticketStore.UpdateTicketClearStatus(id, body.IsTicketCleared); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, th.logger, "update ticket clear status", errors.New("ticket not found"))
			return
		}
		utils.ServerError(w, th.logger, "update ticket clear status", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "ticket clear status updated successfully"})
}

func (th *TicketHandler) HandleGetTicketsByUserID(w http.ResponseWriter, r *http.Request) {
	userID, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, th.logger, "get tickets by user id", err)
		return
	}

	p := utils.ReadPaginationParams(r)
	startDate := utils.ParseDateParam(r, "start_date")
	endDate := utils.ParseDateParam(r, "end_date")

	tickets, err := th.ticketStore.GetTicketsByUserID(userID, p.Limit, p.Offset, startDate, endDate)
	if err != nil {
		utils.ServerError(w, th.logger, "get tickets by user id", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"tickets": tickets})
}

func (th *TicketHandler) HandleGetAllTickets(w http.ResponseWriter, r *http.Request) {
	p := utils.ReadPaginationParams(r)
	startDate := utils.ParseDateParam(r, "start_date")
	endDate := utils.ParseDateParam(r, "end_date")

	tickets, err := th.ticketStore.GetAllTickets(p.Limit, p.Offset, startDate, endDate)
	if err != nil {
		utils.ServerError(w, th.logger, "get all tickets", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"tickets": tickets})
}

func readTicketID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		return 0, errors.New("invalid ticket id")
	}
	return id, nil
}
