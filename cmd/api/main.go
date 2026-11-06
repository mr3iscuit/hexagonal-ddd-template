// Command api is the composition root: the only place that knows which adapters
// are bound to which ports.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/adapters/inbound/httpapi"
	"github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/adapters/inbound/ticker"
	"github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/adapters/outbound/memory"
	"github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/adapters/outbound/system"
	"github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/application"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error(logServerStopped, logKeyError, err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	noteUseCases := application.NewNoteApplicationService(memory.NewInMemoryNoteRepository(), system.SystemClock{}, system.RandomIDGenerator{})

	listener, err := net.Listen(listenNetwork, address())
	if err != nil {
		return err
	}
	server := &http.Server{
		Handler:           httpapi.NewRouter(noteUseCases, logger),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	reminderTicks := time.NewTicker(reminderCheckInterval)
	defer reminderTicks.Stop()
	go ticker.NewTickHandler(noteUseCases, logger).Run(ctx, reminderTicks.C)

	serveError := make(chan error, 1)
	go func() {
		logger.Info(logListening, logKeyAddress, listener.Addr().String())
		serveError <- server.Serve(listener)
	}()

	select {
	case err := <-serveError:
		return err
	case <-ctx.Done():
		// A second signal now ends the process, instead of waiting for shutdown.
		stop()
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		return err
	}
	if err := <-serveError; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func address() string {
	if value := os.Getenv(addressEnvironmentVariable); value != "" {
		return value
	}
	return defaultAddress
}
