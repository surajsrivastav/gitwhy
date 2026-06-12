package passthrough

import (
	"testing"
)

func TestIsPassthroughCommand(t *testing.T) {
	tests := []struct {
		cmd      string
		expected bool
	}{
		{"commit", false},
		{"init", false},
		{"config", false},
		{"why", false},
		{"log", false},
		{"diff", false},
		{"audit", false},
		{"pr", false},
		{"issue", true},
		{"release", true},
		{"repo", true},
		{"auth", true},
		{"help", true},
		{"status", true},
		{"unknown-command", true},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			if got := IsPassthroughCommand(tt.cmd); got != tt.expected {
				t.Errorf("IsPassthroughCommand(%q) = %v, want %v", tt.cmd, got, tt.expected)
			}
		})
	}
}

func TestIsGHAvailable(t *testing.T) {
	available := IsGHAvailable()
	if ghBin != "" && !available {
		t.Error("gh is in PATH but IsGHAvailable returns false")
	}
}

func TestGHPath(t *testing.T) {
	path := GHPath()
	if ghBin != "" && path != ghBin {
		t.Errorf("GHPath() = %q, want %q", path, ghBin)
	}
}

func TestExecuteWithNoGH(t *testing.T) {
	if IsGHAvailable() {
		t.Skip("gh is available, skipping no-GH test")
	}

	err := Execute([]string{"pr", "list"})
	if err == nil {
		t.Error("expected error when gh is not in PATH")
	}
}

func TestExecuteWithOutputNoGH(t *testing.T) {
	if IsGHAvailable() {
		t.Skip("gh is available, skipping no-GH test")
	}

	_, err := ExecuteWithOutput([]string{"pr", "list"})
	if err == nil {
		t.Error("expected error when gh is not in PATH")
	}
}
