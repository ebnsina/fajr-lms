package notify

import (
	"context"
	"log/slog"
	"strings"
)

// Recipient is who a notification is for and how they can be reached.
type Recipient struct {
	UserID string
	Phone  string
	Email  string
}

// Event is one thing worth telling somebody about.
type Event struct {
	TenantID  string
	Recipient Recipient
	Kind      string
	Title     string
	Body      string
	Data      map[string]any
}

// Sink records notifications and queues their deliveries. The API implements it.
type Sink interface {
	Record(ctx context.Context, e Event, channels []Target) error
}

// Target is one queued delivery: which channel, and to what address.
type Target struct {
	Channel     string
	Destination string
}

// Service turns an event into an inbox entry plus outbound deliveries.
type Service struct {
	sink Sink
	// Order to try for a recipient. Email is last because in these markets it
	// is the least likely to be read.
	Channels []string
}

func NewService(sink Sink, channels ...string) *Service {
	if len(channels) == 0 {
		channels = []string{"sms"}
	}
	return &Service{sink: sink, Channels: channels}
}

// Notify never fails the caller's work: a message that cannot be sent is
// logged, because losing a grade is worse than losing its announcement.
func (s *Service) Notify(ctx context.Context, e Event) {
	if s == nil || s.sink == nil {
		return
	}
	if err := s.sink.Record(ctx, e, s.targets(e.Recipient)); err != nil {
		slog.ErrorContext(ctx, "could not record notification",
			"kind", e.Kind, "user", e.Recipient.UserID, "error", err)
	}
}

// targets picks an address per configured channel, skipping any it cannot reach.
func (s *Service) targets(r Recipient) []Target {
	out := make([]Target, 0, len(s.Channels))
	for _, channel := range s.Channels {
		destination := r.Phone
		if channel == "email" {
			destination = r.Email
		}
		if strings.TrimSpace(destination) == "" {
			continue
		}
		out = append(out, Target{Channel: channel, Destination: destination})
	}
	return out
}
