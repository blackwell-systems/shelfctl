package config_test

import (
	"strings"
	"testing"

	"github.com/blackwell-systems/shelfctl/internal/config"
)

func TestShelfByName_Found(t *testing.T) {
	cfg := &config.Config{
		Shelves: []config.ShelfConfig{
			{Name: "programming", Repo: "shelf-prog"},
			{Name: "history", Repo: "shelf-hist"},
		},
	}
	s := cfg.ShelfByName("history")
	if s == nil {
		t.Fatal("ShelfByName returned nil for existing shelf")
	}
	if s.Repo != "shelf-hist" {
		t.Errorf("Repo = %q, want %q", s.Repo, "shelf-hist")
	}
}

func TestShelfByName_NotFound(t *testing.T) {
	cfg := &config.Config{
		Shelves: []config.ShelfConfig{
			{Name: "programming"},
		},
	}
	if cfg.ShelfByName("nope") != nil {
		t.Error("ShelfByName should return nil for missing shelf")
	}
}

func TestShelfByName_Empty(t *testing.T) {
	cfg := &config.Config{}
	if cfg.ShelfByName("any") != nil {
		t.Error("ShelfByName should return nil with no shelves")
	}
}

func TestEffectiveOwner_ShelfOverride(t *testing.T) {
	s := config.ShelfConfig{Owner: "alice"}
	if got := s.EffectiveOwner("bob"); got != "alice" {
		t.Errorf("EffectiveOwner = %q, want %q", got, "alice")
	}
}

func TestEffectiveOwner_GlobalFallback(t *testing.T) {
	s := config.ShelfConfig{}
	if got := s.EffectiveOwner("bob"); got != "bob" {
		t.Errorf("EffectiveOwner = %q, want %q", got, "bob")
	}
}

func TestEffectiveRelease_ShelfDefault(t *testing.T) {
	s := config.ShelfConfig{DefaultRelease: "v2024"}
	if got := s.EffectiveRelease("library"); got != "v2024" {
		t.Errorf("EffectiveRelease = %q, want %q", got, "v2024")
	}
}

func TestEffectiveRelease_GlobalDefault(t *testing.T) {
	s := config.ShelfConfig{}
	if got := s.EffectiveRelease("library"); got != "library" {
		t.Errorf("EffectiveRelease = %q, want %q", got, "library")
	}
}

func TestEffectiveRelease_Hardcoded(t *testing.T) {
	s := config.ShelfConfig{}
	if got := s.EffectiveRelease(""); got != "library" {
		t.Errorf("EffectiveRelease = %q, want %q", got, "library")
	}
}

func TestEffectiveCatalogPath_Custom(t *testing.T) {
	s := config.ShelfConfig{CatalogPath: "books.yml"}
	if got := s.EffectiveCatalogPath(); got != "books.yml" {
		t.Errorf("EffectiveCatalogPath = %q, want %q", got, "books.yml")
	}
}

func TestEffectiveCatalogPath_Default(t *testing.T) {
	s := config.ShelfConfig{}
	if got := s.EffectiveCatalogPath(); got != "catalog.yml" {
		t.Errorf("EffectiveCatalogPath = %q, want %q", got, "catalog.yml")
	}
}

func TestDefaultPath(t *testing.T) {
	p := config.DefaultPath()
	if p == "" {
		t.Fatal("DefaultPath returned empty string")
	}
	if !strings.HasSuffix(p, "config.yml") {
		t.Errorf("DefaultPath = %q, should end with config.yml", p)
	}
}

func TestShelfByName_ReturnsPointer(t *testing.T) {
	cfg := &config.Config{
		Shelves: []config.ShelfConfig{
			{Name: "test", Repo: "original"},
		},
	}
	s := cfg.ShelfByName("test")
	s.Repo = "modified"
	if cfg.Shelves[0].Repo != "modified" {
		t.Error("ShelfByName should return a pointer to the original slice element")
	}
}
