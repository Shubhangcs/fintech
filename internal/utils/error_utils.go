package utils

import (
	"log/slog"
	"net/http"
)

func BadRequest(w http.ResponseWriter, logger *slog.Logger, message string, err error) {
	logger.Error(message, "error", err)
	WriteJSON(w, http.StatusBadRequest, Envelope{"error": err.Error()})
}

func ServerError(w http.ResponseWriter, logger *slog.Logger, message string, err error) {
	logger.Error(message, "error", err)
	WriteJSON(w, http.StatusInternalServerError, Envelope{"error": "internal server error"})
}
