package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const defaultPort = "8080"

type healthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	Time    string `json:"time"`
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		if err := runHealthcheck(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	port := os.Getenv("API_PORT")
	if port == "" {
		port = defaultPort
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthHandler)
	mux.HandleFunc("GET /api/v1/health", healthHandler)

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           requestLogger(logger, mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	shutdownSignal, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("api server started", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("api server stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}()

	<-shutdownSignal.Done()
	logger.Info("api server shutting down")

	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
}

func healthHandler(response http.ResponseWriter, _ *http.Request) {
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(response).Encode(healthResponse{
		Status:  "ok",
		Service: "guaguale-api",
		Time:    time.Now().UTC().Format(time.RFC3339),
	})
}

func requestLogger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		startedAt := time.Now()
		next.ServeHTTP(response, request)
		logger.Info("request completed",
			"method", request.Method,
			"path", request.URL.Path,
			"duration_ms", time.Since(startedAt).Milliseconds(),
		)
	})
}

func runHealthcheck() error {
	port := os.Getenv("API_PORT")
	if port == "" {
		port = defaultPort
	}

	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Get("http://127.0.0.1:" + port + "/healthz")
	if err != nil {
		return fmt.Errorf("healthcheck request failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck returned status %d", response.StatusCode)
	}
	return nil
}
