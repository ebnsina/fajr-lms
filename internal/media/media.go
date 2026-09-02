// Package media keeps video and file handling behind one interface, so an
// embed, an object store or a private transcoder are all interchangeable.
package media

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type State string

const (
	StatePending    State = "pending"
	StateProcessing State = "processing"
	StateReady      State = "ready"
	StateFailed     State = "failed"
)

type PlaybackKind string

const (
	PlaybackEmbed    PlaybackKind = "embed"
	PlaybackHLS      PlaybackKind = "hls"
	PlaybackFile     PlaybackKind = "file"
	PlaybackUpload   PlaybackKind = "upload"
	PlaybackNotReady PlaybackKind = "not_ready"
)

var (
	ErrUnsupportedSource = errors.New("media: this provider cannot ingest that source")
	ErrNotReady          = errors.New("media: asset is not ready for playback")
	ErrUnknownProvider   = errors.New("media: unknown provider")
)

// Source is what the author supplied: a pasted URL or an uploaded file.
type Source struct {
	URL         string
	Filename    string
	ContentType string
	ByteSize    int64
}

// Asset is the stored record a provider acts on.
type Asset struct {
	ID          string
	TenantID    string
	ExternalRef string
	State       State
	ContentType string
	DurationS   int32
}

// Ingested is what a provider returns after accepting a source.
type Ingested struct {
	ExternalRef string
	State       State
	ContentType string
	DurationS   int32
	Metadata    map[string]any
}

// Viewer identifies who is watching, so a URL can be scoped and expiring.
type Viewer struct {
	UserID   string
	TenantID string
}

// Playback is the answer the player needs and nothing more.
type Playback struct {
	Kind      PlaybackKind   `json:"kind"`
	URL       string         `json:"url"`
	ExpiresAt *time.Time     `json:"expires_at,omitempty"`
	Headers   map[string]any `json:"headers,omitempty"`
}

// Caps tells the UI what this provider supports, so it can adapt.
type Caps struct {
	Name        string `json:"name"`
	AcceptsURL  bool   `json:"accepts_url"`
	AcceptsFile bool   `json:"accepts_file"`
	Renditions  bool   `json:"renditions"`
	Captions    bool   `json:"captions"`
	Offline     bool   `json:"offline"`
}

// Provider is the seam. Embed ships today, a transcoder drops in unchanged.
type Provider interface {
	Ingest(ctx context.Context, tenantID string, src Source) (Ingested, error)
	Playback(ctx context.Context, a Asset, v Viewer) (Playback, error)
	Caps() Caps
}

// Registry resolves a provider by name and names the default for new assets.
type Registry struct {
	providers map[string]Provider
	fallback  string
}

func NewRegistry(fallback string, providers ...Provider) (*Registry, error) {
	r := &Registry{providers: make(map[string]Provider, len(providers)), fallback: fallback}
	for _, p := range providers {
		r.providers[p.Caps().Name] = p
	}
	if _, ok := r.providers[fallback]; !ok {
		return nil, fmt.Errorf("media: default provider %q is not registered", fallback)
	}
	return r, nil
}

func (r *Registry) Get(name string) (Provider, error) {
	p, ok := r.providers[name]
	if !ok {
		return nil, ErrUnknownProvider
	}
	return p, nil
}

// For picks the provider that can handle src, preferring the default.
func (r *Registry) For(src Source) (Provider, error) {
	wantURL := src.URL != ""
	if p := r.providers[r.fallback]; accepts(p, wantURL) {
		return p, nil
	}
	for _, p := range r.providers {
		if accepts(p, wantURL) {
			return p, nil
		}
	}
	return nil, ErrUnsupportedSource
}

func (r *Registry) Capabilities() []Caps {
	out := make([]Caps, 0, len(r.providers))
	for _, p := range r.providers {
		out = append(out, p.Caps())
	}
	return out
}

func accepts(p Provider, wantURL bool) bool {
	if p == nil {
		return false
	}
	if wantURL {
		return p.Caps().AcceptsURL
	}
	return p.Caps().AcceptsFile
}
