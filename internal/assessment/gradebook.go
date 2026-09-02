package assessment

import "github.com/google/uuid"

// Item is one graded thing in a course.
type Item struct {
	ID       uuid.UUID
	QuizID   uuid.UUID
	Title    string
	Category string
	Possible int32
	Weight   int32
}

// Score is one learner's result for one item.
type Score struct {
	ItemID     uuid.UUID `json:"item_id"`
	Points     *int32    `json:"points"`
	Possible   int32     `json:"points_possible"`
	Percent    *int      `json:"percent"`
	Overridden bool      `json:"overridden"`
	Note       string    `json:"note,omitempty"`
}

// Report is one learner's whole gradebook row.
type Report struct {
	EnrollmentID uuid.UUID `json:"enrollment_id"`
	FullName     string    `json:"full_name"`
	Scores       []Score   `json:"scores"`
	Percent      int       `json:"percent"`
	Graded       int       `json:"items_graded"`
	Total        int       `json:"items_total"`
}

// Override is a score a teacher entered or corrected.
type Override struct {
	ItemID       uuid.UUID
	EnrollmentID uuid.UUID
	Points       int32
	Note         string
}

// QuizScore is a learner's best graded attempt at a quiz.
type QuizScore struct {
	QuizID       uuid.UUID
	EnrollmentID uuid.UUID
	Points       int32
}

// Learner identifies a row of the gradebook.
type Learner struct {
	EnrollmentID uuid.UUID
	FullName     string
}

// BuildGradebook assembles one row per learner. A teacher's override always
// wins over the quiz result, and an item nobody has sat is left ungraded
// rather than counted as a zero.
func BuildGradebook(items []Item, learners []Learner, quizzes []QuizScore, overrides []Override) []Report {
	byOverride := make(map[key]Override, len(overrides))
	for _, o := range overrides {
		byOverride[key{o.ItemID, o.EnrollmentID}] = o
	}
	byQuiz := make(map[key]QuizScore, len(quizzes))
	for _, s := range quizzes {
		byQuiz[key{s.QuizID, s.EnrollmentID}] = s
	}

	reports := make([]Report, 0, len(learners))
	for _, learner := range learners {
		report := Report{
			EnrollmentID: learner.EnrollmentID, FullName: learner.FullName,
			Scores: make([]Score, 0, len(items)), Total: len(items),
		}

		var weighted, weight int64
		for _, item := range items {
			score := Score{ItemID: item.ID, Possible: item.Possible}

			if o, ok := byOverride[key{item.ID, learner.EnrollmentID}]; ok {
				points := o.Points
				score.Points, score.Overridden, score.Note = &points, true, o.Note
			} else if q, ok := byQuiz[key{item.QuizID, learner.EnrollmentID}]; ok && item.QuizID != uuid.Nil {
				points := q.Points
				score.Points = &points
			}

			if score.Points != nil {
				pct := Percent(*score.Points, item.Possible)
				score.Percent = &pct
				report.Graded++
				weighted += int64(pct) * int64(item.Weight)
				weight += int64(item.Weight)
			}
			report.Scores = append(report.Scores, score)
		}

		if weight > 0 {
			report.Percent = int((weighted*2 + weight) / (weight * 2))
		}
		reports = append(reports, report)
	}
	return reports
}

type key struct {
	item   uuid.UUID
	member uuid.UUID
}
