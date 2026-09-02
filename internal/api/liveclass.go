package api

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/httpx"
)

// meetingHosts are the platforms a link may point at. Anything else is refused,
// so a pasted link cannot send a class somewhere unexpected.
var meetingHosts = map[string]string{
	"meet.google.com":     "google_meet",
	"zoom.us":             "zoom",
	"teams.live.com":      "teams",
	"teams.microsoft.com": "teams",
	"meet.jit.si":         "jitsi",
	"whereby.com":         "whereby",
}

type liveLinkRequest struct {
	JoinURL string `json:"join_url"`
	HostURL string `json:"host_url"`
}

// setSessionLink attaches a meeting link to a class. Meetings are created in
// Google Meet or Zoom by hand for now; a provider that mints its own rooms
// replaces this endpoint's body and nothing else.
func (s *Server) setSessionLink(w http.ResponseWriter, r *http.Request) error {
	sessionID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body liveLinkRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}

	joinURL, provider, err := meetingLink(body.JoinURL, "join_url")
	if err != nil {
		return err
	}
	hostURL := ""
	if strings.TrimSpace(body.HostURL) != "" {
		if hostURL, _, err = meetingLink(body.HostURL, "host_url"); err != nil {
			return err
		}
	}

	var session database.ClassSession
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		session, err = q.SetSessionLink(r.Context(), database.SetSessionLinkParams{
			ID: sessionID, Provider: provider, JoinUrl: joinURL, HostUrl: hostURL,
		})
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, session)
}

// meetingLink validates a pasted URL and names the platform it belongs to.
func meetingLink(raw, field string) (string, string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", "", invalid(field, "Paste the meeting link.")
	}
	u, err := url.Parse(trimmed)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return "", "", invalid(field, "Use an https meeting link.")
	}

	host := strings.ToLower(u.Hostname())
	for suffix, provider := range meetingHosts {
		if host == suffix || strings.HasSuffix(host, "."+suffix) {
			return u.String(), provider, nil
		}
	}
	return "", "", invalid(field, "Links from that platform are not accepted.")
}

type joinResponse struct {
	SessionID uuid.UUID  `json:"session_id"`
	Title     string     `json:"title"`
	Provider  string     `json:"provider"`
	JoinURL   string     `json:"join_url"`
	StartsAt  time.Time  `json:"starts_at"`
	OpensAt   time.Time  `json:"opens_at"`
	Recording *uuid.UUID `json:"recording_media_id,omitempty"`
}

const (
	joinOpensBefore = 15 * time.Minute
	joinClosesAfter = 4 * time.Hour
)

// joinSession hands the link only to an enrolled learner, and only around the
// scheduled time, so a link cannot be collected and shared long in advance.
func (s *Server) joinSession(w http.ResponseWriter, r *http.Request) error {
	sessionID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}

	userID := Authenticated(r.Context()).UserID
	staff := staffRole(CurrentRole(r.Context()))
	var out joinResponse

	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		session, err := q.GetClassSession(r.Context(), sessionID)
		if err != nil {
			return err
		}
		if !staff {
			enrollment, err := q.GetEnrollment(r.Context(), database.GetEnrollmentParams{
				CourseID: session.CourseID, UserID: userID,
			})
			if err != nil {
				return err
			}
			if enrollment.Status == database.EnrollmentStatusCancelled {
				return httpx.ErrForbidden
			}
		}
		if session.JoinUrl == "" {
			return httpx.Errorf(http.StatusConflict, "no_link_yet",
				"The teacher has not shared a meeting link for this class.")
		}

		opens := session.StartsAt.Time.Add(-joinOpensBefore)
		closes := session.StartsAt.Time.Add(joinClosesAfter)
		if session.EndsAt.Valid {
			closes = session.EndsAt.Time.Add(joinClosesAfter)
		}
		if !staff && (time.Now().Before(opens) || time.Now().After(closes)) {
			return httpx.Errorf(http.StatusConflict, "not_open",
				"This class is not open to join right now.")
		}

		link := session.JoinUrl
		if staff && session.HostUrl != "" {
			link = session.HostUrl
		}
		out = joinResponse{
			SessionID: session.ID, Title: session.Title, Provider: session.Provider,
			JoinURL: link, StartsAt: session.StartsAt.Time, OpensAt: opens,
		}
		if session.RecordingMediaID.Valid {
			id := session.RecordingMediaID.UUID
			out.Recording = &id
		}
		return nil
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, out)
}

type recordingRequest struct {
	MediaID string `json:"media_id"`
}

// attachRecording puts the recording back on the class it belongs to, using the
// same media assets as any other lesson video.
func (s *Server) attachRecording(w http.ResponseWriter, r *http.Request) error {
	sessionID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body recordingRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}

	mediaID := uuid.NullUUID{}
	if raw := strings.TrimSpace(body.MediaID); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			return invalid("media_id", "Upload the recording first, then send its media id.")
		}
		mediaID = uuid.NullUUID{UUID: parsed, Valid: true}
	}

	var session database.ClassSession
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		session, err = q.SetSessionRecording(r.Context(), database.SetSessionRecordingParams{
			ID: sessionID, RecordingMediaID: mediaID,
		})
		return err
	})
	if database.IsNotFound(err) || isForeignKeyViolation(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, session)
}
