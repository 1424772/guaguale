package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	_ "time/tzdata"

	rediscache "github.com/1424772/guaguale/apps/api/internal/cache/redis"
	"github.com/1424772/guaguale/apps/api/internal/httpapi"
	"github.com/1424772/guaguale/apps/api/internal/service"
	"github.com/1424772/guaguale/apps/api/internal/store"
	"github.com/1424772/guaguale/apps/api/internal/store/memory"
	mysqlstore "github.com/1424772/guaguale/apps/api/internal/store/mysql"
	"github.com/1424772/guaguale/apps/api/migrations"
)

const defaultPort = "8080"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		if err := runHealthcheck(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	applicationStore, closeStore, err := initializeStore(logger)
	if err != nil {
		logger.Error("initialize store", "error", err)
		os.Exit(1)
	}
	defer closeStore()

	leaderboardCache, closeCache := initializeLeaderboardCache(logger)
	defer closeCache()
	serviceLayer := service.NewWithLeaderboardCache(applicationStore, leaderboardCache)
	adminConfig, err := loadAdminConfig()
	if err != nil {
		logger.Error("load admin configuration", "error", err)
		os.Exit(1)
	}
	handler := httpapi.NewWithAdmin(serviceLayer, applicationStore, logger, envBool("COOKIE_SECURE", false), adminConfig)
	server := &http.Server{
		Addr:              ":" + envString("API_PORT", defaultPort),
		Handler:           httpapi.RequestLogger(logger, handler),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	shutdownSignal, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go cleanExpiredSessions(shutdownSignal, logger, applicationStore)
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

func loadAdminConfig() (httpapi.AdminConfig, error) {
	username := os.Getenv("ADMIN_USERNAME")
	passwordHashEncoded := os.Getenv("ADMIN_PASSWORD_HASH_B64")
	sessionSecretEncoded := os.Getenv("ADMIN_SESSION_SECRET_B64")
	if username == "" && passwordHashEncoded == "" && sessionSecretEncoded == "" {
		return httpapi.AdminConfig{}, nil
	}
	if username == "" || passwordHashEncoded == "" || sessionSecretEncoded == "" {
		return httpapi.AdminConfig{}, errors.New("ADMIN_USERNAME, ADMIN_PASSWORD_HASH_B64 and ADMIN_SESSION_SECRET_B64 must be set together")
	}
	passwordHash, err := base64.StdEncoding.DecodeString(passwordHashEncoded)
	if err != nil {
		return httpapi.AdminConfig{}, fmt.Errorf("decode ADMIN_PASSWORD_HASH_B64: %w", err)
	}
	sessionSecret, err := base64.StdEncoding.DecodeString(sessionSecretEncoded)
	if err != nil {
		return httpapi.AdminConfig{}, fmt.Errorf("decode ADMIN_SESSION_SECRET_B64: %w", err)
	}
	if len(sessionSecret) < 32 {
		return httpapi.AdminConfig{}, errors.New("ADMIN_SESSION_SECRET_B64 must decode to at least 32 bytes")
	}
	return httpapi.AdminConfig{Username: username, PasswordHash: passwordHash, SessionSecret: sessionSecret}, nil
}

func initializeLeaderboardCache(logger *slog.Logger) (service.LeaderboardCache, func()) {
	address := os.Getenv("REDIS_ADDR")
	if address == "" {
		return nil, func() {}
	}
	cache := rediscache.New(address, os.Getenv("REDIS_PASSWORD"))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := cache.Ping(ctx); err != nil {
		logger.Warn("redis leaderboard cache unavailable; falling back to mysql", "error", err)
		_ = cache.Close()
		return nil, func() {}
	}
	logger.Info("redis leaderboard cache enabled", "address", address)
	return cache, func() { _ = cache.Close() }
}

func initializeStore(logger *slog.Logger) (store.Store, func(), error) {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		if os.Getenv("APP_ENV") == "production" {
			return nil, func() {}, errors.New("MYSQL_DSN is required in production")
		}
		logger.Warn("MYSQL_DSN is empty; using volatile in-memory store")
		return memory.New(), func() {}, nil
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, func() {}, err
	}
	db.SetMaxOpenConns(envInt("MYSQL_MAX_OPEN_CONNS", 40))
	db.SetMaxIdleConns(envInt("MYSQL_MAX_IDLE_CONNS", 10))
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, func() {}, err
	}
	if err := migrations.Apply(ctx, db); err != nil {
		db.Close()
		return nil, func() {}, err
	}
	return mysqlstore.New(db), func() { _ = db.Close() }, nil
}

func cleanExpiredSessions(ctx context.Context, logger *slog.Logger, applicationStore store.Store) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			cleanupContext, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := applicationStore.DeleteExpiredSessions(cleanupContext, now.UTC())
			cancel()
			if err != nil && !errors.Is(err, context.Canceled) {
				logger.Error("delete expired sessions", "error", err)
			}
		}
	}
}

func runHealthcheck() error {
	port := envString("API_PORT", defaultPort)
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

func envString(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func envBool(name string, fallback bool) bool {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envInt(name string, fallback int) int {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
