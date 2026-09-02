// Package notify delivers short messages over a pluggable channel.
package notify

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// Message is one outbound notification. Body is already rendered.
type Message struct {
	To      string
	Purpose string
	Body    string
}

// Channel is the seam an SMS, WhatsApp or email provider plugs into.
type Channel interface {
	Send(ctx context.Context, m Message) error
	Name() string
}

// LogChannel writes messages to the log instead of sending them.
type LogChannel struct{}

func (LogChannel) Name() string { return "log" }

func (LogChannel) Send(ctx context.Context, m Message) error {
	if strings.TrimSpace(m.To) == "" {
		return fmt.Errorf("notify: message has no destination")
	}
	slog.InfoContext(ctx, "notification (not sent)", "channel", "log", "to", m.To, "purpose", m.Purpose, "body", m.Body)
	return nil
}
