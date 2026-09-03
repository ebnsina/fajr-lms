package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const endpoint = "https://api.anthropic.com/v1/messages"

// Anthropic drafts questions with Claude. It speaks the REST API directly:
// one call, one shape, no dependency to keep current.
type Anthropic struct {
	Key   string
	Model string
	HTTP  *http.Client
}

func (Anthropic) Name() string { return "anthropic" }

const system = `You write quiz questions for a school, from the lesson a teacher has written.

Rules:
- Ask only about what the lesson itself says. Never add facts of your own.
- Write in the language the lesson is written in.
- Every question must be answerable by somebody who read the lesson and unanswerable by somebody who did not.
- kind is one of: mcq_single, mcq_multi, true_false.
- mcq_single has exactly one correct option and at least three options.
- mcq_multi has at least two correct options.
- true_false has exactly two options, labelled for true and false in the lesson's language, one correct.
- points is between 1 and 5.
- explanation says why the answer is right, in one sentence.

Answer with JSON only: {"questions":[{"kind":"","prompt":"","points":1,"explanation":"","options":[{"label":"","is_correct":false}]}]}
No prose, no code fence.`

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type request struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system"`
	Messages  []message `json:"messages"`
}

type response struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (a Anthropic) DraftQuestions(ctx context.Context, lesson Lesson, count int) ([]Question, error) {
	if a.Key == "" {
		return nil, ErrOff
	}
	body, err := json.Marshal(request{
		Model: a.Model, MaxTokens: 4000, System: system,
		Messages: []message{{
			Role: "user",
			Content: fmt.Sprintf("Write %d questions on this lesson.\n\nTitle: %s\n\n%s",
				count, lesson.Title, lesson.Body),
		}},
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-api-key", a.Key)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := a.HTTP
	if client == nil {
		client = &http.Client{Timeout: 90 * time.Second}
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ai: %w", err)
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	var answer response
	if err := json.Unmarshal(raw, &answer); err != nil {
		return nil, fmt.Errorf("ai: the model answered with something that is not JSON")
	}
	if res.StatusCode != http.StatusOK {
		if answer.Error != nil {
			return nil, fmt.Errorf("ai: %s", answer.Error.Message)
		}
		return nil, fmt.Errorf("ai: the model refused the request (%d)", res.StatusCode)
	}

	var text strings.Builder
	for _, block := range answer.Content {
		if block.Type == "text" {
			text.WriteString(block.Text)
		}
	}
	return parse(text.String())
}

// parse reads the model's answer, forgiving a code fence but nothing else.
func parse(text string) ([]Question, error) {
	trimmed := strings.TrimSpace(text)
	if fence := strings.Index(trimmed, "```"); fence >= 0 {
		trimmed = trimmed[fence+3:]
		trimmed = strings.TrimPrefix(trimmed, "json")
		if end := strings.LastIndex(trimmed, "```"); end >= 0 {
			trimmed = trimmed[:end]
		}
	}
	trimmed = strings.TrimSpace(trimmed)

	var out struct {
		Questions []Question `json:"questions"`
	}
	if err := json.Unmarshal([]byte(trimmed), &out); err != nil {
		return nil, fmt.Errorf("ai: the model did not answer with the shape asked for")
	}
	if len(out.Questions) == 0 {
		return nil, fmt.Errorf("ai: the model drafted nothing from this lesson")
	}
	return out.Questions, nil
}
