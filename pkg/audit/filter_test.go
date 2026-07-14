package audit

import (
	"strings"
	"testing"

	"github.com/surajsrivastav/gitwhy/pkg/provenance"
)

func TestApplyFilterEmpty(t *testing.T) {
	records := []*provenance.Record{
		makeRecord("a", provenance.AttributionHuman, "", ""),
		makeRecord("b", provenance.AgentAttribution("blue"), "", ""),
	}

	out := ApplyFilter(records, FilterOptions{})
	if len(out) != 2 {
		t.Fatalf("expected empty filter to match all records, got %d", len(out))
	}
}

func TestApplyFilterBy(t *testing.T) {
	records := []*provenance.Record{
		makeRecord("a", provenance.AttributionHuman, "", ""),
		makeRecord("b", provenance.AgentAttribution("claude-code"), "", ""),
	}

	out := ApplyFilter(records, FilterOptions{By: "agent:claude-code"})
	if len(out) != 1 || out[0].Target.Ref != "b" {
		t.Fatalf("expected 1 record (b), got %d", len(out))
	}
}

func TestApplyFilterModelAndTicket(t *testing.T) {
	r1 := provenance.NewRecord(provenance.TargetCommit, "a")
	r1.SetContext("JIRA-1", "", "claude-sonnet-4-6")

	r2 := provenance.NewRecord(provenance.TargetCommit, "b")
	r2.SetContext("JIRA-2", "", "claude-opus-4-7")

	records := []*provenance.Record{r1, r2}

	out := ApplyFilter(records, FilterOptions{Model: "claude-sonnet-4-6"})
	if len(out) != 1 || out[0].Target.Ref != "a" {
		t.Fatalf("expected 1 record (a), got %d", len(out))
	}

	out = ApplyFilter(records, FilterOptions{Ticket: "JIRA-2"})
	if len(out) != 1 || out[0].Target.Ref != "b" {
		t.Fatalf("expected 1 record (b), got %d", len(out))
	}
}

func TestApplyFilterMissingModel(t *testing.T) {
	r1 := provenance.NewRecord(provenance.TargetCommit, "a")
	r1.SetContext("", "", "claude-sonnet-4-6")

	r2 := provenance.NewRecord(provenance.TargetCommit, "b")
	r2.SetContext("", "", "")

	records := []*provenance.Record{r1, r2}

	out := ApplyFilter(records, FilterOptions{Missing: "model"})
	if len(out) != 1 || out[0].Target.Ref != "b" {
		t.Fatalf("expected 1 record (b) missing model, got %d", len(out))
	}
}

func TestApplyFilterCombinedAND(t *testing.T) {
	r1 := provenance.NewRecord(provenance.TargetCommit, "a")
	r1.SetAttribution(provenance.AttributionHuman)
	r1.SetContext("JIRA-1", "", "claude-sonnet-4-6")

	r2 := provenance.NewRecord(provenance.TargetCommit, "b")
	r2.SetAttribution(provenance.AttributionHuman)
	r2.SetContext("JIRA-2", "", "claude-sonnet-4-6")

	records := []*provenance.Record{r1, r2}

	out := ApplyFilter(records, FilterOptions{By: "human", Ticket: "JIRA-2"})
	if len(out) != 1 || out[0].Target.Ref != "b" {
		t.Fatalf("expected 1 record (b) from combined filter, got %d", len(out))
	}
}

func TestTableFormatEmpty(t *testing.T) {
	out := TableFormat(nil)
	if !strings.Contains(out, "no matching records") {
		t.Errorf("expected empty message, got %q", out)
	}
}

func TestTableFormatRows(t *testing.T) {
	r := provenance.NewRecord(provenance.TargetCommit, "abc123456")
	r.SetAttribution(provenance.AgentAttribution("claude-code"))
	r.SetContext("JIRA-42", "", "claude-sonnet-4-6")
	r.SetIntent("fix bug", provenance.OriginSpec, "SPEC-1", "")

	out := TableFormat([]*provenance.Record{r})
	if !strings.Contains(out, "COMMIT") {
		t.Errorf("expected table header, got %q", out)
	}
	if !strings.Contains(out, "abc123456"[:9]) {
		t.Errorf("expected truncated commit hash, got %q", out)
	}
	if !strings.Contains(out, "JIRA-42") {
		t.Errorf("expected ticket in row, got %q", out)
	}
	if !strings.Contains(out, "SPEC-1") {
		t.Errorf("expected spec in row, got %q", out)
	}
}
