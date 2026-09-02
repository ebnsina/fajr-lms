package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/httpx"
	"github.com/ebnsina/fajr-lms/internal/site"
)

// pageView sends blocks as JSON rather than the base64 a []byte would become.
type pageView struct {
	database.SitePage
	Blocks json.RawMessage `json:"blocks"`
}

func view(p database.SitePage) pageView { return pageView{SitePage: p, Blocks: p.Blocks} }

type publishedView struct {
	database.PublishedPage
	Blocks json.RawMessage `json:"blocks"`
}

type sitePageRequest struct {
	Slug        string          `json:"slug"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Dir         string          `json:"dir"`
	NavLabel    string          `json:"nav_label"`
	NavOrder    *int32          `json:"nav_order"`
	Blocks      json.RawMessage `json:"blocks"`
}

// pageSlug accepts an empty slug, which is the site's front page.
func pageSlug(raw, title string) (string, error) {
	slug := strings.TrimSpace(raw)
	if slug == "" {
		return "", nil
	}
	if slug == "home" || slug == "index" {
		return "", nil
	}
	if !slugRe.MatchString(slug) {
		slug = Slugify(title)
	}
	if !slugRe.MatchString(slug) {
		return "", invalid("slug", "Use lowercase letters, numbers and hyphens.")
	}
	return slug, nil
}

func (s *Server) createSitePage(w http.ResponseWriter, r *http.Request) error {
	var body sitePageRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	title, err := requireText("title", body.Title, 200)
	if err != nil {
		return err
	}
	slug, err := pageSlug(body.Slug, title)
	if err != nil {
		return err
	}
	dir, err := parseDir(body.Dir)
	if err != nil {
		return err
	}
	blocks, err := encodeBlocks(body.Blocks)
	if err != nil {
		return err
	}

	var order int32
	if body.NavOrder != nil {
		order = *body.NavOrder
	}
	tenant := CurrentTenant(r.Context())
	var page database.SitePage
	err = s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		var err error
		page, err = q.CreateSitePage(r.Context(), database.CreateSitePageParams{
			TenantID: tenant.ID, Slug: slug, Title: title,
			Description: strings.TrimSpace(body.Description), Dir: dir, Blocks: blocks,
			NavLabel: strings.TrimSpace(body.NavLabel), NavOrder: order,
			UpdatedBy: uuid.NullUUID{UUID: Authenticated(r.Context()).UserID, Valid: true},
		})
		return err
	})
	if isUniqueViolation(err) {
		return &httpx.Error{Status: http.StatusConflict, Code: "slug_taken",
			Message: "A page with this address already exists.", Field: "slug"}
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, view(page))
}

func (s *Server) listSitePages(w http.ResponseWriter, r *http.Request) error {
	var pages []database.SitePage
	err := s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		pages, err = q.ListSitePages(r.Context())
		return err
	})
	if err != nil {
		return err
	}
	views := make([]pageView, len(pages))
	for i, page := range pages {
		views[i] = view(page)
	}
	return httpx.JSON(w, http.StatusOK, map[string]any{"pages": views, "kinds": site.Kinds()})
}

func (s *Server) getSitePage(w http.ResponseWriter, r *http.Request) error {
	id, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var page database.SitePage
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		page, err = q.GetSitePage(r.Context(), id)
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, view(page))
}

func (s *Server) updateSitePage(w http.ResponseWriter, r *http.Request) error {
	id, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body sitePageRequest
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}

	params := database.UpdateSitePageParams{
		ID:        id,
		UpdatedBy: uuid.NullUUID{UUID: Authenticated(r.Context()).UserID, Valid: true},
	}
	if raw := strings.TrimSpace(body.Title); raw != "" {
		title, err := requireText("title", raw, 200)
		if err != nil {
			return err
		}
		params.Title = &title
	}
	if body.Description != "" {
		description := strings.TrimSpace(body.Description)
		params.Description = &description
	}
	if body.Dir != "" {
		dir, err := parseDir(body.Dir)
		if err != nil {
			return err
		}
		params.Dir = &dir
	}
	if body.NavLabel != "" {
		label := strings.TrimSpace(body.NavLabel)
		params.NavLabel = &label
	}
	if body.NavOrder != nil {
		params.NavOrder = body.NavOrder
	}
	if body.Blocks != nil {
		blocks, err := encodeBlocks(body.Blocks)
		if err != nil {
			return err
		}
		params.Blocks = blocks
	}

	var page database.SitePage
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		page, err = q.UpdateSitePage(r.Context(), params)
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, view(page))
}

func (s *Server) setSitePageStatus(w http.ResponseWriter, r *http.Request) error {
	id, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	status, err := parseStatus(body.Status)
	if err != nil {
		return err
	}

	var page database.SitePage
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		page, err = q.SetSitePageStatus(r.Context(), database.SetSitePageStatusParams{
			ID: id, Status: status,
			UpdatedBy: uuid.NullUUID{UUID: Authenticated(r.Context()).UserID, Valid: true},
		})
		return err
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, view(page))
}

func (s *Server) deleteSitePage(w http.ResponseWriter, r *http.Request) error {
	id, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var rows int64
	err = s.store.InTenant(r.Context(), CurrentTenant(r.Context()).ID, func(q *database.Queries) error {
		var err error
		rows, err = q.DeleteSitePage(r.Context(), id)
		return err
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return httpx.ErrNotFound
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// publicPage serves a published page to anyone, with the site's navigation and
// the courses it may need to list. Drafts are not in the view it reads.
func (s *Server) publicPage(w http.ResponseWriter, r *http.Request) error {
	tenantSlug := strings.ToLower(strings.TrimSpace(r.PathValue("tenant")))
	slug := strings.TrimSpace(r.PathValue("slug"))
	if !slugRe.MatchString(tenantSlug) || (slug != "" && !slugRe.MatchString(slug)) {
		return httpx.ErrNotFound
	}

	q := s.store.Unscoped()
	page, err := q.GetPublishedPage(r.Context(), database.GetPublishedPageParams{
		TenantSlug: tenantSlug, Slug: slug,
	})
	if database.IsNotFound(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	nav, err := q.ListSiteNav(r.Context(), tenantSlug)
	if err != nil {
		return err
	}

	body := map[string]any{
		"page":  publishedView{PublishedPage: page, Blocks: page.Blocks},
		"nav":   nav,
		"theme": page.SiteTheme,
	}
	if courses, err := s.publicCourses(r, page); err != nil {
		return err
	} else if courses != nil {
		body["courses"] = courses
	}
	return httpx.JSON(w, http.StatusOK, body)
}

// publicCourses loads the catalog only when a page actually lists it.
func (s *Server) publicCourses(r *http.Request, page database.PublishedPage) ([]database.PublishedCourse, error) {
	blocks, err := site.Parse(page.Blocks)
	if err != nil {
		return nil, nil
	}
	limit := 0
	for _, b := range blocks {
		if b.Type == "courses" {
			if b.Limit == 0 {
				b.Limit = 6
			}
			if b.Limit > limit {
				limit = b.Limit
			}
		}
	}
	if limit == 0 {
		return nil, nil
	}
	return s.store.Unscoped().ListPublishedCourses(r.Context(), database.ListPublishedCoursesParams{
		TenantSlug: page.TenantSlug, PageLimit: int32(limit),
	})
}

// setSiteTheme dresses the public site for where the school teaches.
func (s *Server) setSiteTheme(w http.ResponseWriter, r *http.Request) error {
	var body struct {
		Theme string `json:"theme"`
	}
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	theme := strings.ToLower(strings.TrimSpace(body.Theme))
	switch theme {
	case "plain", "gulf", "bengal":
	default:
		return invalid("theme", "Choose plain, gulf or bengal.")
	}

	tenant := CurrentTenant(r.Context())
	var updated database.Tenant
	err := s.store.InTenant(r.Context(), tenant.ID, func(q *database.Queries) error {
		var err error
		updated, err = q.SetSiteTheme(r.Context(), database.SetSiteThemeParams{ID: tenant.ID, SiteTheme: theme})
		return err
	})
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, updated)
}

func encodeBlocks(raw json.RawMessage) ([]byte, error) {
	blocks, err := site.Parse(raw)
	if err != nil {
		return nil, invalid("blocks", capitalise(err.Error())+".")
	}
	return site.Encode(blocks)
}

func capitalise(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
