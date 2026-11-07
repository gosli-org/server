package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gosli-org/server/internal/config"
	"github.com/gosli-org/server/internal/connection"
	"github.com/gosli-org/server/internal/dispatcher"
	"github.com/gosli-org/server/internal/terminal"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Parse(os.Args[1:])
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", cfg.Listen)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	terminals := terminal.New(cfg.ProjectDir)
	defer terminals.Close()
	mux := http.NewServeMux()
	mux.Handle("/ws", connection.New(ctx, dispatcher.New(terminals)))
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()
	fmt.Printf("listening ws://%s/ws\n", listener.Addr())
	select {
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	case err := <-served:
		if err != http.ErrServerClosed {
			return err
		}
		return nil
	}
}
