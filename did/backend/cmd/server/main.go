package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"didcontract/backend/internal/chain"
	"didcontract/backend/internal/config"
	"didcontract/backend/internal/httpapi"
	"didcontract/backend/internal/service"
	"didcontract/backend/internal/store"
)

func main() {
	configPath := flag.String("config", "./configs/backend.yml", "backend YAML configuration path")
	flag.Parse()
	if err := run(*configPath); err != nil {
		log.Fatal(err)
	}
}

func run(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	repository, err := store.Open(cfg.Storage.StatePath, cfg.Storage.AuditPath)
	if err != nil {
		return err
	}
	pool, err := chain.NewPool(cfg)
	if err != nil {
		return err
	}
	defer pool.Close()
	application := service.New(pool, repository)
	server := &http.Server{
		Addr: cfg.Server.Address, Handler: httpapi.New(cfg, application).Router(),
		ReadTimeout: cfg.ReadTimeout(), WriteTimeout: cfg.WriteTimeout(),
	}
	errorsChannel := make(chan error, 1)
	go func() {
		log.Printf("Gov-DID backend listening on %s", cfg.Server.Address)
		errorsChannel <- server.ListenAndServe()
	}()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	select {
	case signalValue := <-signals:
		log.Printf("received %s, shutting down", signalValue)
	case serveErr := <-errorsChannel:
		if !errors.Is(serveErr, http.ErrServerClosed) {
			return fmt.Errorf("serve backend: %w", serveErr)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout())
	defer cancel()
	return server.Shutdown(ctx)
}
