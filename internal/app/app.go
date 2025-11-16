package app

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/lukasbarti/musixx/internal/config"
	"github.com/lukasbarti/musixx/internal/downloader"
	"github.com/lukasbarti/musixx/internal/server"
	"github.com/lukasbarti/musixx/internal/storage/sqlite"
)

// App wires together core services and orchestrates lifecycle management.
type App struct {
	store      *sqlite.Store
	httpServer *server.Server
}

// New constructs an App after configuration has been loaded.
func New() (*App, error) {
	store, err := sqlite.Open(config.DatabasePath())
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	downloaderSvc, err := downloader.NewYTDLP(config.YTDLPPath(), config.MusicDir())
	if err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("init downloader: %w", err)
	}

	srv, err := server.New(store.TrackRepository(), downloaderSvc)
	if err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("init server: %w", err)
	}

	return &App{store: store, httpServer: srv}, nil
}

// Run starts the Musixx application.
func (a *App) Run(ctx context.Context) error {
	log.Printf("starting Musixx server on port %d", config.HTTPPort())
	defer func() {
		if err := a.store.Close(); err != nil {
			log.Printf("error closing database: %v", err)
		}
	}()

	err := a.httpServer.Run(ctx)
	if errors.Is(err, context.Canceled) {
		log.Printf("musixx server shutdown complete")
		return context.Canceled
	}
	if err != nil {
		return fmt.Errorf("run http server: %w", err)
	}

	log.Printf("musixx server stopped")
	return nil
}
