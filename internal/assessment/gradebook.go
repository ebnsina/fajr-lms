package assessment

import "github.com/google/uuid"

// Item is one graded thing in a course. Exactly one source id is set.
type Item struct {
	ID           uuid.UUID
	QuizID       uuid.UUID
	AssignmentID uuid.UUID
	Title        string
	Category     string
	Possible     int32
	Weight       int32
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

// SourceScore is a score computed elsewhere: a quiz attempt or a marked
// submission. The source id matches the item that owns it.
type SourceScore struct {
	SourceID     uuid.UUID
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
func BuildGradebook(items []Item, learners []Learner, computed []SourceScore, overrides []Override) []Report {
	byOverride := make(map[key]Override, len(overrides))
	for _, o := range overrides {
		byOverride[key{o.ItemID, o.EnrollmentID}] = o
	}
	bySource := make(map[key]SourceScore, len(computed))
	for _, s := range computed {
		bySource[key{s.SourceID, s.EnrollmentID}] = s
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
			} else if source := item.sourceID(); source != uuid.Nil {
				if computed, ok := bySource[key{source, learner.EnrollmentID}]; ok {
					points := computed.Points
					score.Points = &points
				}
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

// sourceID is whichever of the two source columns this item carries.
func (i Item) sourceID() uuid.UUID {
	if i.QuizID != uuid.Nil {
		return i.QuizID
	}
	return i.AssignmentID
}

type key struct {
	item   uuid.UUID
	member uuid.UUID
}

// LatePenalty removes a percentage from a mark, never below zero. It is applied
// once, when the work is graded, so the learner sees one number.
func LatePenalty(points int32, penaltyPercent int) int32 {
	if penaltyPercent <= 0 || points <= 0 {
		return points
	}
	if penaltyPercent >= 100 {
		return 0
	}
	kept := int64(points) * int64(100-penaltyPercent)
	return int32(kept / 100)
}
