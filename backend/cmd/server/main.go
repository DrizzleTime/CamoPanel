package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"camopanel/backend/internal/config"
	"camopanel/backend/internal/docker"
	"camopanel/backend/internal/httpapi"
	"github.com/gin-gonic/gin"
)

func main() {
	for _, path := range []string{".env", "../.env"} {
		if err := config.LoadEnv(path); err != nil {
			slog.Error("load env", "error", err)
			os.Exit(1)
		}
	}
	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration", "error", err)
		os.Exit(1)
	}
	service, err := docker.NewEngine(cfg.DockerHost)
	if err != nil {
		slog.Error("docker configuration", "error", err)
		os.Exit(1)
	}
	gin.SetMode(gin.ReleaseMode)
	server := &http.Server{Addr: cfg.Address, Handler: httpapi.New(cfg, service), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Minute, WriteTimeout: 10 * time.Minute, IdleTimeout: 60 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	serveError := make(chan error, 1)
	go func() {
		slog.Info("CamoPanel started", "address", cfg.Address, "docker_host", cfg.DockerHost)
		if err := httpapi.Serve(server, cfg.TLSCert, cfg.TLSKey); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveError <- err
		}
	}()
	select {
	case err := <-serveError:
		slog.Error("server", "error", err)
		os.Exit(1)
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdown); err != nil {
		slog.Error("shutdown", "error", err)
	}
}
