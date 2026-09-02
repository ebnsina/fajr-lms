package media_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/ebnsina/fajr-lms/internal/media"
)

func newObjectStore(t *testing.T) *media.ObjectStore {
	t.Helper()
	endpoint := os.Getenv("FAJR_S3_ENDPOINT")
	if endpoint == "" {
		t.Skip("FAJR_S3_ENDPOINT not set")
	}
	store, err := media.NewObjectStore(context.Background(), media.ObjectStoreConfig{
		Endpoint:  endpoint,
		Bucket:    "fajr-test",
		AccessKey: os.Getenv("FAJR_S3_ACCESS_KEY"),
		SecretKey: os.Getenv("FAJR_S3_SECRET_KEY"),
		Region:    "us-east-1",
		MaxBytes:  1 << 20,
	})
	if err != nil {
		t.Fatalf("new object store: %v", err)
	}
	return store
}

func TestObjectStoreRoundTrip(t *testing.T) {
	store := newObjectStore(t)
	ctx := context.Background()

	ingested, err := store.Ingest(ctx, "tenant-1", media.Source{
		Filename: "الدرس الأول.mp4", ContentType: "video/mp4", ByteSize: 11,
	})
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if ingested.State != media.StatePending || ingested.Upload == nil {
		t.Fatalf("got %+v, want pending with an upload target", ingested)
	}
	if !strings.HasPrefix(ingested.ExternalRef, "t/tenant-1/") {
		t.Errorf("key %q is not scoped to the tenant", ingested.ExternalRef)
	}

	asset := media.Asset{ExternalRef: ingested.ExternalRef, State: media.StatePending, ContentType: "video/mp4"}

	t.Run("playback before upload is refused", func(t *testing.T) {
		if _, err := store.Playback(ctx, asset, media.Viewer{}); !errors.Is(err, media.ErrNotReady) {
			t.Errorf("got %v, want ErrNotReady", err)
		}
	})

	t.Run("finalizing before upload reports the missing file", func(t *testing.T) {
		if _, err := store.Finalize(ctx, asset); !errors.Is(err, media.ErrMissingUpload) {
			t.Errorf("got %v, want ErrMissingUpload", err)
		}
	})

	body := "hello video"
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, ingested.Upload.URL, strings.NewReader(body))
	if err != nil {
		t.Fatalf("build upload request: %v", err)
	}
	req.Header.Set("Content-Type", "video/mp4")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		out, _ := io.ReadAll(resp.Body)
		t.Fatalf("upload: got %d: %s", resp.StatusCode, out)
	}

	result, err := store.Finalize(ctx, asset)
	if err != nil {
		t.Fatalf("finalize: %v", err)
	}
	if result.State != media.StateReady || result.Metadata["byte_size"] != int64(len(body)) {
		t.Fatalf("got %+v, want ready with %d bytes", result, len(body))
	}

	asset.State = media.StateReady
	playback, err := store.Playback(ctx, asset, media.Viewer{UserID: "u1"})
	if err != nil {
		t.Fatalf("playback: %v", err)
	}
	if playback.Kind != media.PlaybackFile || playback.ExpiresAt == nil {
		t.Fatalf("got %+v", playback)
	}

	got, err := http.Get(playback.URL)
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	defer got.Body.Close()
	content, err := io.ReadAll(got.Body)
	if err != nil {
		t.Fatalf("read download: %v", err)
	}
	if string(content) != body {
		t.Errorf("downloaded %q, want %q", content, body)
	}
}

func TestObjectStoreRejects(t *testing.T) {
	store := newObjectStore(t)
	ctx := context.Background()

	cases := []struct {
		name string
		src  media.Source
		want error
	}{
		{"no filename", media.Source{ContentType: "video/mp4"}, media.ErrUnsupportedSource},
		{"disallowed type", media.Source{Filename: "page.html", ContentType: "text/html"}, media.ErrUnsupportedSource},
		{"no content type", media.Source{Filename: "clip.mp4"}, media.ErrUnsupportedSource},
		{"too large", media.Source{Filename: "big.mp4", ContentType: "video/mp4", ByteSize: 1 << 30}, media.ErrTooLarge},
	}
	for _, c := range cases {
		if _, err := store.Ingest(ctx, "t", c.src); !errors.Is(err, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, err, c.want)
		}
	}

	// A traversing filename must not escape the tenant prefix.
	ingested, err := store.Ingest(ctx, "t", media.Source{
		Filename: "../../etc/passwd", ContentType: "application/pdf",
	})
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if strings.Contains(ingested.ExternalRef, "..") || !strings.HasPrefix(ingested.ExternalRef, "t/t/") {
		t.Errorf("key %q escaped the tenant prefix", ingested.ExternalRef)
	}
}
