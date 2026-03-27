package handlers

import (
	"log/slog"
	"net/http"

	"github.com/levionstudio/fintech/internal/store"
	"github.com/levionstudio/fintech/internal/utils"
)

type LoginActivityHandler struct {
	loginActivityStore store.LoginActivityStore
	logger             *slog.Logger
}

func NewLoginActivityHandler(loginActivityStore store.LoginActivityStore, logger *slog.Logger) *LoginActivityHandler {
	return &LoginActivityHandler{
		loginActivityStore: loginActivityStore,
		logger:             logger,
	}
}

func (lh *LoginActivityHandler) HandleGetAllLoginActivities(w http.ResponseWriter, r *http.Request) {
	p := utils.ReadQueryParams(r)
	results, err := lh.loginActivityStore.GetAllLoginActivities(p)
	if err != nil {
		utils.ServerError(w, lh.logger, "get all login activities", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"message":          "login activities fetched",
		"login_activities": results,
	})
}

func (lh *LoginActivityHandler) HandleGetLoginActivitiesByUserID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadParamID(r)
	if err != nil {
		utils.BadRequest(w, lh.logger, "get login activities by user", err)
		return
	}
	p := utils.ReadQueryParams(r)
	results, err := lh.loginActivityStore.GetLoginActivitiesByUserID(id, p)
	if err != nil {
		utils.ServerError(w, lh.logger, "get login activities by user", err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"message":          "login activities fetched",
		"login_activities": results,
	})
}
