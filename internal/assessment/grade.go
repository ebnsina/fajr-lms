// Package assessment holds quiz grading rules, kept free of the database so
// they can be reasoned about and tested on their own.
package assessment

import "github.com/google/uuid"

type Kind string

const (
	MCQSingle   Kind = "mcq_single"
	MCQMulti    Kind = "mcq_multi"
	TrueFalse   Kind = "true_false"
	ShortAnswer Kind = "short_answer"
	Essay       Kind = "essay"
)

// NeedsHuman reports whether a question kind can only be graded by a person.
func (k Kind) NeedsHuman() bool { return k == ShortAnswer || k == Essay }

// Valid reports whether k is a kind this system knows.
func (k Kind) Valid() bool {
	switch k {
	case MCQSingle, MCQMulti, TrueFalse, ShortAnswer, Essay:
		return true
	default:
		return false
	}
}

// Question is what grading needs: the kind, its worth, and which options count.
type Question struct {
	ID      uuid.UUID
	Kind    Kind
	Points  int32
	Correct []uuid.UUID
}

// Answer is what the learner submitted.
type Answer struct {
	QuestionID uuid.UUID
	OptionIDs  []uuid.UUID
	Text       string
}

// Verdict is the outcome for one question.
type Verdict struct {
	QuestionID uuid.UUID
	Points     int32
	NeedsHuman bool
}

// Result is the outcome for a whole attempt.
type Result struct {
	Verdicts       []Verdict
	PointsAwarded  int32
	PointsPossible int32
	NeedsHuman     bool
}

// Grade scores every question, awarding nothing for one left unanswered.
// Choice questions are all-or-nothing; partial credit is a separate decision.
func Grade(questions []Question, answers map[uuid.UUID]Answer) Result {
	out := Result{Verdicts: make([]Verdict, 0, len(questions))}

	for _, q := range questions {
		out.PointsPossible += q.Points
		verdict := Verdict{QuestionID: q.ID}
		answer, answered := answers[q.ID]

		switch {
		case q.Kind.NeedsHuman():
			// An empty essay is a zero, not a queue entry for a marker.
			if answered && answer.Text != "" {
				verdict.NeedsHuman = true
				out.NeedsHuman = true
			}
		case !answered:
		case matches(q.Correct, answer.OptionIDs):
			verdict.Points = q.Points
			out.PointsAwarded += q.Points
		}
		out.Verdicts = append(out.Verdicts, verdict)
	}
	return out
}

// Percent is the score as a whole number, rounded half up.
func Percent(awarded, possible int32) int {
	if possible <= 0 {
		return 0
	}
	return int((int64(awarded)*200 + int64(possible)) / (int64(possible) * 2))
}

// Passed reports whether a score clears the pass mark.
func Passed(awarded, possible int32, passPercent int) bool {
	return Percent(awarded, possible) >= passPercent
}

// matches reports whether two option sets are equal, ignoring order and
// duplicates. Selecting a right answer and a wrong one scores nothing.
func matches(correct, chosen []uuid.UUID) bool {
	if len(correct) == 0 {
		return false
	}
	want := make(map[uuid.UUID]bool, len(correct))
	for _, id := range correct {
		want[id] = true
	}
	got := make(map[uuid.UUID]bool, len(chosen))
	for _, id := range chosen {
		if !want[id] {
			return false
		}
		got[id] = true
	}
	return len(got) == len(want)
}
