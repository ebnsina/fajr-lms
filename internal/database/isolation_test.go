package database_test

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"testing"

	"github.com/google/uuid"

	"github.com/ebnsina/fajr-lms/internal/database"
)

// newStore skips the suite when no database is configured, rather than failing.
func newStore(t *testing.T) *database.Store {
	t.Helper()
	url := os.Getenv("FAJR_DATABASE_URL")
	if url == "" {
		t.Skip("FAJR_DATABASE_URL not set")
	}
	store, err := database.Open(context.Background(), url)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(store.Close)
	return store
}

func seedTenant(t *testing.T, store *database.Store, slug, phone string) (uuid.UUID, uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	q := store.Unscoped()

	tenant, err := q.ProvisionTenant(ctx, database.ProvisionTenantParams{
		Slug: slug, Name: slug, Kind: database.TenantKindInstitution,
		DefaultDir: database.TextDirAuto, Locale: "en", Currency: "BDT",
	})
	if err != nil {
		t.Fatalf("create tenant %s: %v", slug, err)
	}

	user, err := q.SignupUser(ctx, database.SignupUserParams{
		Phone: phone, FullName: "Member of " + slug,
	})
	if err != nil {
		t.Fatalf("create user for %s: %v", slug, err)
	}

	err = store.InTenant(ctx, tenant.ID, func(q *database.Queries) error {
		_, err := q.CreateMembership(ctx, database.CreateMembershipParams{
			TenantID: tenant.ID, UserID: user.ID, Role: database.MemberRoleOwner,
		})
		return err
	})
	if err != nil {
		t.Fatalf("create membership for %s: %v", slug, err)
	}
	return tenant.ID, user.ID
}

// TestTenantIsolation is the load-bearing test: one missing filter ends the company.
func TestTenantIsolation(t *testing.T) {
	store := newStore(t)
	ctx := context.Background()

	suffix := fmt.Sprintf("%09d", rand.IntN(1_000_000_000))
	tenantA, userA := seedTenant(t, store, "alpha-"+suffix, "+8801"+suffix)
	tenantB, _ := seedTenant(t, store, "beta-"+suffix, "+8802"+suffix)

	t.Run("sees only its own members", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			id   uuid.UUID
		}{{"A", tenantA}, {"B", tenantB}} {
			err := store.InTenant(ctx, tc.id, func(q *database.Queries) error {
				n, err := q.CountTenantMembers(ctx)
				if err != nil {
					return err
				}
				if n != 1 {
					t.Errorf("tenant %s sees %d members, want 1", tc.name, n)
				}
				return nil
			})
			if err != nil {
				t.Fatalf("count members for %s: %v", tc.name, err)
			}
		}
	})

	t.Run("cannot read another tenant's user", func(t *testing.T) {
		err := store.InTenant(ctx, tenantB, func(q *database.Queries) error {
			_, err := q.GetUser(ctx, userA)
			return err
		})
		if !database.IsNotFound(err) {
			t.Fatalf("tenant B read tenant A's user: got %v, want no rows", err)
		}
	})

	t.Run("cannot read another tenant's record", func(t *testing.T) {
		err := store.InTenant(ctx, tenantB, func(q *database.Queries) error {
			_, err := q.GetTenant(ctx, tenantA)
			return err
		})
		if !database.IsNotFound(err) {
			t.Fatalf("tenant B read tenant A: got %v, want no rows", err)
		}
	})

	t.Run("cannot write into another tenant", func(t *testing.T) {
		err := store.InTenant(ctx, tenantB, func(q *database.Queries) error {
			_, err := q.CreateMembership(ctx, database.CreateMembershipParams{
				TenantID: tenantA, UserID: userA, Role: database.MemberRoleStudent,
			})
			return err
		})
		if err == nil {
			t.Fatal("tenant B wrote a membership into tenant A")
		}
	})

	t.Run("unscoped queries see nothing", func(t *testing.T) {
		n, err := store.Unscoped().CountTenantMembers(ctx)
		if err != nil {
			t.Fatalf("unscoped count: %v", err)
		}
		if n != 0 {
			t.Errorf("unscoped query returned %d members, want 0", n)
		}
	})

	t.Run("rejects an empty tenant id", func(t *testing.T) {
		err := store.InTenant(ctx, uuid.Nil, func(*database.Queries) error { return nil })
		if err != database.ErrNoTenant {
			t.Fatalf("got %v, want ErrNoTenant", err)
		}
	})
}
