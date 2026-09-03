// Package ai drafts teaching material from a lesson. What it returns is a
// suggestion: a teacher reads it, edits it and decides, and nothing reaches a
// learner until they do.
package ai

import (
	"context"
	"errors"
)

// ErrOff is returned when no model is configured, which is the default.
var ErrOff = errors.New("ai: no model is configured")

// Question is one drafted question, in the shape the quiz builder already uses.
type Question struct {
	Kind        string   `json:"kind"`
	Prompt      string   `json:"prompt"`
	Points      int32    `json:"points"`
	Explanation string   `json:"explanation"`
	Options     []Option `json:"options"`
}

type Option struct {
	Label     string `json:"label"`
	IsCorrect bool   `json:"is_correct"`
}

// Lesson is the material a draft is made from.
type Lesson struct {
	Title string
	Body  string
	Dir   string
}

// Drafter is the seam a model plugs into.
type Drafter interface {
	DraftQuestions(ctx context.Context, lesson Lesson, count int) ([]Question, error)
	Name() string
}

// Off is the default: it drafts nothing and says so.
type Off struct{}

func (Off) Name() string { return "off" }

func (Off) DraftQuestions(context.Context, Lesson, int) ([]Question, error) {
	return nil, ErrOff
}
