package assessment_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/ebnsina/fajr-lms/internal/assessment"
)

func TestBuildGradebook(t *testing.T) {
	quizID := uuid.New()
	quizItem := assessment.Item{ID: uuid.New(), QuizID: quizID, Title: "Quiz 1", Possible: 10, Weight: 100}
	examItem := assessment.Item{ID: uuid.New(), Title: "Final exam", Possible: 50, Weight: 300}
	items := []assessment.Item{quizItem, examItem}

	amina := assessment.Learner{EnrollmentID: uuid.New(), FullName: "আমিনা"}
	yusuf := assessment.Learner{EnrollmentID: uuid.New(), FullName: "يوسف"}
	learners := []assessment.Learner{amina, yusuf}

	quizzes := []assessment.QuizScore{
		{QuizID: quizID, EnrollmentID: amina.EnrollmentID, Points: 8},
	}
	overrides := []assessment.Override{
		{ItemID: examItem.ID, EnrollmentID: amina.EnrollmentID, Points: 40, Note: "well argued"},
	}

	got := assessment.BuildGradebook(items, learners, quizzes, overrides)
	if len(got) != 2 {
		t.Fatalf("got %d rows, want 2", len(got))
	}

	t.Run("weights the exam more heavily than the quiz", func(t *testing.T) {
		row := got[0]
		if row.FullName != "আমিনা" || row.Graded != 2 {
			t.Fatalf("got %+v", row)
		}
		// 80% at weight 100 and 80% at weight 300 is 80%.
		if row.Percent != 80 {
			t.Errorf("percent = %d, want 80", row.Percent)
		}
		if !row.Scores[1].Overridden || row.Scores[1].Note != "well argued" {
			t.Errorf("the exam score should be a teacher's: %+v", row.Scores[1])
		}
	})

	t.Run("an unsat item is ungraded, not a zero", func(t *testing.T) {
		row := got[1]
		if row.Graded != 0 || row.Total != 2 {
			t.Fatalf("got %+v", row)
		}
		for _, score := range row.Scores {
			if score.Points != nil || score.Percent != nil {
				t.Errorf("got %+v, want no score", score)
			}
		}
		if row.Percent != 0 {
			t.Errorf("percent = %d, want 0 for a learner with nothing graded", row.Percent)
		}
	})

	t.Run("an override beats the quiz result", func(t *testing.T) {
		with := append(overrides, assessment.Override{
			ItemID: quizItem.ID, EnrollmentID: amina.EnrollmentID, Points: 3, Note: "resat under supervision",
		})
		row := assessment.BuildGradebook(items, []assessment.Learner{amina}, quizzes, with)[0]
		if *row.Scores[0].Points != 3 || !row.Scores[0].Overridden {
			t.Fatalf("got %+v, want the override", row.Scores[0])
		}
		// 30% at weight 100 and 80% at weight 300 is 67.5, rounded to 68.
		if row.Percent != 68 {
			t.Errorf("percent = %d, want 68", row.Percent)
		}
	})

	t.Run("a zero-weight item scores but does not count", func(t *testing.T) {
		practice := assessment.Item{ID: uuid.New(), QuizID: quizID, Possible: 10, Weight: 0}
		row := assessment.BuildGradebook([]assessment.Item{practice}, []assessment.Learner{amina}, quizzes, nil)[0]
		if row.Graded != 1 || *row.Scores[0].Percent != 80 {
			t.Fatalf("got %+v", row)
		}
		if row.Percent != 0 {
			t.Errorf("percent = %d, want 0 when every weight is zero", row.Percent)
		}
	})

	t.Run("no learners and no items are handled", func(t *testing.T) {
		if got := assessment.BuildGradebook(nil, nil, nil, nil); len(got) != 0 {
			t.Errorf("got %d rows, want none", len(got))
		}
		row := assessment.BuildGradebook(nil, []assessment.Learner{amina}, nil, nil)[0]
		if row.Total != 0 || row.Percent != 0 || len(row.Scores) != 0 {
			t.Errorf("got %+v", row)
		}
	})
}
