package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/httpx"
)

// Points and badges, off unless a school switches them on. What earns points
// is fixed and few: finishing a lesson, passing a quiz, finishing a course.
// A school that wants a leaderboard gets one; a school that thinks ranking
// children is wrong never sees it.
const (
	pointsPerLesson = 10
	pointsPerQuiz   = 25
	pointsPerCourse = 100
)

// award pays for one thing once. It is called from inside the work it rewards,
// and never fails that work: a missing point is not worth losing a grade over.
func (s *Server) award(ctx context.Context, q *database.Queries, tenant database.Tenant,
	userID uuid.UUID, kind string, ref uuid.UUID, points int32) {

	if !tenant.PointsOn {
		return
	}
	if err := q.AwardPoints(ctx, database.AwardPointsParams{
		TenantID: tenant.ID, UserID: userID, Kind: kind, RefID: ref, Points: points,
	}); err != nil {
		return
	}

	// A badge is earned the moment the total passes its mark.
	total, err := q.MyPoints(ctx, userID)
	if err != nil {
		return
	}
	earned, err := q.BadgesToAward(ctx, database.BadgesToAwardParams{
		Points: int32(total), UserID: userID,
	})
	if err != nil {
		return
	}
	for _, badge := range earned {
		_ = q.AwardBadge(ctx, database.AwardBadgeParams{
			BadgeID: badge.ID, UserID: userID, TenantID: tenant.ID,
		})
	}
}

// myStanding is a person's own points and badges, which is the part that works
// even in a school that keeps the leaderboard switched off.
func (s *Server) myStanding(w http.ResponseWriter, r *http.Request) error {
	tenant := CurrentTenant(r.Context())
	userID := Authenticated(r.Context()).UserID

	var points int64
	var badges []database.MyBadgesRow
	err := s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		var err error
		if points, err = q.MyPoints(r.Context(), userID); err != nil {
			return err
		}
		badges, err = q.MyBadges(r.Context(), userID)
		return err
	})
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{
		"on": tenant.PointsOn, "points": points, "badges": badges,
	})
}

// leaderboard is this month by default, because a board that never resets
// belongs to whoever joined first.
func (s *Server) leaderboard(w http.ResponseWriter, r *http.Request) error {
	tenant := CurrentTenant(r.Context())
	if !tenant.PointsOn {
		return httpx.Errorf(http.StatusConflict, "points_off",
			"This school does not keep a leaderboard.")
	}

	since := time.Now().AddDate(0, -1, 0)
	if r.URL.Query().Get("window") == "all" {
		since = time.Unix(0, 0)
	}
	limit, _, err := pagination(r)
	if err != nil {
		return err
	}

	var rows []database.LeaderboardRow
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		var err error
		rows, err = q.Leaderboard(r.Context(), database.LeaderboardParams{
			Since: pgtype.Timestamptz{Time: since, Valid: true}, PageLimit: limit,
		})
		return err
	})
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"standings": rows})
}

type badgeRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Emoji       string `json:"emoji"`
	Threshold   int32  `json:"threshold"`
}

func (s *Server) createBadge(w http.ResponseWriter, r *http.Request) error {
	var body badgeRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	name, err := requireText("name", body.Name, 60)
	if err != nil {
		return err
	}
	if body.Threshold <= 0 {
		return invalid("threshold", "A badge is earned at some number of points.")
	}
	if len(body.Description) > 300 {
		return invalid("description", "Keep it under 300 characters.")
	}
	if len(body.Emoji) > 8 {
		return invalid("emoji", "One emoji is enough.")
	}

	tenant := CurrentTenant(r.Context())
	var badge database.Badge
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		var err error
		badge, err = q.CreateBadge(r.Context(), database.CreateBadgeParams{
			TenantID: tenant.ID, Name: name, Description: strings.TrimSpace(body.Description),
			Emoji: strings.TrimSpace(body.Emoji), Threshold: body.Threshold,
		})
		return err
	})
	if isUniqueViolation(err) {
		return httpx.Errorf(http.StatusConflict, "badge_exists", "This school already has a badge by that name.")
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, badge)
}

func (s *Server) listBadges(w http.ResponseWriter, r *http.Request) error {
	var rows []database.ListBadgesRow
	err := s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		rows, err = q.ListBadges(r.Context())
		return err
	})
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"badges": rows})
}

func (s *Server) deleteBadge(w http.ResponseWriter, r *http.Request) error {
	return s.removeRow(w, r, func(q *database.Queries, id uuid.UUID) (int64, error) {
		return q.DeleteBadge(r.Context(), id)
	})
}

type pointsSettingRequest struct {
	On bool `json:"on"`
}

func (s *Server) setPointsOn(w http.ResponseWriter, r *http.Request) error {
	var body pointsSettingRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	tenant := CurrentTenant(r.Context())
	var updated database.Tenant
	err := s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		var err error
		updated, err = q.SetPointsOn(r.Context(), database.SetPointsOnParams{
			ID: tenant.ID, PointsOn: body.On,
		})
		return err
	})
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"on": updated.PointsOn})
}
