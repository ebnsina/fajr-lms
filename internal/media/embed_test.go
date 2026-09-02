package media_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ebnsina/fajr-lms/internal/media"
)

func TestEmbedIngest(t *testing.T) {
	p := media.Embed{AllowedHosts: []string{"video.madrasa.edu.bd"}}
	ctx := context.Background()

	accepted := map[string]string{
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ": "https://www.youtube-nocookie.com/embed/dQw4w9WgXcQ",
		"https://youtu.be/dQw4w9WgXcQ":                "https://www.youtube-nocookie.com/embed/dQw4w9WgXcQ",
		"https://www.youtube.com/embed/dQw4w9WgXcQ":   "https://www.youtube-nocookie.com/embed/dQw4w9WgXcQ",
		"https://vimeo.com/123456789":                 "https://player.vimeo.com/video/123456789",
		"https://player.vimeo.com/video/123456789":    "https://player.vimeo.com/video/123456789",
		"https://video.madrasa.edu.bd/w/abc":          "https://video.madrasa.edu.bd/w/abc",
	}
	for in, want := range accepted {
		got, err := p.Ingest(ctx, "t", media.Source{URL: in})
		if err != nil {
			t.Errorf("Ingest(%q): %v", in, err)
			continue
		}
		if got.ExternalRef != want {
			t.Errorf("Ingest(%q) = %q, want %q", in, got.ExternalRef, want)
		}
		if got.State != media.StateReady {
			t.Errorf("Ingest(%q) state = %q, want ready", in, got.State)
		}
	}

	rejected := []string{
		"", "not a url", "http://youtube.com/watch?v=dQw4w9WgXcQ",
		"https://youtube.com/watch?v=short", "https://vimeo.com/abc",
		"https://evil.example.com/video", "javascript:alert(1)",
	}
	for _, in := range rejected {
		if _, err := p.Ingest(ctx, "t", media.Source{URL: in}); !errors.Is(err, media.ErrUnsupportedSource) {
			t.Errorf("Ingest(%q) = %v, want ErrUnsupportedSource", in, err)
		}
	}
}

func TestEmbedPlayback(t *testing.T) {
	p := media.Embed{}
	ctx := context.Background()

	ready := media.Asset{State: media.StateReady, ExternalRef: "https://player.vimeo.com/video/1"}
	got, err := p.Playback(ctx, ready, media.Viewer{})
	if err != nil {
		t.Fatalf("Playback: %v", err)
	}
	if got.Kind != media.PlaybackEmbed || got.URL != ready.ExternalRef {
		t.Errorf("got %+v", got)
	}

	if _, err := p.Playback(ctx, media.Asset{State: media.StatePending}, media.Viewer{}); !errors.Is(err, media.ErrNotReady) {
		t.Errorf("pending asset: got %v, want ErrNotReady", err)
	}
}

func TestRegistry(t *testing.T) {
	r, err := media.NewRegistry("embed", media.Embed{})
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	if _, err := r.Get("embed"); err != nil {
		t.Errorf("Get(embed): %v", err)
	}
	if _, err := r.Get("transcoder"); !errors.Is(err, media.ErrUnknownProvider) {
		t.Errorf("Get(transcoder) = %v, want ErrUnknownProvider", err)
	}
	if _, err := r.For(media.Source{URL: "https://youtu.be/x"}); err != nil {
		t.Errorf("For(url): %v", err)
	}
	if _, err := r.For(media.Source{Filename: "lesson.mp4"}); !errors.Is(err, media.ErrUnsupportedSource) {
		t.Errorf("For(file) = %v, want ErrUnsupportedSource until a file provider is registered", err)
	}
	if _, err := media.NewRegistry("nope", media.Embed{}); err == nil {
		t.Error("NewRegistry with an unregistered default should fail")
	}
}
