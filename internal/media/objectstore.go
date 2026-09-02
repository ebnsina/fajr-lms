package media

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// ObjectStoreConfig points the file provider at any S3-compatible bucket.
type ObjectStoreConfig struct {
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
	Region    string
	UseSSL    bool
	MaxBytes  int64
}

// ObjectStore stores uploads in an S3-compatible bucket and hands out expiring
// URLs. Bytes go client-to-bucket directly; the API never proxies them.
type ObjectStore struct {
	client   *minio.Client
	bucket   string
	maxBytes int64
}

const (
	defaultMaxUpload = 2 << 30 // 2 GiB
	uploadTTL        = 30 * time.Minute
	playbackTTL      = 6 * time.Hour
)

// allowedTypes is a fixed list, so a mislabelled upload cannot become an
// HTML page served from the media origin.
var allowedTypes = map[string]bool{
	"video/mp4": true, "video/webm": true, "video/quicktime": true,
	"audio/mpeg": true, "audio/mp4": true, "audio/ogg": true, "audio/wav": true,
	"application/pdf": true,
	"image/jpeg":      true, "image/png": true, "image/webp": true, "image/avif": true,
	"text/vtt": true, "text/plain": true,
}

func NewObjectStore(ctx context.Context, cfg ObjectStoreConfig) (*ObjectStore, error) {
	if cfg.Endpoint == "" || cfg.Bucket == "" {
		return nil, fmt.Errorf("media: object store needs an endpoint and a bucket")
	}
	if cfg.MaxBytes <= 0 {
		cfg.MaxBytes = defaultMaxUpload
	}

	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("media: connect to object store: %w", err)
	}

	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("media: reach bucket %q: %w", cfg.Bucket, err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{Region: cfg.Region}); err != nil {
			return nil, fmt.Errorf("media: create bucket %q: %w", cfg.Bucket, err)
		}
	}
	return &ObjectStore{client: client, bucket: cfg.Bucket, maxBytes: cfg.MaxBytes}, nil
}

func (o *ObjectStore) Caps() Caps {
	return Caps{Name: "file", AcceptsFile: true, Captions: true, Offline: true}
}

// Ingest reserves an object key and returns the URL to upload to.
func (o *ObjectStore) Ingest(ctx context.Context, tenantID string, src Source) (Ingested, error) {
	if strings.TrimSpace(src.Filename) == "" {
		return Ingested{}, fmt.Errorf("%w: a file needs a name", ErrUnsupportedSource)
	}
	contentType := strings.ToLower(strings.TrimSpace(src.ContentType))
	if !allowedTypes[contentType] {
		return Ingested{}, fmt.Errorf("%w: %s files are not accepted", ErrUnsupportedSource, contentType)
	}
	if src.ByteSize > o.maxBytes {
		return Ingested{}, fmt.Errorf("%w: limit is %d bytes", ErrTooLarge, o.maxBytes)
	}

	key := path.Join("t", tenantID, uuid.NewString(), safeName(src.Filename))
	signed, err := o.client.PresignedPutObject(ctx, o.bucket, key, uploadTTL)
	if err != nil {
		return Ingested{}, fmt.Errorf("media: sign upload url: %w", err)
	}

	return Ingested{
		ExternalRef: key,
		State:       StatePending,
		ContentType: contentType,
		Metadata:    map[string]any{"filename": safeName(src.Filename)},
		Upload: &Upload{
			URL: signed.String(), Method: "PUT",
			Headers:   map[string]string{"Content-Type": contentType},
			ExpiresAt: time.Now().Add(uploadTTL), MaxBytes: o.maxBytes,
		},
	}, nil
}

// Finalize confirms the object really landed and records its true size.
func (o *ObjectStore) Finalize(ctx context.Context, a Asset) (Ingested, error) {
	info, err := o.client.StatObject(ctx, o.bucket, a.ExternalRef, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).StatusCode == 404 {
			return Ingested{}, ErrMissingUpload
		}
		return Ingested{}, fmt.Errorf("media: stat uploaded object: %w", err)
	}
	if info.Size > o.maxBytes {
		return Ingested{}, fmt.Errorf("%w: limit is %d bytes", ErrTooLarge, o.maxBytes)
	}
	return Ingested{
		ExternalRef: a.ExternalRef,
		State:       StateReady,
		ContentType: a.ContentType,
		Metadata:    map[string]any{"byte_size": info.Size, "etag": info.ETag},
	}, nil
}

// Playback hands back a short-lived URL scoped to one viewer.
func (o *ObjectStore) Playback(ctx context.Context, a Asset, v Viewer) (Playback, error) {
	if a.State != StateReady {
		return Playback{}, ErrNotReady
	}

	// Attribution only; the signature is what actually grants access.
	params := url.Values{}
	if v.UserID != "" {
		params.Set("x-fajr-viewer", v.UserID)
	}

	signed, err := o.client.PresignedGetObject(ctx, o.bucket, a.ExternalRef, playbackTTL, params)
	if err != nil {
		return Playback{}, fmt.Errorf("media: sign playback url: %w", err)
	}
	expires := time.Now().Add(playbackTTL)
	return Playback{Kind: PlaybackFile, URL: signed.String(), ExpiresAt: &expires}, nil
}

// safeName keeps a recognizable filename without letting it escape the key.
func safeName(name string) string {
	name = path.Base(strings.ReplaceAll(name, "\\", "/"))
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-.")
	if out == "" || out == ".." {
		return "file"
	}
	if len(out) > 120 {
		out = out[len(out)-120:]
	}
	return out
}
