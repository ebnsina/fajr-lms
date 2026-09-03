package api

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/httpx"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// The back office. These routes read across every school, which is the one
// thing the rest of the API is built to make impossible — so they are few,
// they are guarded by their own sign-in, and each one is written down.

type staffKeyType struct{}

var staffKey staffKeyType

type staffLogin struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// adminLogin is a password, not a code: this is one account, not a school full
// of people, and a code to a phone helps nobody at three in the morning.
func (s *Server) adminLogin(w http.ResponseWriter, r *http.Request) error {
	var body staffLogin
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	email := strings.ToLower(strings.TrimSpace(body.Email))

	q := s.store.Unscoped()
	staff, err := q.StaffByEmail(r.Context(), email)
	if database.IsNotFound(err) {
		return wrongStaffPassword()
	}
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(staff.PasswordHash), []byte(body.Password)) != nil {
		return wrongStaffPassword()
	}

	session, err := s.identity.SignInUnverified(r.Context(), email, "", r.UserAgent(), clientIP(r))
	if err != nil {
		return err
	}
	if err := q.TouchStaff(r.Context(), staff.UserID); err != nil {
		return err
	}
	s.logStaff(r.Context(), staff.UserID, "signed in", email, "")

	return httpx.JSON(w, http.StatusOK, map[string]any{
		"token": session.Token, "expires_at": session.ExpiresAt, "role": staff.Role,
	})
}

// The same answer either way, so the reply cannot say who is staff.
func wrongStaffPassword() error {
	return httpx.Errorf(http.StatusUnauthorized, "wrong_password", "That is not the right address or password.")
}

// RequireStaff admits only the platform's own people, and only after a live
// session. Everyone else is told the route is not there.
func (s *Server) RequireStaff(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := Authenticated(r.Context())
		role, err := s.store.Unscoped().StaffRole(r.Context(), session.UserID)
		if database.IsNotFound(err) || (err == nil && role == nil) {
			httpx.WriteError(w, r, httpx.ErrNotFound)
			return
		}
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), staffKey, *role)))
	})
}

// logStaff writes what was done before it is done, so an action that fails
// halfway is still on the record.
func (s *Server) logStaff(ctx context.Context, userID uuid.UUID, action, subject, detail string) {
	_ = s.store.Unscoped().LogStaffAction(ctx, database.LogStaffActionParams{
		UserID: userID, Action: action, Subject: subject, Detail: detail,
	})
}

func (s *Server) adminOverview(w http.ResponseWriter, r *http.Request) error {
	numbers, err := s.store.Unscoped().AdminOverview(r.Context())
	if err != nil {
		return err
	}
	return writeJSONDocument(w, numbers)
}

func (s *Server) adminTenants(w http.ResponseWriter, r *http.Request) error {
	limit, offset, err := pagination(r)
	if err != nil {
		return err
	}
	rows, err := s.store.Unscoped().AdminTenants(r.Context(), database.AdminTenantsParams{
		Query: trim(r.URL.Query().Get("q"), 120), LimitTo: int32(limit), OffsetBy: int32(offset),
	})
	if err != nil {
		return err
	}
	return writeJSONList(w, "schools", rows)
}

func (s *Server) adminTenant(w http.ResponseWriter, r *http.Request) error {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return httpx.ErrNotFound
	}
	row, err := s.store.Unscoped().AdminTenant(r.Context(), id)
	if database.IsNotFound(err) || len(row) == 0 || string(row) == "null" {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	s.logStaff(r.Context(), Authenticated(r.Context()).UserID, "read a school", id.String(), "")
	return writeJSONDocument(w, row)
}

func (s *Server) adminLeads(w http.ResponseWriter, r *http.Request) error {
	leads, err := s.leadRows(r)
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"leads": leads})
}

// adminLeadsCSV exists because the customer relationship lives in somebody
// else's product, and always will.
func (s *Server) adminLeadsCSV(w http.ResponseWriter, r *http.Request) error {
	leads, err := s.leadRows(r)
	if err != nil {
		return err
	}
	s.logStaff(r.Context(), Authenticated(r.Context()).UserID, "exported leads",
		strconv.Itoa(len(leads)), "")

	w.Header().Set("content-type", "text/csv; charset=utf-8")
	w.Header().Set("content-disposition", `attachment; filename="fajr-leads.csv"`)
	w.Header().Set("cache-control", "private, no-store")

	out := csv.NewWriter(w)
	_ = out.Write([]string{"when", "name", "email", "phone", "organisation", "role",
		"learners", "runs", "state", "note", "what they asked"})
	for _, lead := range leads {
		_ = out.Write([]string{
			lead.CreatedAt.Time.Format("2006-01-02 15:04"), lead.FullName, lead.Email,
			lead.Phone, lead.Organisation, lead.Role, lead.Learners, lead.Runs,
			lead.State, lead.WorkedNote, lead.Note,
		})
	}
	out.Flush()
	return out.Error()
}

func (s *Server) leadRows(r *http.Request) ([]database.DemoLead, error) {
	limit, offset, err := pagination(r)
	if err != nil {
		return nil, err
	}
	state := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("state")))
	if state != "" && !leadStates[state] {
		return nil, invalid("state", "That is not a state a lead can be in.")
	}
	return s.store.Unscoped().AdminLeads(r.Context(), database.AdminLeadsParams{
		State: state, Query: trim(r.URL.Query().Get("q"), 120),
		LimitTo: int32(limit), OffsetBy: int32(offset),
	})
}

var leadStates = map[string]bool{
	"new": true, "contacted": true, "qualified": true, "won": true, "lost": true,
}

type leadUpdate struct {
	State string `json:"state"`
	Note  string `json:"note"`
}

func (s *Server) adminSetLead(w http.ResponseWriter, r *http.Request) error {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return httpx.ErrNotFound
	}
	var body leadUpdate
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	state := strings.ToLower(strings.TrimSpace(body.State))
	if !leadStates[state] {
		return invalid("state", "A lead is new, contacted, qualified, won or lost.")
	}

	s.logStaff(r.Context(), Authenticated(r.Context()).UserID, "worked a lead", id.String(), state)
	lead, err := s.store.Unscoped().AdminSetLead(r.Context(), database.AdminSetLeadParams{
		ID: id, State: state, Note: trim(body.Note, 2000),
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, lead)
}

// The views come back from Postgres as JSON already, so they are passed
// through rather than decoded into a shape only to be encoded again.
func writeJSONDocument(w http.ResponseWriter, document []byte) error {
	w.Header().Set("content-type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write(document)
	return err
}

func writeJSONList(w http.ResponseWriter, name string, list []byte) error {
	if len(list) == 0 {
		list = []byte("[]")
	}
	wrapped, err := json.Marshal(map[string]json.RawMessage{name: list})
	if err != nil {
		return err
	}
	return writeJSONDocument(w, wrapped)
}
