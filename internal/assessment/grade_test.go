package assessment_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/ebnsina/fajr-lms/internal/assessment"
)

func TestGrade(t *testing.T) {
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	single := assessment.Question{ID: uuid.New(), Kind: assessment.MCQSingle, Points: 2, Correct: []uuid.UUID{a}}
	multi := assessment.Question{ID: uuid.New(), Kind: assessment.MCQMulti, Points: 3, Correct: []uuid.UUID{a, b}}
	essay := assessment.Question{ID: uuid.New(), Kind: assessment.Essay, Points: 5}
	questions := []assessment.Question{single, multi, essay}

	cases := []struct {
		name       string
		answers    map[uuid.UUID]assessment.Answer
		want       int32
		needsHuman bool
	}{
		{
			name: "everything right, essay pending",
			answers: map[uuid.UUID]assessment.Answer{
				single.ID: {OptionIDs: []uuid.UUID{a}},
				multi.ID:  {OptionIDs: []uuid.UUID{b, a}},
				essay.ID:  {Text: "كتبت الإجابة"},
			},
			want: 5, needsHuman: true,
		},
		{
			name: "a right answer plus a wrong one scores nothing",
			answers: map[uuid.UUID]assessment.Answer{
				multi.ID: {OptionIDs: []uuid.UUID{a, b, c}},
			},
		},
		{
			name: "a partial selection scores nothing",
			answers: map[uuid.UUID]assessment.Answer{
				multi.ID: {OptionIDs: []uuid.UUID{a}},
			},
		},
		{
			name: "duplicates do not double count or break equality",
			answers: map[uuid.UUID]assessment.Answer{
				multi.ID: {OptionIDs: []uuid.UUID{a, a, b, b}},
			},
			want: 3,
		},
		{
			name:    "an empty essay is a zero, not a marking job",
			answers: map[uuid.UUID]assessment.Answer{essay.ID: {Text: ""}},
		},
		{
			name:    "nothing answered scores nothing",
			answers: map[uuid.UUID]assessment.Answer{},
		},
		{
			name: "an empty selection is not a correct answer",
			answers: map[uuid.UUID]assessment.Answer{
				single.ID: {OptionIDs: nil},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := assessment.Grade(questions, tc.answers)
			if got.PointsAwarded != tc.want {
				t.Errorf("awarded %d, want %d", got.PointsAwarded, tc.want)
			}
			if got.PointsPossible != 10 {
				t.Errorf("possible %d, want 10", got.PointsPossible)
			}
			if got.NeedsHuman != tc.needsHuman {
				t.Errorf("needsHuman = %v, want %v", got.NeedsHuman, tc.needsHuman)
			}
			if len(got.Verdicts) != len(questions) {
				t.Errorf("got %d verdicts, want %d", len(got.Verdicts), len(questions))
			}
		})
	}
}

func TestPercentAndPass(t *testing.T) {
	cases := []struct {
		awarded, possible int32
		want              int
	}{
		{0, 0, 0}, {0, 10, 0}, {10, 10, 100}, {5, 10, 50},
		{1, 3, 33}, {2, 3, 67}, {1, 6, 17}, {7, 8, 88},
	}
	for _, c := range cases {
		if got := assessment.Percent(c.awarded, c.possible); got != c.want {
			t.Errorf("Percent(%d, %d) = %d, want %d", c.awarded, c.possible, got, c.want)
		}
	}

	if !assessment.Passed(5, 10, 50) {
		t.Error("exactly the pass mark should pass")
	}
	if assessment.Passed(4, 10, 50) {
		t.Error("below the pass mark should fail")
	}
	// A quiz with no questions cannot be passed by accident.
	if assessment.Passed(0, 0, 50) {
		t.Error("an empty quiz should not pass")
	}
}

func TestKindValidity(t *testing.T) {
	for _, k := range []assessment.Kind{"mcq_single", "mcq_multi", "true_false", "short_answer", "essay"} {
		if !k.Valid() {
			t.Errorf("%q should be valid", k)
		}
	}
	for _, k := range []assessment.Kind{"", "matching", "MCQ_SINGLE"} {
		if k.Valid() {
			t.Errorf("%q should not be valid", k)
		}
	}
	if !assessment.Essay.NeedsHuman() || assessment.MCQSingle.NeedsHuman() {
		t.Error("only written answers need a human")
	}
}
