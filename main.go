package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/levionstudio/fintech/internal/app"
	"github.com/levionstudio/fintech/internal/routes"
)

func main() {
	app, err := app.NewApplication()
	if err != nil {
		log.Fatalf("failed to initilize application: %v\n", err)
	}
	defer app.DB.Close()

	port := os.Getenv("SERVER_PORT")

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      routes.SetupRoutes(app),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	serverErr := make(chan error, 1)

	go func() {
		app.Logger.Info("server listening", "port", port)
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err = <-serverErr:
		if err != nil && err != http.ErrServerClosed {
			app.Logger.Error("server")
			os.Exit(0)
		}
	case sig := <-quit:
		app.Logger.Info("server shutting down", "signal", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
		defer cancel()

		if err = server.Shutdown(ctx); err != nil {
			app.Logger.Error("graceful shutdown failed", "error", err)
		} else {
			app.Logger.Info("server stopped cleanly")
		}
	}
}
