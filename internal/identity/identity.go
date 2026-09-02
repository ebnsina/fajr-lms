// Package identity handles OTP login, sessions and tenant membership checks.
package identity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"math/big"
	"net/netip"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/notify"
)

const (
	otpTTL        = 10 * time.Minute
	otpMaxPerHour = 5
	otpMaxAttempt = 5
	sessionTTL    = 30 * 24 * time.Hour
)

var (
	ErrInvalidDestination = errors.New("identity: destination must be a phone number or email")
	ErrRateLimited        = errors.New("identity: too many codes requested")
	ErrInvalidCode        = errors.New("identity: code is wrong or expired")
	ErrNoMembership       = errors.New("identity: user is not a member of this tenant")
	ErrInvalidSession     = errors.New("identity: session is invalid or expired")

	phoneRe = regexp.MustCompile(`^\+[1-9][0-9]{6,14}$`)
	emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s.]+\.[^@\s]{2,}$`)
)

type Service struct {
	store   *database.Store
	channel notify.Channel
}

func New(store *database.Store, channel notify.Channel) *Service {
	return &Service{store: store, channel: channel}
}

// Destination is a login identifier, normalized to exactly one of phone or email.
type Destination struct {
	Phone string
	Email string
}

func (d Destination) key() string {
	if d.Phone != "" {
		return d.Phone
	}
	return d.Email
}

// ParseDestination normalizes a raw identifier, rejecting anything ambiguous.
func ParseDestination(raw string) (Destination, error) {
	v := strings.TrimSpace(raw)
	switch {
	case phoneRe.MatchString(v):
		return Destination{Phone: v}, nil
	case emailRe.MatchString(v):
		return Destination{Email: strings.ToLower(v)}, nil
	default:
		return Destination{}, ErrInvalidDestination
	}
}

// RequestOTP issues a code and hands it to the notification channel.
func (s *Service) RequestOTP(ctx context.Context, dest Destination) error {
	q := s.store.Unscoped()

	recent, err := q.CountRecentOTPs(ctx, database.CountRecentOTPsParams{
		Destination: dest.key(), Purpose: database.OtpPurposeLogin, Lookback: hours(1),
	})
	if err != nil {
		return fmt.Errorf("count recent codes: %w", err)
	}
	if recent >= otpMaxPerHour {
		return ErrRateLimited
	}

	code, err := newCode()
	if err != nil {
		return err
	}
	if _, err := q.CreateOTPChallenge(ctx, database.CreateOTPChallengeParams{
		Destination: dest.key(),
		Purpose:     database.OtpPurposeLogin,
		CodeHash:    hashCode(dest.key(), code),
		TtlInterval: minutes(otpTTL),
	}); err != nil {
		return fmt.Errorf("store challenge: %w", err)
	}

	return s.channel.Send(ctx, notify.Message{
		To:      dest.key(),
		Purpose: "login_code",
		Body:    fmt.Sprintf("Your Fajr LMS code is %s. It expires in %d minutes.", code, int(otpTTL.Minutes())),
	})
}

// Session is an authenticated session plus the user it belongs to.
type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	FullName  string
	Token     string
	ExpiresAt time.Time
}

// Membership is one tenant a user may act in.
type Membership struct {
	TenantID uuid.UUID `json:"tenant_id"`
	Role     string    `json:"role"`
}

// VerifyOTP consumes a code and starts a session, creating the user on first login.
func (s *Service) VerifyOTP(ctx context.Context, dest Destination, code, fullName, userAgent string, ip *netip.Addr) (Session, []Membership, error) {
	q := s.store.Unscoped()

	challenge, err := q.LatestOTPChallenge(ctx, database.LatestOTPChallengeParams{
		Destination: dest.key(), Purpose: database.OtpPurposeLogin,
	})
	if database.IsNotFound(err) {
		return Session{}, nil, ErrInvalidCode
	}
	if err != nil {
		return Session{}, nil, fmt.Errorf("load challenge: %w", err)
	}

	if challenge.Attempts >= otpMaxAttempt || challenge.ExpiresAt.Time.Before(time.Now()) {
		return Session{}, nil, ErrInvalidCode
	}
	if subtle.ConstantTimeCompare(challenge.CodeHash, hashCode(dest.key(), code)) != 1 {
		if err := q.RecordOTPAttempt(ctx, challenge.ID); err != nil {
			return Session{}, nil, fmt.Errorf("record attempt: %w", err)
		}
		return Session{}, nil, ErrInvalidCode
	}
	if err := q.ConsumeOTPChallenge(ctx, challenge.ID); err != nil {
		return Session{}, nil, fmt.Errorf("consume challenge: %w", err)
	}

	user, err := q.FindUserForAuth(ctx, database.FindUserForAuthParams{Phone: dest.Phone, Email: dest.Email})
	if database.IsNotFound(err) {
		name := strings.TrimSpace(fullName)
		if name == "" {
			name = dest.key()
		}
		created, err := q.SignupUser(ctx, database.SignupUserParams{
			Phone: dest.Phone, Email: dest.Email, FullName: name,
		})
		if err != nil {
			return Session{}, nil, fmt.Errorf("create user: %w", err)
		}
		user = created
	} else if err != nil {
		return Session{}, nil, fmt.Errorf("find user: %w", err)
	}

	session, err := s.startSession(ctx, user.ID, userAgent, ip)
	if err != nil {
		return Session{}, nil, err
	}
	session.FullName = user.FullName

	members, err := s.Memberships(ctx, user.ID)
	if err != nil {
		return Session{}, nil, err
	}
	return session, members, nil
}

func (s *Service) startSession(ctx context.Context, userID uuid.UUID, userAgent string, ip *netip.Addr) (Session, error) {
	token, err := newToken()
	if err != nil {
		return Session{}, err
	}
	if len(userAgent) > 512 {
		userAgent = userAgent[:512]
	}

	row, err := s.store.Unscoped().CreateSession(ctx, database.CreateSessionParams{
		UserID: userID, TokenHash: hashToken(token), UserAgent: userAgent, Ip: ip, TtlInterval: minutes(sessionTTL),
	})
	if err != nil {
		return Session{}, fmt.Errorf("create session: %w", err)
	}
	return Session{ID: row.ID, UserID: userID, Token: token, ExpiresAt: row.ExpiresAt.Time}, nil
}

// Authenticate resolves a bearer token to a live session.
func (s *Service) Authenticate(ctx context.Context, token string) (Session, error) {
	if len(token) < 32 {
		return Session{}, ErrInvalidSession
	}
	row, err := s.store.Unscoped().GetSessionByToken(ctx, hashToken(token))
	if database.IsNotFound(err) {
		return Session{}, ErrInvalidSession
	}
	if err != nil {
		return Session{}, fmt.Errorf("load session: %w", err)
	}
	if err := s.store.Unscoped().TouchSession(ctx, row.SessionID); err != nil {
		return Session{}, fmt.Errorf("touch session: %w", err)
	}
	return Session{
		ID: row.SessionID, UserID: row.UserID, FullName: row.FullName,
		ExpiresAt: row.ExpiresAt.Time,
	}, nil
}

func (s *Service) Memberships(ctx context.Context, userID uuid.UUID) ([]Membership, error) {
	rows, err := s.store.Unscoped().ListUserMemberships(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list memberships: %w", err)
	}
	out := make([]Membership, 0, len(rows))
	for _, r := range rows {
		out = append(out, Membership{TenantID: r.TenantID, Role: string(r.Role)})
	}
	return out, nil
}

// ResolveTenant looks up a tenant by slug before any tenant scope exists.
func (s *Service) ResolveTenant(ctx context.Context, slug string) (database.Tenant, error) {
	tenant, err := s.store.Unscoped().ResolveTenant(ctx, slug)
	if database.IsNotFound(err) {
		return database.Tenant{}, ErrNoMembership
	}
	if err != nil {
		return database.Tenant{}, fmt.Errorf("resolve tenant: %w", err)
	}
	if tenant.Status != database.TenantStatusActive {
		return database.Tenant{}, ErrNoMembership
	}
	return tenant, nil
}

// MembershipIn returns the user's membership in one tenant, by slug.
func (s *Service) MembershipIn(ctx context.Context, userID uuid.UUID, slug string) (database.Tenant, Membership, error) {
	tenant, err := s.ResolveTenant(ctx, slug)
	if err != nil {
		return database.Tenant{}, Membership{}, err
	}
	members, err := s.Memberships(ctx, userID)
	if err != nil {
		return database.Tenant{}, Membership{}, err
	}
	for _, m := range members {
		if m.TenantID == tenant.ID {
			return tenant, m, nil
		}
	}
	return database.Tenant{}, Membership{}, ErrNoMembership
}

func (s *Service) Logout(ctx context.Context, sessionID uuid.UUID) error {
	if err := s.store.Unscoped().RevokeSession(ctx, sessionID); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

// newCode returns a uniformly random six-digit code.
func newCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", fmt.Errorf("generate code: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func newToken() (string, error) {
	return rand.Text() + rand.Text(), nil
}

// hashCode binds a code to its destination so a leak elsewhere cannot replay it.
func hashCode(destination, code string) []byte {
	sum := sha256.Sum256([]byte(destination + "\x00" + code))
	return sum[:]
}

func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

func minutes(d time.Duration) pgtype.Interval {
	return pgtype.Interval{Microseconds: d.Microseconds(), Valid: true}
}

func hours(n int) pgtype.Interval {
	return minutes(time.Duration(n) * time.Hour)
}
