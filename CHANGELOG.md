# Changelog

All user-facing changes. Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Changed

- The Fajr LMS mark is simpler and has its own colours: a sky graded from deep
  blue to cyan, a soft amber sun rising out of the bottom-right corner, and the
  letter fā' set in it. The tab icon and the home-screen icon match.

### Added
- Anyone can see the product with work in it: **Get a demo** asks who you are
  and what you run, then opens a demo school of that kind — a madrasah, a
  school, a coaching centre, a teacher on their own, or a company — already
  holding courses, learners part way through, marks, a register taken and a
  certificate somebody earned. The demo schools are shared and read-only:
  every screen opens, nothing can be changed.
- The landing page now says what the product actually does: certificates you
  design, classes and sections, points and badges, discussion in the course,
  SCORM packages, and signing in with the school's own accounts.
- A school can lay out its own certificate: drag each line — the learner's
  name, the course, the date, the serial, words of your own — where it belongs,
  set its size, colour and weight, and print it on paper of your own. The public
  verification page draws the same layout, and a school that draws nothing keeps
  the design we ship.
- A mark for Fajr LMS: the dawn the name means, a sun rising behind a twin ridge
  in a squircle tile, drifting slowly like the Fajr AI orb and holding still
  under reduced motion. It sits above the school switcher in the sidebar, with
  "A product of Fajr Labs" quietly beneath the account menu, and it is the tab
  and home-screen icon too.
- Fajr LMS has a front door of its own: a landing page, pricing, and a form
  that opens a school in about a minute without anybody talking to us.
- A features section saying what each part of the product actually does, from
  lessons and quizzes through to the website builder.
- A section about the AI work being built — a draft quiz, subtitles, a first
  pass at grading — lit by a slowly drifting orb. It says plainly that none of
  it is on yet, and that it drafts rather than decides.
- Every school gets a public website with a page builder: pages are built from
  validated plain-text sections, published one at a time, with the menu built
  from the pages themselves.
- Eight website templates — a school, a college, a madrasah and a university,
  each written twice, once for Bangladesh and once for the Gulf, in the
  language it teaches in. Picking one writes its pages as drafts and dresses
  the site to match.
- Three looks for that website: Gulf reads right to left with Arabic set
  larger, Bengal sets Bengali, and Plain keeps the product palette.
- Two more page sections: numbers, and a notice board.
- Build a course from the app rather than the API: create it, add sections and
  lessons, paste a video or audio link to attach it, reorder lessons, and
  publish a lesson or the whole course when it is ready.
- Set a quiz from the app: write questions of any kind, tick the right choices,
  and see the answer key laid out. Set an assignment the same way, and change
  its due date or late penalty afterwards.
- Live classes have a page: staff paste the Google Meet or Zoom link and attach
  the recording afterwards, and anyone enrolled joins from there when it opens.
- Certificates have a page: what you have been awarded, the serial anyone can
  check, and a claim on a course you have finished.
- Course packages built elsewhere. A SCORM 1.2 zip from a publisher uploads
  onto a lesson, plays inside it, and what it reports comes back as progress and
  as a mark a teacher can change. Packages that could not be served safely are
  refused on the way in.
- Signing in with a school account. A school points Fajr at its Google
  Workspace, Microsoft Entra or any OpenID Connect provider, and people sign in
  with the account they already have instead of waiting for a code. The school
  chooses which email domains may sign in, what role a new person joins as, and
  whether anybody the provider vouches for may join at all.
- A course can be sold in instalments. The school sets how many payments and
  how far apart; a learner picks paying in full or in parts at checkout, is
  enrolled from the first payment, and can see what is left to pay. The office
  gets a list of who is paying a course off over time.
- The app has a checkout at last: a learner can buy a course from the web, see
  the bank details and reference, and tell the office what they paid.
- Money is now formatted and read by the currency's own subunit, so a dinar's
  three decimal places and a yen's none are right instead of assumed to be two.
- Topics, so a catalog of forty courses is navigable, and collections: a path
  is several courses worked through in order, a bundle is several bought
  together. A path carries no price, because it is worked through rather than
  sold.
- Points and badges, off unless a school switches them on. Finishing a lesson,
  passing a quiz and finishing a course each pay once, however many times they
  are repeated; a school sets its own badges and the mark each is earned at.
  The leaderboard runs on the last month, because a board that never resets
  belongs to whoever joined first.
- Every course has a discussion. Anybody on the course can ask and answer;
  teachers pin what matters and close what is finished. A person can fix their
  own words and nobody else's, and a removed post keeps its place in the thread
  without keeping what it said.
- Notices to guardians. A school tells one section, one class or everybody
  something — a closure, a fee due, an absence — and it reaches guardians in
  their inbox and by text message where the school has an SMS gateway. A
  guardian with two children in the same class hears once, not twice.
- A guardian signs in and sees their own children: where each one sits this
  year and what the school records for them. Only their own, checked on every
  read.
- Hifz. A teacher keeps the daily record a madrasah already keeps — sabaq,
  sabqi and manzil, with the range heard, how it went and how many slips — and
  a student or their guardian can see how much has been committed, counted in
  ayahs. Ranges are checked against the real length of each surah, so a slip of
  the finger cannot record an ayah that does not exist.
- The school's own shape: the academic year being taught in and the terms
  inside it, and the classes, sections and subjects everything else is arranged
  by. A school names its own ladder — Class Six, Ibtidaiyyah, HSC first year —
  because no two schools agree on one. Students are placed in a section for the
  year, with a roll number that is theirs alone within it.
- Fajr AI. It drafts quiz questions from a lesson a teacher has written, and
  writes a first draft of a course summary or a lesson's text, streaming as it
  goes. Nothing is saved until the teacher accepts it, and a drafted question
  that could not be graded is dropped before it is shown. The provider is a
  setting — Anthropic, OpenAI, Gemini, or Ollama on the school's own machine —
  and with none configured every AI surface says so plainly instead of failing.
- An answer written while the connection is gone is kept on the device and sent
  when it returns. The paper says how many are waiting.
- A quiz being sat now shows the paper that attempt was served rather than the
  quiz's own list, which is what a drawn quiz needs.
- The app installs to a phone's home screen and survives a dropped connection:
  the shell is cached, and a page that cannot load says so instead of showing a
  browser error. Signed-in pages are never cached, so a shared phone does not
  show one learner's work to the next.
- The favicon existed in the markup but not in the app; it does now.
- A school's site is now served on its own domain: the Website page takes the
  domain, shows the record to publish, and checks it. Once the record is there,
  the site answers at the root of that domain.
- Error messages were showing in the brand green, which read as good news. They
  are red now, everywhere.
- A certificate's verification link now opens on the school's own site instead
  of the API address, so the link on a printed certificate works.
- Refunds. The payments page now lists the money taken as well as the money
  waiting, and staff can hand back all of a payment or part of it with a reason.
  A full refund closes the enrolment it paid for unless you say to keep access.
  The money itself still moves at the bank or the gateway; this is the record.
- A quiz can draw a few questions at random from a larger pool, and the shuffle
  setting now really shuffles. Each attempt keeps the paper it was handed, so a
  learner who reconnects gets the same questions back and is marked out of them.
- Discount codes: a percentage or a fixed amount, tied to one course or open to
  all, capped by uses or by dates. A code is priced onto the order, so what the
  learner is told to pay is what the record says, and a use is only counted once
  the money is actually in.
- Put a school's public site on its own domain: name it, publish one DNS record
  to prove the domain is yours, and the site answers there.
- Upload a video, audio file or PDF onto a lesson, with a progress bar, instead
  of only pasting a link. The file goes straight from the browser to storage.
- Award a certificate to anybody on a course, see everything awarded on it, and
  revoke one with a reason that appears on the public page.
- Invite somebody to your school by phone number or email address, set what
  they may do, change it later, or remove them. The last owner cannot be
  removed or demoted, so a school is never left without one.
- Edit a course after making it: its title, summary, fee and who may see it.
- Rename, reorder and delete the sections of a course, not only its lessons.
- Take a roll call: pick a class, mark everyone present, late, absent or
  excused, and marking somebody absent tells them and anyone listed as their
  guardian.
- Review a bank transfer or wallet payment: the learner's transaction id, their
  note and the deposit slip on one card, with approving enrolling them straight
  away.
- The home page is an overview rather than a list of courses: the date and a
  running clock beside the greeting, stat cards and recent activity chosen for
  what you do in the school, a chart of the work waiting to be graded, and a
  ring showing how far through your lessons you are.
- A certificate looks like a certificate: the public page anyone can check is
  set as a real document, with the recipient's name large, a ruled border, the
  issuer, the date and the serial, printable as it stands and stamped REVOKED
  when it no longer holds. Your own certificates are listed as scaled-down
  copies of that same page rather than a table of fields.

### Changed
- The product is written as Fajr LMS throughout, and the wording is US English:
  Marking is now Grading, a mark is now a grade.
- The brand color is emerald green, on every screen and in both themes.
- The landing pages are set in Clash Display and Excon, with motion that is
  skipped whenever the reader has asked for less of it.
- Dropdowns are our own design rather than the browser's, and turn into a
  searchable list once there is enough to scroll through.
- The button that finishes a form now sits at the end of it rather than the
  start.
- A trail deeper than three steps folds its middle into a button that lists the
  steps in between, so the header stays readable on a lesson page.
- The account menu no longer wears the chevron a dropdown wears. Only a chooser
  — the school switcher, a select — carries one.

### Fixed
- Opening a dropdown no longer shifts the page under it. Focusing inside the
  open panel used to scroll whatever was around it, sometimes leaving a table
  stuck part way down with no way to scroll it back.
- Switching school now shows the new school straight away. It used to set the
  choice and leave the old school's courses, grades and notifications on screen
  until the page was reloaded by hand.

### Added
- A proper application shell: the menu stays put while the page scrolls, your
  school and a switcher sit at the top of it, and your account sits at the
  bottom with settings and sign out.
- Breadcrumbs, a notification bell with the recent few and an unread count, and
  a Settings page holding the appearance choice and your details.

- Mark an assignment: the work, the brief and any attached photos on one page,
  with the mark and a comment returned to the learner.
- Late work shows what the learner will actually receive once the penalty comes
  off, before the mark is saved.

- Gradebook grid: every learner against every graded item, with the course
  percentage worked out for you. Type in a box to enter or correct a score, and
  empty it to go back to the marked one.
- Add your own graded items, such as an oral exam, and give them their own weight.

- Open an attempt from the marking queue and mark it: the learner's answer, the
  correct answer and the points available are on one row, and written answers
  get a score and a comment.
- A result can only be released once every written answer is marked, and the
  learner is told the moment it is.

- Hand in an assignment: write an answer, attach photos of handwritten work or a
  PDF, save a draft while you go, and hand in when you are ready.
- The page says plainly what happens if you are late, and whether late work is
  taken at all.
- Once marked, the mark and the teacher's comment appear on the same page.

- Sit a quiz: multiple choice, multiple answer, true or false and written
  answers, with a countdown when there is a time limit.
- Answers save as you go, so a dropped connection loses nothing, and reopening
  the page brings back what you had already chosen.
- After handing in, you see what each question scored and why. Written answers
  say plainly that they are with a teacher and the score may still go up.

- A proper dashboard: a grouped menu down the side for learning, teaching and
  administration, with the work sitting on its own panel.
- New pages behind it: all courses, my grades, notifications, marking,
  submissions, members and payments to review.

- Icons throughout: each lesson shows what kind it is at a glance, and buttons,
  links and empty states carry a mark rather than words alone.

- The six digit code is entered in six boxes, one per digit, with the cursor
  showing which one is next. Pasting a code and the phone's own autofill both
  still work.

- Sign in and choosing a school are now full pages, with the form on one side and
  a plain statement of what Fajr does on the other.

### Changed
- The quiz timer is now a proper clock face, and turns amber then red as the
  time runs down.
- The page frame stays put while the content inside it scrolls.

- New brand color: a deep indigo, with a warm amber used only for progress.

- Every button and input is the same height, so a stack of them lines up.

- Primary buttons are forest green with white text, in both appearances.
- Typeface is now Cabin, with Geist Mono for codes, numbers and placeholders.
- A light and a dark appearance, and a control to choose between them or follow
  the device. The choice is remembered and applied before the page paints, so
  there is no flash of the wrong one.

### Changed
- New typeface and a quieter surface treatment: layered whites in light, charcoal
  rather than black in dark, hairline borders in place of shadows, and softer
  corners on cards and controls.
- Course pages: see the whole outline, how far you have got, and jump straight
  back to where you stopped.
- Lesson pages with video, written content and a finish button, plus next and
  previous that point the right way in a right-to-left interface.
- Video downloads nothing until you press play, so opening a lesson on a metered
  connection costs almost nothing.
- Web app: sign in with a one-time code, choose which school you are working in,
  and see your courses and unread messages.
- The interface renders right to left for an Arabic school from the first paint,
  with no flicker, and names in any script display correctly wherever they appear.
- Signing in keeps you signed in on that device for thirty days, and signing out
  works even if the server cannot be reached.
- Live classes: paste a Google Meet, Zoom, Teams, Jitsi or Whereby link onto a
  class and enrolled learners can join it.
- The link opens fifteen minutes before the class and is not handed out days in
  advance; teachers can open the room whenever they need to.
- Teachers can keep a separate host link, and attach the recording to the class
  afterwards.
- Certificates for finished courses, each with a number anyone can check at a
  public address without signing in.
- The verification page prints cleanly and shows Arabic and Bengali names
  correctly, so a certificate can be handed to an employer or a board.
- Certificates can be revoked, and the public page then says so plainly.
- A certificate keeps the names it was issued with, so renaming a course later
  does not rewrite what was awarded.
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
  member of staff approve it. Approval enrolls the learner immediately.
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
- Enrollment: learners join a free public course themselves, staff enroll anyone,
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
