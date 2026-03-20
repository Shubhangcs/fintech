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

type TicketHandler struct {
	ticketStore store.TicketStore
	logger      *slog.Logger
}

func NewTicketHandler(ticketStore store.TicketStore, logger *slog.Logger) *TicketHandler {
	return &TicketHandler{ticketStore: ticketStore, logger: logger}
}

// Create Ticket Handler
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

	adminID, err := th.ticketStore.GetAdminIDByUserID(t.UserID)
	if err != nil {
		utils.BadRequest(w, th.logger, "create ticket", err)
		return
	}
	t.AdminID = adminID

	if err := th.ticketStore.CreateTicket(&t); err != nil {
		utils.ServerError(w, th.logger, "create ticket", err)
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"message": "ticket created successfully", "ticket": t})
}

// Update Ticket Handler
func (th *TicketHandler) HandleUpdateTicket(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamIDInt(r)
	if err != nil {
		utils.BadRequest(w, th.logger, "update ticket", err)
		return
	}

	var t models.TicketModel
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		utils.BadRequest(w, th.logger, "update ticket", err)
		return
	}

	t.TicketID = id
	if err := th.ticketStore.UpdateTicket(&t); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, th.logger, "update ticket", errors.New("ticket not found"))
			return
		}
		utils.ServerError(w, th.logger, "update ticket", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "ticket updated successfully"})
}

// Delete Ticket Handler
func (th *TicketHandler) HandleDeleteTicket(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamIDInt(r)
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

// Update Ticket Clear Status Handler
func (th *TicketHandler) HandleUpdateTicketClearStatus(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamIDInt(r)
	if err != nil {
		utils.BadRequest(w, th.logger, "update ticket clear status", err)
		return
	}

	var req models.TicketModel
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, th.logger, "update ticket clear status", err)
		return
	}

	req.TicketID = id
	if err := th.ticketStore.UpdateTicketClearStatus(&req); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.BadRequest(w, th.logger, "update ticket clear status", errors.New("ticket not found"))
			return
		}
		utils.ServerError(w, th.logger, "update ticket clear status", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "ticket clear status updated successfully"})
}

// Get All Tickets Handler
func (th *TicketHandler) HandleGetAllTickets(w http.ResponseWriter, r *http.Request) {
	p := utils.ReadQueryParams(r)

	tickets, err := th.ticketStore.GetAllTickets(p)
	if err != nil {
		utils.ServerError(w, th.logger, "get all tickets", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "tickets fetched successfully", "tickets": tickets})
}

// Get Tickets By User ID Handler
func (th *TicketHandler) HandleGetTicketsByUserID(w http.ResponseWriter, r *http.Request) {
	userID, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, th.logger, "get tickets by user id", err)
		return
	}

	p := utils.ReadQueryParams(r)

	tickets, err := th.ticketStore.GetTicketsByUserID(userID, p)
	if err != nil {
		utils.ServerError(w, th.logger, "get tickets by user id", err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "tickets fetched successfully", "tickets": tickets})
}
