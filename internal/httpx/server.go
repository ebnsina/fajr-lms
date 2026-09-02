package httpx

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

// Serve runs srv until ctx is cancelled, then drains connections.
func Serve(ctx context.Context, srv *http.Server, shutdownTimeout time.Duration) error {
	errCh := make(chan error, 1)
	go func() {
		slog.Info("http server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	slog.Info("shutting down", "timeout", shutdownTimeout)
	shutCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutCtx); err != nil {
		return errors.Join(err, srv.Close())
	}
	return <-errCh
}
