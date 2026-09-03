package config

import (
	"strings"
	"testing"
)

func TestPublicURLMustBeRealInProduction(t *testing.T) {
	t.Setenv("FAJR_DATABASE_URL", "postgres://localhost/fajr")

	t.Run("development is left alone", func(t *testing.T) {
		c, err := Load()
		if err != nil {
			t.Fatalf("load: %v", err)
		}
		if c.PublicURL == "" {
			t.Error("development should still get a usable address")
		}
	})

	t.Run("production refuses the localhost default", func(t *testing.T) {
		t.Setenv("FAJR_ENV", "production")
		if _, err := Load(); err == nil || !strings.Contains(err.Error(), "FAJR_PUBLIC_URL") {
			t.Fatalf("want a complaint about FAJR_PUBLIC_URL, got %v", err)
		}
	})

	t.Run("production accepts a real address", func(t *testing.T) {
		t.Setenv("FAJR_ENV", "production")
		t.Setenv("FAJR_PUBLIC_URL", "https://greenfield.example/")
		c, err := Load()
		if err != nil {
			t.Fatalf("load: %v", err)
		}
		if c.PublicURL != "https://greenfield.example" {
			t.Errorf("the trailing slash should go: %q", c.PublicURL)
		}
	})
}
