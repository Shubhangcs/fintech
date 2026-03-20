package models

import (
	"errors"
	"time"
)

type TicketModel struct {
	AdminID              string    `json:"admin_id"`
	UserID               string    `json:"user_id"`
	TicketID             int64     `json:"ticket_id,omitempty"`
	TicketTitle          string    `json:"ticket_title"`
	TicketDescription    string    `json:"ticket_description"`
	IsTicketCleared      bool      `json:"is_ticket_cleared"`
	UserName             string    `json:"user_name,omitempty"`
	UserBusinessName     *string   `json:"user_business_name,omitempty"`
	CreatedAT            time.Time `json:"created_at"`
	UpdatedAT            time.Time `json:"updated_at"`
}

func (t *TicketModel) Validate() error {
	if t.UserID == "" {
		return errors.New("user_id is required")
	}
	if t.TicketTitle == "" {
		return errors.New("ticket_title is required")
	}
	if t.TicketDescription == "" {
		return errors.New("ticket_description is required")
	}
	return nil
}
