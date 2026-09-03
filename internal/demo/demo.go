// Package demo fills a school with a term's worth of believable work, so a
// visitor sees the product with something in it rather than empty screens.
package demo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// A Kind is one shape of customer, with the school that shows it best.
type Kind struct {
	Slug        string
	Name        string
	Label       string
	Tenant      database.TenantKind
	Institution database.InstitutionKind
	Dir         database.TextDir
	Locale      string
	Currency    string
	Courses     []Course
	Learners    []string
	Attends     bool
}

type Course struct {
	Title      string
	Summary    string
	PriceMinor int64
	Modules    []Module
	Quiz       Quiz
}

type Module struct {
	Title   string
	Lessons []Lesson
}

type Lesson struct {
	Title string
	Kind  database.LessonKind
	Body  string
}

type Quiz struct {
	Title     string
	Questions []Question
}

type Question struct {
	Prompt  string
	Options []string
	Correct int
}

// Kinds are ordered as the demo form offers them.
var Kinds = []Kind{madrasah, school, coaching, creator, corporate}

// Find returns the demo school for what somebody says they run.
func Find(slug string) (Kind, bool) {
	for _, kind := range Kinds {
		if kind.Slug == strings.ToLower(strings.TrimSpace(slug)) {
			return kind, true
		}
	}
	return Kind{}, false
}

// Seed writes the whole school. It runs once, when the demo tenant is first
// provisioned, and every write is scoped to that tenant.
func Seed(ctx context.Context, store *database.Store, tenant database.Tenant,
	kind Kind, serial func() string) error {

	learners, err := learnerAccounts(ctx, store, tenant, kind)
	if err != nil {
		return err
	}

	return store.InTenant(ctx, tenant.ID, func(q *database.Queries) error {
		for index, course := range kind.Courses {
			if err := seedCourse(ctx, q, tenant, kind, course, index, learners, serial); err != nil {
				return err
			}
		}
		return nil
	})
}

// learnerAccounts creates the people first: users are global, memberships are
// not, so the two cannot share one scoped transaction.
func learnerAccounts(ctx context.Context, store *database.Store, tenant database.Tenant,
	kind Kind) ([]database.User, error) {

	unscoped := store.Unscoped()
	people := make([]database.User, 0, len(kind.Learners))
	for index, name := range kind.Learners {
		email := fmt.Sprintf("%s.%d@demo.fajr.school", tenant.Slug, index+1)
		user, err := unscoped.SignupUser(ctx, database.SignupUserParams{
			Email: email, FullName: name,
		})
		if err != nil {
			return nil, fmt.Errorf("demo learner: %w", err)
		}
		people = append(people, user)
	}

	err := store.InTenant(ctx, tenant.ID, func(q *database.Queries) error {
		for index, user := range people {
			// One of them teaches, so the roster is not all students.
			role := database.MemberRoleStudent
			if index == 0 {
				role = database.MemberRoleInstructor
			}
			if _, err := q.CreateMembership(ctx, database.CreateMembershipParams{
				TenantID: tenant.ID, UserID: user.ID, Role: role,
			}); err != nil {
				return fmt.Errorf("demo membership: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return people, nil
}

func seedCourse(ctx context.Context, q *database.Queries, tenant database.Tenant, kind Kind,
	course Course, index int, people []database.User, serial func() string) error {

	teacher := uuid.NullUUID{}
	if len(people) > 0 {
		teacher = uuid.NullUUID{UUID: people[0].ID, Valid: true}
	}

	made, err := q.CreateCourse(ctx, database.CreateCourseParams{
		TenantID: tenant.ID, Slug: slugify(course.Title), Title: course.Title,
		Summary: course.Summary, Dir: kind.Dir, Visibility: database.CourseVisibilityPublic,
		PriceMinor: course.PriceMinor, Currency: kind.Currency, CreatedBy: teacher,
	})
	if err != nil {
		return fmt.Errorf("demo course: %w", err)
	}
	if _, err := q.SetCourseStatus(ctx, database.SetCourseStatusParams{
		Status: database.PublishStatusPublished, ID: made.ID,
	}); err != nil {
		return fmt.Errorf("publish demo course: %w", err)
	}

	lessons, err := seedLessons(ctx, q, tenant, kind, made.ID, course)
	if err != nil {
		return err
	}

	// Only the first course is graded and certified; the rest are there to make
	// the catalogue look like a catalogue.
	students := people
	if len(students) > 1 {
		students = students[1:]
	}
	if err := seedRoster(ctx, q, tenant, made, kind, lessons, students, index, serial); err != nil {
		return err
	}
	return nil
}

func seedLessons(ctx context.Context, q *database.Queries, tenant database.Tenant, kind Kind,
	courseID uuid.UUID, course Course) ([]uuid.UUID, error) {

	ids := make([]uuid.UUID, 0, 8)
	for _, module := range course.Modules {
		made, err := q.CreateModule(ctx, database.CreateModuleParams{
			TenantID: tenant.ID, CourseID: courseID, Title: module.Title,
		})
		if err != nil {
			return nil, fmt.Errorf("demo module: %w", err)
		}
		for _, lesson := range module.Lessons {
			one, err := q.CreateLesson(ctx, database.CreateLessonParams{
				TenantID: tenant.ID, ModuleID: made.ID, Title: lesson.Title,
				Kind: lesson.Kind, Body: lesson.Body, Dir: kind.Dir,
				DurationS: 480, IsPreview: len(ids) == 0,
			})
			if err != nil {
				return nil, fmt.Errorf("demo lesson: %w", err)
			}
			ids = append(ids, one.ID)
		}
	}

	if len(course.Quiz.Questions) > 0 && len(ids) > 0 {
		if err := seedQuiz(ctx, q, tenant, kind, ids[len(ids)-1], course.Quiz); err != nil {
			return nil, err
		}
	}
	return ids, nil
}

func seedQuiz(ctx context.Context, q *database.Queries, tenant database.Tenant, kind Kind,
	lessonID uuid.UUID, quiz Quiz) error {

	made, err := q.CreateQuiz(ctx, database.CreateQuizParams{
		TenantID: tenant.ID, LessonID: lessonID, Title: quiz.Title,
		Instructions: "Answer every question. You may try twice.", Dir: kind.Dir,
		TimeLimitS: 900, MaxAttempts: 2, PassPercent: 60, Shuffle: true, RevealAnswers: true,
	})
	if err != nil {
		return fmt.Errorf("demo quiz: %w", err)
	}

	for _, question := range quiz.Questions {
		asked, err := q.CreateQuestion(ctx, database.CreateQuestionParams{
			TenantID: tenant.ID, QuizID: made.ID, Kind: database.QuestionKindMcqSingle,
			Prompt: question.Prompt, Dir: kind.Dir, Points: 1,
		})
		if err != nil {
			return fmt.Errorf("demo question: %w", err)
		}
		for at, label := range question.Options {
			if _, err := q.CreateOption(ctx, database.CreateOptionParams{
				TenantID: tenant.ID, QuestionID: asked.ID, Label: label,
				IsCorrect: at == question.Correct,
			}); err != nil {
				return fmt.Errorf("demo option: %w", err)
			}
		}
	}
	return nil
}

// seedRoster enrolls the learners and leaves the traces a real term leaves:
// progress part way through, a mark in the gradebook, a class that was taken,
// and a certificate for whoever finished.
func seedRoster(ctx context.Context, q *database.Queries, tenant database.Tenant,
	course database.Course, kind Kind, lessons []uuid.UUID, people []database.User,
	index int, serial func() string) error {

	item, err := q.CreateGradeItem(ctx, database.CreateGradeItemParams{
		TenantID: tenant.ID, CourseID: course.ID, Source: database.GradeSourceManual,
		Title: "Term mark", Category: "Term", PointsPossible: 100, Weight: 100,
	})
	if err != nil {
		return fmt.Errorf("demo grade item: %w", err)
	}

	var session database.ClassSession
	if kind.Attends {
		when := time.Now().Add(-48 * time.Hour)
		session, err = q.CreateClassSession(ctx, database.CreateClassSessionParams{
			TenantID: tenant.ID, CourseID: course.ID, Title: "Tuesday, first period",
			Location: "Room 2", StartsAt: stamp(when), EndsAt: stamp(when.Add(time.Hour)),
		})
		if err != nil {
			return fmt.Errorf("demo class: %w", err)
		}
	}

	for at, person := range people {
		enrolled, err := q.EnrollUser(ctx, database.EnrollUserParams{
			TenantID: tenant.ID, CourseID: course.ID, UserID: person.ID,
			Source: database.EnrollmentSourceStaff,
		})
		if err != nil {
			return fmt.Errorf("demo enrollment: %w", err)
		}

		// Each learner is a little further along than the one before.
		done := (at + 1) * len(lessons) / (len(people) + 1)
		for _, lesson := range lessons[:done] {
			if _, err := q.RecordProgress(ctx, database.RecordProgressParams{
				TenantID: tenant.ID, EnrollmentID: enrolled.ID, LessonID: lesson,
				Completed: true, PositionS: 480,
			}); err != nil {
				return fmt.Errorf("demo progress: %w", err)
			}
		}

		mark := int32(58 + (at*7)%40)
		if _, err := q.SetGradeOverride(ctx, database.SetGradeOverrideParams{
			TenantID: tenant.ID, GradeItemID: item.ID, EnrollmentID: enrolled.ID,
			Points: mark, Note: "Entered from the mark sheet.",
		}); err != nil {
			return fmt.Errorf("demo grade: %w", err)
		}

		if kind.Attends {
			status := database.AttendanceStatusPresent
			if at%4 == 3 {
				status = database.AttendanceStatusAbsent
			}
			if _, err := q.MarkAttendance(ctx, database.MarkAttendanceParams{
				TenantID: tenant.ID, SessionID: session.ID, EnrollmentID: enrolled.ID,
				Status: status,
			}); err != nil {
				return fmt.Errorf("demo attendance: %w", err)
			}
		}

		// The first course has somebody who finished it, so the certificate is
		// not a screen nobody can reach.
		if index == 0 && at == 0 {
			if _, err := q.CompleteEnrollment(ctx, enrolled.ID); err != nil {
				return fmt.Errorf("demo completion: %w", err)
			}
			percent := int16(mark)
			if _, err := q.IssueCertificate(ctx, database.IssueCertificateParams{
				TenantID: tenant.ID, CourseID: course.ID, EnrollmentID: enrolled.ID,
				UserID: person.ID, Serial: serial(), RecipientName: person.FullName,
				CourseTitle: course.Title, IssuerName: tenant.Name, GradePercent: &percent,
			}); err != nil {
				return fmt.Errorf("demo certificate: %w", err)
			}
		}
	}
	return nil
}

func stamp(at time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: at, Valid: true}
}

// slugify is the demo's own, because seeded titles are known and plain.
func slugify(title string) string {
	out := make([]rune, 0, len(title))
	for _, r := range strings.ToLower(title) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			out = append(out, r)
		case len(out) > 0 && out[len(out)-1] != '-':
			out = append(out, '-')
		}
	}
	return strings.Trim(string(out), "-")
}
