package cmd

import (
	"testing"

	"github.com/anomalyco/gitwhy/pkg/provenance"
)

func TestIsPassthrough(t *testing.T) {
	tests := []struct {
		args     []string
		expected bool
	}{
		{[]string{}, false},
		{[]string{"commit"}, false},
		{[]string{"init"}, false},
		{[]string{"config"}, false},
		{[]string{"why"}, false},
		{[]string{"log"}, false},
		{[]string{"diff"}, false},
		{[]string{"audit"}, false},
		{[]string{"version"}, false},
		{[]string{"pr", "create"}, false},
		{[]string{"pr", "view"}, false},
		{[]string{"pr", "list"}, false},
		{[]string{"pr", "checkout"}, true},
		{[]string{"pr", "merge"}, true},
		{[]string{"issue", "list"}, true},
		{[]string{"release", "create"}, true},
		{[]string{"repo", "view"}, true},
		{[]string{"auth", "login"}, true},
		{[]string{"help"}, false},
		{[]string{"--help"}, false},
		{[]string{"-h"}, false},
		{[]string{"unknown"}, true},
	}

	for _, tt := range tests {
		name := ""
		for _, a := range tt.args {
			if name != "" {
				name += " "
			}
			name += a
		}
		t.Run(name, func(t *testing.T) {
			if got := isPassthrough(tt.args); got != tt.expected {
				t.Errorf("isPassthrough(%v) = %v, want %v", tt.args, got, tt.expected)
			}
		})
	}
}

func TestHasProvenanceFlags(t *testing.T) {
	commitFlag = commitFlags{}
	if hasProvenanceFlags() {
		t.Error("expected false when all flags are empty")
	}
}

func TestHasPRProvenanceFlags(t *testing.T) {
	prCreateFlags = struct {
		intent string
		origin string
		spec   string
		agent  string
		title  string
		body   string
	}{}
	if hasPRProvenanceFlags() {
		t.Error("expected false when all PR flags are empty")
	}
}

func TestExtractPRNumber(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://github.com/org/repo/pull/412", "412"},
		{"https://github.com/org/repo/pull/1", "1"},
		{"https://github.com/org/repo/pull/abc", ""},
		{"", ""},
		{"not-a-url", ""},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			if got := extractPRNumber(tt.url); got != tt.expected {
				t.Errorf("extractPRNumber(%q) = %q, want %q", tt.url, got, tt.expected)
			}
		})
	}
}

func TestPrintWhyPanel(t *testing.T) {
	record := provenance.NewRecord(provenance.TargetCommit, "abc123")
	record.SetAttribution(provenance.AgentAttribution("blue"))
	record.SetIntent("add retry logic", provenance.OriginSpec, "BLUE-317", "a1b2c3d")
	record.SetContext("JIRA-42", "implement exponential backoff", "claude-4")

	if record.Schema != "gitwhy/v1" {
		t.Errorf("expected schema gitwhy/v1, got %s", record.Schema)
	}
}

func TestPrintLogAnnotation(t *testing.T) {
	record := provenance.NewRecord(provenance.TargetCommit, "abc123")
	record.SetAttribution(provenance.AgentAttribution("blue"))
	record.SetIntent("add retry logic", provenance.OriginSpec, "BLUE-317", "a1b2c3d")

	if record.Intent.Summary != "add retry logic" {
		t.Errorf("expected intent summary, got %s", record.Intent.Summary)
	}
}

func TestPrintFullProvenance(t *testing.T) {
	record := provenance.NewRecord(provenance.TargetCommit, "abc123")
	record.SetAttribution(provenance.AgentAttribution("blue"))
	record.SetIntent("add retry logic", provenance.OriginSpec, "BLUE-317", "a1b2c3d")
	record.SetContext("JIRA-42", "implement exponential backoff", "claude-4")

	if record.Target.Ref != "abc123" {
		t.Errorf("expected ref abc123, got %s", record.Target.Ref)
	}
}
