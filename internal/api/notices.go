package api

import (
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/httpx"
)

type noticeRequest struct {
	Audience string `json:"audience"`
	TargetID string `json:"target_id"`
	To       string `json:"to"`
	Title    string `json:"title"`
	Body     string `json:"body"`
}

// sendNotice tells a section, a class or the whole school something: an
// absence, a fee due, a holiday. It reaches guardians the way they are already
// reachable — the inbox here, and SMS where the school has a gateway.
func (s *Server) sendNotice(w http.ResponseWriter, r *http.Request) error {
	var body noticeRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	title, err := requireText("title", body.Title, 200)
	if err != nil {
		return err
	}
	message, err := requireText("body", body.Body, 2000)
	if err != nil {
		return err
	}

	to := strings.ToLower(strings.TrimSpace(body.To))
	if to == "" {
		to = "guardians"
	}
	if to != "guardians" && to != "students" && to != "both" {
		return invalid("to", "Send it to guardians, students, or both.")
	}

	audience := strings.ToLower(strings.TrimSpace(body.Audience))
	var target uuid.UUID
	if audience == "section" || audience == "class" {
		if target, err = uuid.Parse(strings.TrimSpace(body.TargetID)); err != nil {
			return invalid("target_id", "Name the section or class this is for.")
		}
	} else if audience != "school" {
		return invalid("audience", "Send it to a section, a class, or the whole school.")
	}

	tenant := CurrentTenant(r.Context())
	// Deduplicated: a guardian with two children in the same class hears once.
	recipients := map[uuid.UUID]bool{}

	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		add := func(student uuid.UUID, guardian uuid.NullUUID) {
			if to == "students" || to == "both" {
				recipients[student] = true
			}
			if guardian.Valid && (to == "guardians" || to == "both") {
				recipients[guardian.UUID] = true
			}
		}

		switch audience {
		case "section":
			rows, err := q.SectionAudience(r.Context(), target)
			if err != nil {
				return err
			}
			for _, row := range rows {
				add(row.StudentID, row.GuardianID)
			}
		case "class":
			rows, err := q.ClassAudience(r.Context(), target)
			if err != nil {
				return err
			}
			for _, row := range rows {
				add(row.StudentID, row.GuardianID)
			}
		default:
			rows, err := q.SchoolAudience(r.Context())
			if err != nil {
				return err
			}
			for _, row := range rows {
				add(row.StudentID, row.GuardianID)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(recipients) == 0 {
		return httpx.Errorf(http.StatusConflict, "nobody_to_tell",
			"Nobody there has an account to send this to yet.")
	}

	for userID := range recipients {
		s.notifyUser(r.Context(), tenant.ID, userID, "notice", title, message,
			map[string]any{"audience": audience})
	}
	return httpx.JSON(w, http.StatusAccepted, map[string]any{"sent_to": len(recipients)})
}

// myChildren is where a guardian starts: who they are guardian of, and where
// each of them sits this year.
func (s *Server) myChildren(w http.ResponseWriter, r *http.Request) error {
	var rows []database.MyChildrenRow
	err := s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		rows, err = q.MyChildren(r.Context(), Authenticated(r.Context()).UserID)
		return err
	})
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"children": rows})
}
