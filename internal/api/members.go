package api

import (
	"net/http"
	"strings"

	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/httpx"
	"github.com/ebnsina/fajr-lms/internal/identity"
)

type inviteRequest struct {
	FullName    string `json:"full_name"`
	Destination string `json:"destination"`
	Role        string `json:"role"`
}

func parseRole(raw string) (database.MemberRole, error) {
	role := database.MemberRole(strings.TrimSpace(raw))
	switch role {
	case database.MemberRoleOwner, database.MemberRoleAdmin, database.MemberRoleInstructor,
		database.MemberRoleAssistant, database.MemberRoleStudent, database.MemberRoleParent:
		return role, nil
	}
	return "", invalid("role", "Choose one of the roles this school uses.")
}

// invite adds somebody to the school by phone or email. If they have never used
// Fajr they get an account here; they still have to sign in with that number to
// reach it, so nothing is handed out on somebody else's say-so.
func (s *Server) invite(w http.ResponseWriter, r *http.Request) error {
	var body inviteRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	name, err := requireText("full_name", body.FullName, 200)
	if err != nil {
		return err
	}
	role, err := parseRole(body.Role)
	if err != nil {
		return err
	}
	dest, err := identity.ParseDestination(body.Destination)
	if err != nil {
		return invalid("destination", "Enter a phone number with its country code, or an email address.")
	}

	// Finding and creating a person happens before any tenant scope exists, so
	// it goes through the audited functions rather than a plain table read.
	unscoped := s.store.Unscoped()
	user, err := unscoped.FindUserForAuth(r.Context(), database.FindUserForAuthParams{
		Phone: dest.Phone, Email: dest.Email,
	})
	if database.IsNotFound(err) {
		user, err = unscoped.SignupUser(r.Context(), database.SignupUserParams{
			Phone: dest.Phone, Email: dest.Email, FullName: name,
		})
	}
	if err != nil {
		return err
	}

	tenant := CurrentTenant(r.Context())
	var membership database.Membership
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		var err error
		membership, err = q.CreateMembership(r.Context(), database.CreateMembershipParams{
			TenantID: tenant.ID, UserID: user.ID, Role: role,
		})
		return err
	})
	if isUniqueViolation(err) {
		return &httpx.Error{Status: http.StatusConflict, Code: "already_a_member",
			Message: "That person is already in this school.", Field: "destination"}
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, map[string]any{
		"membership": membership, "full_name": user.FullName,
	})
}

type roleRequest struct {
	Role string `json:"role"`
}

// setMemberRole changes what somebody may do. The last owner keeps the school,
// so a school can never be left without one.
func (s *Server) setMemberRole(w http.ResponseWriter, r *http.Request) error {
	userID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body roleRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	role, err := parseRole(body.Role)
	if err != nil {
		return err
	}

	tenant := CurrentTenant(r.Context())
	var membership database.Membership
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		current, err := q.GetMembership(r.Context(), database.GetMembershipParams{
			TenantID: tenant.ID, UserID: userID,
		})
		if err != nil {
			return err
		}
		if err := lastOwnerCheck(r, q, current.Role, role); err != nil {
			return err
		}
		membership, err = q.SetMembershipRole(r.Context(), database.SetMembershipRoleParams{
			TenantID: tenant.ID, UserID: userID, Role: role,
		})
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, membership)
}

func (s *Server) removeMember(w http.ResponseWriter, r *http.Request) error {
	userID, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	if userID == Authenticated(r.Context()).UserID {
		return httpx.Errorf(http.StatusConflict, "cannot_remove_self",
			"You cannot remove yourself. Ask another owner to do it.")
	}

	tenant := CurrentTenant(r.Context())
	var rows int64
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		current, err := q.GetMembership(r.Context(), database.GetMembershipParams{
			TenantID: tenant.ID, UserID: userID,
		})
		if err != nil {
			return err
		}
		if err := lastOwnerCheck(r, q, current.Role, database.MemberRoleStudent); err != nil {
			return err
		}
		rows, err = q.DeleteMembership(r.Context(), database.DeleteMembershipParams{
			TenantID: tenant.ID, UserID: userID,
		})
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	if rows == 0 {
		return httpx.ErrNotFound
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// lastOwnerCheck refuses a change that would leave the school with no owner.
func lastOwnerCheck(r *http.Request, q *database.Queries, from, to database.MemberRole) error {
	if from != database.MemberRoleOwner || to == database.MemberRoleOwner {
		return nil
	}
	owners, err := q.CountOwners(r.Context())
	if err != nil {
		return err
	}
	if owners <= 1 {
		return httpx.Errorf(http.StatusConflict, "last_owner",
			"This is the only owner. Make somebody else an owner first.")
	}
	return nil
}
