package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/httpx"
	"github.com/ebnsina/fajr-lms/internal/media"
)

func (s *Server) mediaProviders(w http.ResponseWriter, r *http.Request) error {
	return httpx.JSON(w, http.StatusOK, map[string]any{"providers": s.media.Capabilities()})
}

type ingestRequest struct {
	Provider string `json:"provider"`
	URL      string `json:"url"`
	Title    string `json:"title"`
	Kind     string `json:"kind"`
}

// ingestMedia records an asset for whichever provider can handle the source.
func (s *Server) ingestMedia(w http.ResponseWriter, r *http.Request) error {
	var body ingestRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}

	kind, err := parseLessonKind(body.Kind)
	if err != nil {
		return err
	}
	if body.Kind == "" {
		kind = database.LessonKindVideo
	}

	src := media.Source{URL: strings.TrimSpace(body.URL)}
	provider, err := s.pickProvider(body.Provider, src)
	if err != nil {
		return err
	}

	tenant := CurrentTenant(r.Context())
	ingested, err := provider.Ingest(r.Context(), tenant.ID.String(), src)
	if errors.Is(err, media.ErrUnsupportedSource) {
		return &httpx.Error{Status: http.StatusUnprocessableEntity, Code: "unsupported_source",
			Message: err.Error(), Field: "url"}
	}
	if err != nil {
		return err
	}

	metadata, err := json.Marshal(ingested.Metadata)
	if err != nil {
		return err
	}

	var asset database.MediaAsset
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		var err error
		asset, err = q.CreateMediaAsset(r.Context(), database.CreateMediaAssetParams{
			TenantID: tenant.ID, Provider: provider.Caps().Name, ExternalRef: ingested.ExternalRef,
			State: database.MediaState(ingested.State), Kind: kind,
			Title: strings.TrimSpace(body.Title), DurationS: ingested.DurationS,
			ContentType: ingested.ContentType, Metadata: metadata,
			CreatedBy: uuid.NullUUID{UUID: Authenticated(r.Context()).UserID, Valid: true},
		})
		return err
	})
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, asset)
}

// pickProvider honours an explicit choice, otherwise asks the registry.
func (s *Server) pickProvider(name string, src media.Source) (media.Provider, error) {
	if name = strings.TrimSpace(name); name != "" {
		provider, err := s.media.Get(name)
		if errors.Is(err, media.ErrUnknownProvider) {
			return nil, invalid("provider", "No media provider by that name is configured.")
		}
		return provider, err
	}

	provider, err := s.media.For(src)
	if errors.Is(err, media.ErrUnsupportedSource) {
		return nil, &httpx.Error{Status: http.StatusUnprocessableEntity, Code: "unsupported_source",
			Message: "No configured provider can handle that source.", Field: "url"}
	}
	return provider, err
}

func (s *Server) mediaPlayback(w http.ResponseWriter, r *http.Request) error {
	id, err := pathUUID(r, "id")
	if err != nil {
		return err
	}

	tenant := CurrentTenant(r.Context())
	var asset database.MediaAsset
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		var err error
		asset, err = q.GetMediaAsset(r.Context(), id)
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}

	provider, err := s.media.Get(asset.Provider)
	if errors.Is(err, media.ErrUnknownProvider) {
		return httpx.Errorf(http.StatusNotImplemented, "provider_unavailable",
			"The provider that holds this media is not configured on this server.")
	}
	if err != nil {
		return err
	}

	playback, err := provider.Playback(r.Context(), media.Asset{
		ID: asset.ID.String(), TenantID: asset.TenantID.String(), ExternalRef: asset.ExternalRef,
		State: media.State(asset.State), ContentType: asset.ContentType, DurationS: asset.DurationS,
	}, media.Viewer{UserID: Authenticated(r.Context()).UserID.String(), TenantID: tenant.ID.String()})

	if errors.Is(err, media.ErrNotReady) {
		return httpx.JSON(w, http.StatusOK, media.Playback{Kind: media.PlaybackNotReady})
	}
	if err != nil {
		return err
	}

	// One embed hand-out is one delivery; a file provider adds real byte counts.
	if err := s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		return q.RecordMediaDelivery(r.Context(), database.RecordMediaDeliveryParams{TenantID: tenant.ID, Bytes: 0})
	}); err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, playback)
}

type attachMediaRequest struct {
	MediaID string `json:"media_id"`
}

func (s *Server) attachMedia(w http.ResponseWriter, r *http.Request) error {
	lessonID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body attachMediaRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}

	mediaID := uuid.NullUUID{}
	if raw := strings.TrimSpace(body.MediaID); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			return invalid("media_id", "Provide a media id, or null to detach.")
		}
		mediaID = uuid.NullUUID{UUID: parsed, Valid: true}
	}

	var lesson database.Lesson
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		lesson, err = q.AttachMediaToLesson(r.Context(), database.AttachMediaToLessonParams{
			ID: lessonID, MediaID: mediaID,
		})
		return err
	})
	if database.IsNotFound(err) || isForeignKeyViolation(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, lesson)
}

func (s *Server) mediaUsage(w http.ResponseWriter, r *http.Request) error {
	var rows []database.MediaDelivery
	err := s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		rows, err = q.MediaUsage(r.Context(), 30)
		return err
	})
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"days": rows})
}
