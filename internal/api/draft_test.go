package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/ebnsina/fajr-lms/internal/ai"
	"github.com/ebnsina/fajr-lms/internal/api"
	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/identity"
)

// stubDrafter answers with whatever the test hands it, so the endpoint can be
// checked without a model and without a network.
type stubDrafter struct {
	questions []ai.Question
	err       error
}

func (stubDrafter) Name() string { return "stub" }

func (s stubDrafter) DraftQuestions(context.Context, ai.Lesson, int) ([]ai.Question, error) {
	return s.questions, s.err
}

func newDraftHarness(t *testing.T, drafter ai.Drafter) (http.Handler, *captureChannel, *database.Store) {
	t.Helper()
	url := os.Getenv("FAJR_DATABASE_URL")
	if url == "" {
		t.Skip("FAJR_DATABASE_URL not set")
	}
	store, err := database.Open(context.Background(), url)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(store.Close)

	ch := &captureChannel{}
	server := api.NewServer(store, identity.New(store, ch), testRegistry(t), testPayments(t),
		"https://fajr.test")
	server.UseAI(drafter)
	return server.Routes(), ch, store
}

func draftLesson(t *testing.T, h http.Handler, owner actor, title, body string) string {
	t.Helper()
	courseID := createdID(t, do(t, h, "POST", "/v1/courses", owner.token, owner.slug,
		map[string]any{"title": title, "visibility": "public"}))
	moduleID := createdID(t, do(t, h, "POST", "/v1/courses/"+courseID+"/modules", owner.token, owner.slug,
		map[string]any{"title": "Unit"}))
	return createdID(t, do(t, h, "POST", "/v1/modules/"+moduleID+"/lessons", owner.token, owner.slug,
		map[string]any{"title": "Roots", "kind": "text", "body": body}))
}

func TestDraftQuestions(t *testing.T) {
	good := []ai.Question{
		{Kind: "mcq_single", Prompt: "How many root letters?", Points: 2, Explanation: "Three.",
			Options: []ai.Option{{Label: "Two"}, {Label: "Three", IsCorrect: true}, {Label: "Four"}}},
		// Two right answers on a single-answer question: unusable, and dropped.
		{Kind: "mcq_single", Prompt: "Which is a letter?", Points: 1,
			Options: []ai.Option{{Label: "A", IsCorrect: true}, {Label: "B", IsCorrect: true}, {Label: "C"}}},
		// A kind no marker could grade automatically.
		{Kind: "essay", Prompt: "Discuss.", Points: 3},
	}

	h, ch, store := newDraftHarness(t, stubDrafter{questions: good})
	owner := enroll(t, h, ch, store, "owner")
	student := enrollIn(t, h, ch, store, owner.slug, "student")
	lessonID := draftLesson(t, h, owner, "Grammar",
		strings.Repeat("Arabic verbs are built on three root letters. ", 8))

	t.Run("only what could be graded comes back", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/lessons/"+lessonID+"/quiz/draft", owner.token, owner.slug, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d: %s", rec.Code, rec.Body)
		}
		var out struct {
			Questions []ai.Question `json:"questions"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(out.Questions) != 1 || out.Questions[0].Prompt != "How many root letters?" {
			t.Fatalf("got %+v, want only the one usable question", out.Questions)
		}
	})

	t.Run("a learner cannot draft", func(t *testing.T) {
		rec := do(t, h, "POST", "/v1/lessons/"+lessonID+"/quiz/draft", student.token, owner.slug, nil)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("got %d, want 403: %s", rec.Code, rec.Body)
		}
	})

	t.Run("an empty lesson has nothing to ask about", func(t *testing.T) {
		bare := draftLesson(t, h, owner, "Grammar notes", "Short.")
		rec := do(t, h, "POST", "/v1/lessons/"+bare+"/quiz/draft", owner.token, owner.slug, nil)
		if rec.Code != http.StatusConflict {
			t.Fatalf("got %d, want 409: %s", rec.Code, rec.Body)
		}
	})

	t.Run("nothing usable is an error, not an empty list", func(t *testing.T) {
		h, ch, store := newDraftHarness(t, stubDrafter{questions: good[1:]})
		owner := enroll(t, h, ch, store, "owner")
		lessonID := draftLesson(t, h, owner, "Grammar",
			strings.Repeat("Arabic verbs are built on three root letters. ", 8))
		rec := do(t, h, "POST", "/v1/lessons/"+lessonID+"/quiz/draft", owner.token, owner.slug, nil)
		if rec.Code != http.StatusBadGateway {
			t.Fatalf("got %d, want 502: %s", rec.Code, rec.Body)
		}
	})

	t.Run("without a model the endpoint says so", func(t *testing.T) {
		h, ch, store := newHarness(t)
		owner := enroll(t, h, ch, store, "owner")
		lessonID := draftLesson(t, h, owner, "Grammar",
			strings.Repeat("Arabic verbs are built on three root letters. ", 8))
		rec := do(t, h, "POST", "/v1/lessons/"+lessonID+"/quiz/draft", owner.token, owner.slug, nil)
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("got %d, want 503: %s", rec.Code, rec.Body)
		}
	})
}
