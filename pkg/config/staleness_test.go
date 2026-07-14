package config

import (
	"testing"
	"time"
)

func TestModelStaleDaysNoModel(t *testing.T) {
	cfg := DefaultConfig()
	if _, ok := cfg.ModelStaleDays(); ok {
		t.Error("expected ok=false when no default_model is set")
	}
}

func TestModelStaleDaysNoTimestamp(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultModel = "claude-sonnet-4-6"
	if _, ok := cfg.ModelStaleDays(); ok {
		t.Error("expected ok=false when model_updated_at is unset")
	}
}

func TestModelStaleDaysRecent(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultModel = "claude-sonnet-4-6"
	cfg.ModelUpdatedAt = time.Now().UTC().Format(time.RFC3339)

	days, ok := cfg.ModelStaleDays()
	if !ok {
		t.Fatal("expected ok=true")
	}
	if days != 0 {
		t.Errorf("expected 0 days, got %d", days)
	}
	if cfg.IsModelStale() {
		t.Error("expected recent model to not be stale")
	}
}

func TestIsModelStaleOverThreshold(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultModel = "claude-sonnet-4-6"
	cfg.ModelUpdatedAt = time.Now().UTC().Add(-45 * 24 * time.Hour).Format(time.RFC3339)

	if !cfg.IsModelStale() {
		t.Error("expected model set 45 days ago to be stale")
	}
	days, _ := cfg.ModelStaleDays()
	if days < 44 || days > 45 {
		t.Errorf("expected ~45 days, got %d", days)
	}
}

func TestIsModelStaleInvalidTimestamp(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultModel = "claude-sonnet-4-6"
	cfg.ModelUpdatedAt = "not-a-timestamp"

	if cfg.IsModelStale() {
		t.Error("expected invalid timestamp to not be treated as stale")
	}
}
