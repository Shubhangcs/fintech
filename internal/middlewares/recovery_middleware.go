package middlewares

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/levionstudio/fintech/internal/utils"
)

// RecoveryMiddleware catches any panics and returns a 500 response instead of crashing the server
func RecoveryMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("PANIC", "error", err, "stack", string(debug.Stack()))
					utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"error": "internal server error"})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
