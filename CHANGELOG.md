# Changelog

All user-facing changes. Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added
- Attendance: schedule class sessions and call the register for a whole class in
  one go, marking learners present, late, absent or excused.
- Parents and guardians can be linked to a learner and are told the same day
  when that learner is marked absent.
- Learners see their own attendance record and rate. A late arrival still counts
  as attended, and authorised leave does not damage the rate.
- Notifications: learners are told when a payment is approved or rejected, when
  a quiz result is released and when their work has been marked.
- An in-app inbox with an unread count, per notification and mark-all-read.
- Messages go out by SMS as well, through a channel that can be pointed at any
  local gateway by configuration rather than code.
- A gateway that is down does not lose a message: delivery is retried with a
  growing wait and given up on only after five attempts.
- Assignments on any lesson, with a due date, a points value, an attachment
  limit and an optional late penalty.
- Learners save drafts as they work and hand in when ready; attachments reuse
  the same upload path as lesson media.
- Late work is flagged by the server clock, not the learner's device, and the
  penalty is applied once at marking so the learner sees a single mark.
- A deadline can be closed to late work entirely, though a draft can still be
  saved so nothing already typed is lost.
- Marked work cannot be resubmitted or marked twice, and marks appear in the
  gradebook alongside quiz results.
- Gradebook showing every learner against every graded item, with a weighted
  course percentage.
- Quizzes appear in the gradebook automatically and stay worth whatever their
  questions add up to.
- Teachers can add their own graded items, such as an oral exam, and give them
  more or less weight than a quiz.
- A teacher's score always wins over a computed one, and removing it restores
  the quiz result.
- An item nobody has sat yet is left blank rather than counted as a zero, so a
  course average is never wrong halfway through a term.
- Learners see their own grades; only staff see the class.
- Marking queue showing every attempt waiting on a teacher, who wrote it and how
  many answers are outstanding.
- Marking view with the learner's answer, the correct answer and the points
  available, plus written feedback per question.
- Results are released deliberately: an attempt cannot be released while any
  answer is still unmarked, and once released it cannot be changed.
- A mark can never exceed what a question is worth.
- Quizzes on any lesson: multiple choice, multiple answer, true or false, short
  answer and essay, with a time limit, an attempt limit and a pass mark.
- Choice questions are marked the moment a learner submits; written answers wait
  for a teacher and the attempt is held rather than reported as passed.
- Learners never receive the answer key. Correct options and explanations are
  withheld until the attempt is submitted.
- An interrupted attempt resumes where it left off instead of burning one of the
  learner's tries.
- Questions that could never be marked, such as a choice question with no
  correct option, are refused when they are written rather than when they are sat.
- Pay with bKash directly, without going through a card gateway.
- Returning from a payment now lands the payer on a readable page instead of
  raw JSON.
- Pay by card, bKash, Nagad or bank through SSLCommerz: the learner is sent to
  the gateway and enrolled automatically once the payment clears.
- Payments are confirmed with SSLCommerz directly rather than trusting what the
  callback says, so a forged notification cannot unlock a course.
- Repeated notifications from a gateway are recorded once and change nothing,
  which matters because gateways retry.
- Each tenant gets its own callback address, so one deployment can serve many
  institutions with separate gateway accounts.
- Sell a course: a learner starts an order and gets a payment reference plus
  the bank or wallet details to send money to.
- Pay by bank transfer or mobile wallet, upload the deposit slip, and have a
  member of staff approve it. Approval enrols the learner immediately.
- Staff review queue showing who paid, for what, with the slip attached.
- Tapping buy twice returns the order already in progress instead of creating a
  second one.
- A rejected payment lets the learner try again; an approved one cannot be
  reviewed, cancelled or re-submitted.
- Payment methods are pluggable, so bKash, SSLCommerz, Tap and Stripe fit
  behind the same flow.
- Upload files straight to storage: the API signs a one-time upload URL and the
  file goes from the browser to the bucket without passing through the server.
- Uploads are confirmed before a lesson can use them, so a half-finished upload
  never becomes a video nobody can play.
- File types are limited to video, audio, PDF, images and subtitles, and a size
  limit is enforced before the upload starts rather than after it finishes.
- Playback links expire and are issued per viewer.
- Storage is any S3-compatible bucket, self-hosted MinIO by default; without one
  configured the API still runs on embeds alone.
- Enrolment: learners join a free public course themselves, staff enrol anyone,
  and enrolling twice is harmless.
- Progress tracking with a resume point per lesson, a completion percentage per
  course, and automatic course completion on the last lesson.
- Progress reports merge safely across devices: a resume point only moves
  forward and a finished lesson never reverts, so a phone that was offline for
  a week cannot undo work done since.
- `GET /v1/courses/{id}/roster` shows every learner's completion in one query.
- `PATCH /v1/lessons/{id}` edits a lesson and publishes it.
- Requests that legitimately carry no body are no longer rejected.
- Media is pluggable: one provider interface with an embed adapter shipping
  first, so video costs nothing at launch. A transcoder drops in behind the
  same interface without changing anything above it.
- Paste a YouTube, Vimeo, Dailymotion or self-hosted video link onto a lesson;
  anything else is refused with a clear reason.
- Playback URLs are handed out per viewer, and embeds are sandboxed.
- Delivery is counted per tenant per day from day one, so bandwidth can be
  priced later without a backfill.
- Course catalog: courses with modules and lessons, draft/published/archived
  states, per-course text direction, price and visibility.
- `GET /v1/courses/{slug}` returns the full outline in display order.
- Drag-to-reorder writes a single row, using fractional positions.
- Titles in Arabic or Bengali get a generated web address instead of an empty
  one, and keep their own text direction.
- Authoring is limited to owners, admins and instructors; any member can read.
- Passwordless login by one-time code to a phone number or email address, with
  a first-login signup, rate limiting and a replay-proof code.
- Session tokens that can be revoked instantly, and `POST /v1/auth/logout`.
- `GET /v1/me` returns the signed-in user and the tenants they belong to.
- Tenant-scoped endpoints selected by the `X-Fajr-Tenant` header, with role
  checks: `GET /v1/tenant` and `GET /v1/tenant/members`.
- Notifications go through a pluggable channel; SMS and WhatsApp adapters drop
  in behind it. Development logs the code instead of sending it.
- Multi-tenant data model: tenants (institution, creator, corporate), global
  users, and per-tenant memberships with roles.
- Tenant isolation enforced by Postgres row-level security, not application
  code, with a test suite that proves cross-tenant reads and writes fail.
- Bidi-ready schema: per-tenant text direction and locale, ICU collation so
  Latin, Arabic and Bengali names sort correctly, trigram index for name search.
- `GET /readyz` reports database reachability.
- HTTP API skeleton with `GET /healthz`, JSON error responses, request ids on
  every response, and graceful shutdown.
