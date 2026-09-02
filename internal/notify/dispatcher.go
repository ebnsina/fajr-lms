package notify

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

// Delivery is one queued message the dispatcher must attempt.
type Delivery struct {
	ID          string
	Channel     string
	Destination string
	Body        string
	Attempts    int16
}

// Queue is the storage the dispatcher drains. The database implements it.
type Queue interface {
	Claim(ctx context.Context, limit int32) ([]Delivery, error)
	Settle(ctx context.Context, id, state, failure string, backoff time.Duration) error
}

const (
	maxAttempts = 5
	batchSize   = 25
)

// Dispatcher drains queued deliveries and hands each to its channel.
type Dispatcher struct {
	queue    Queue
	channels map[string]Channel
	Interval time.Duration
}

func NewDispatcher(queue Queue, channels ...Channel) *Dispatcher {
	d := &Dispatcher{queue: queue, channels: make(map[string]Channel, len(channels)), Interval: 10 * time.Second}
	for _, c := range channels {
		d.channels[c.Name()] = c
	}
	return d
}

// Run drains the queue until the context is cancelled.
func (d *Dispatcher) Run(ctx context.Context) {
	ticker := time.NewTicker(d.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("notification dispatcher stopped")
			return
		case <-ticker.C:
			if n, err := d.Drain(ctx); err != nil && !errors.Is(err, context.Canceled) {
				slog.Error("notification drain failed", "error", err)
			} else if n > 0 {
				slog.Debug("notifications dispatched", "count", n)
			}
		}
	}
}

// Drain attempts one batch and reports how many it handled.
func (d *Dispatcher) Drain(ctx context.Context) (int, error) {
	deliveries, err := d.queue.Claim(ctx, batchSize)
	if err != nil {
		return 0, err
	}

	for _, delivery := range deliveries {
		channel, ok := d.channels[delivery.Channel]
		if !ok {
			// Nothing will ever deliver this, so stop retrying it.
			d.settle(ctx, delivery, "skipped", "channel "+delivery.Channel+" is not configured", 0)
			continue
		}

		err := channel.Send(ctx, Message{To: delivery.Destination, Body: delivery.Body})
		switch {
		case err == nil:
			d.settle(ctx, delivery, "sent", "", 0)
		case delivery.Attempts >= maxAttempts:
			d.settle(ctx, delivery, "failed", err.Error(), 0)
		default:
			d.settle(ctx, delivery, "queued", err.Error(), backoff(delivery.Attempts))
		}
	}
	return len(deliveries), nil
}

func (d *Dispatcher) settle(ctx context.Context, delivery Delivery, state, failure string, wait time.Duration) {
	if err := d.queue.Settle(ctx, delivery.ID, state, failure, wait); err != nil {
		slog.ErrorContext(ctx, "could not record delivery outcome",
			"delivery", delivery.ID, "state", state, "error", err)
	}
}

// backoff doubles the wait per attempt, capped so a queue cannot stall for hours.
func backoff(attempts int16) time.Duration {
	wait := time.Minute << min(attempts, 5)
	if wait > 30*time.Minute {
		return 30 * time.Minute
	}
	return wait
}
