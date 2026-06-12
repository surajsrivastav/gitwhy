package audit

import (
	"testing"
	"time"

	"github.com/anomalyco/gitwhy/pkg/provenance"
)

func makeRecord(ref string, by provenance.AttributionType, spec string, ts string) *provenance.Record {
	r := provenance.NewRecord(provenance.TargetCommit, ref)
	r.SetAttribution(by)
	r.SetIntent("test intent", provenance.OriginSpec, spec, "hash123")
	if ts != "" {
		r.Attribution.Timestamp = ts
	}
	return r
}

func TestGenerateSummary(t *testing.T) {
	records := []*provenance.Record{
		makeRecord("abc", provenance.AttributionHuman, "SPEC-1", ""),
		makeRecord("def", provenance.AgentAttribution("blue"), "SPEC-2", ""),
		makeRecord("ghi", provenance.AttributionCopilot, "", ""),
		makeRecord("jkl", provenance.AttributionHuman, "SPEC-3", ""),
	}

	summary := GenerateSummary(records)

	if summary.TotalCommits != 4 {
		t.Errorf("expected 4 commits, got %d", summary.TotalCommits)
	}
	if summary.AttributedCommits != 4 {
		t.Errorf("expected 4 attributed, got %d", summary.AttributedCommits)
	}
	if summary.SpecCoverage != 3 {
		t.Errorf("expected 3 spec coverage, got %d", summary.SpecCoverage)
	}
	if summary.AIPercentage <= 0 {
		t.Errorf("expected AI percentage > 0, got %f", summary.AIPercentage)
	}
}

func TestGenerateSummaryEmpty(t *testing.T) {
	summary := GenerateSummary(nil)

	if summary.TotalCommits != 0 {
		t.Errorf("expected 0 commits, got %d", summary.TotalCommits)
	}
	if summary.AIPercentage != 0 {
		t.Errorf("expected 0%% AI, got %f", summary.AIPercentage)
	}
}

func TestGenerateSummaryNoAI(t *testing.T) {
	records := []*provenance.Record{
		makeRecord("abc", provenance.AttributionHuman, "", ""),
		makeRecord("def", provenance.AttributionHuman, "SPEC-1", ""),
	}

	summary := GenerateSummary(records)

	if summary.AIPercentage != 0 {
		t.Errorf("expected 0%% AI, got %f", summary.AIPercentage)
	}
	if summary.HumanPercentage != 100 {
		t.Errorf("expected 100%% human, got %f", summary.HumanPercentage)
	}
}

func TestGenerateSummaryAgentBreakdown(t *testing.T) {
	records := []*provenance.Record{
		makeRecord("a", provenance.AgentAttribution("blue"), "", ""),
		makeRecord("b", provenance.AgentAttribution("blue"), "", ""),
		makeRecord("c", provenance.AgentAttribution("copilot"), "", ""),
	}

	summary := GenerateSummary(records)

	if len(summary.AgentBreakdown) != 2 {
		t.Errorf("expected 2 agent types, got %d: %v", len(summary.AgentBreakdown), summary.AgentBreakdown)
	}
	if summary.AgentBreakdown["agent:blue"] != 2 {
		t.Errorf("expected 2 blue agents, got %d", summary.AgentBreakdown["agent:blue"])
	}
}

func TestGenerateExportJSON(t *testing.T) {
	records := []*provenance.Record{
		makeRecord("abc", provenance.AttributionHuman, "SPEC-1", "2026-01-15T10:00:00Z"),
		makeRecord("def", provenance.AgentAttribution("blue"), "SPEC-2", "2026-02-20T10:00:00Z"),
	}

	opts := ExportOptions{
		Format: FormatJSON,
	}

	output, err := GenerateExport(records, opts)
	if err != nil {
		t.Fatalf("GenerateExport() error = %v", err)
	}
	if len(output) == 0 {
		t.Error("expected non-empty output")
	}
}

func TestGenerateExportCSV(t *testing.T) {
	records := []*provenance.Record{
		makeRecord("abc", provenance.AttributionHuman, "SPEC-1", "2026-01-15T10:00:00Z"),
	}

	opts := ExportOptions{
		Format: FormatCSV,
	}

	output, err := GenerateExport(records, opts)
	if err != nil {
		t.Fatalf("GenerateExport() error = %v", err)
	}
	if len(output) == 0 {
		t.Error("expected non-empty CSV output")
	}
}

func TestGenerateExportWithDateFilter(t *testing.T) {
	records := []*provenance.Record{
		makeRecord("a", provenance.AttributionHuman, "", "2026-01-01T00:00:00Z"),
		makeRecord("b", provenance.AttributionHuman, "", "2026-06-15T00:00:00Z"),
		makeRecord("c", provenance.AttributionHuman, "", "2026-12-31T00:00:00Z"),
	}

	from, _ := time.Parse("2006-01-02", "2026-06-01")
	to, _ := time.Parse("2006-01-02", "2026-12-31")

	opts := ExportOptions{
		From:   from,
		To:     to,
		Format: FormatJSON,
	}

	output, err := GenerateExport(records, opts)
	if err != nil {
		t.Fatalf("GenerateExport() error = %v", err)
	}

	if len(output) == 0 {
		t.Error("expected records in date range")
	}
}

func TestGenerateExportEmptyDateRange(t *testing.T) {
	records := []*provenance.Record{
		makeRecord("a", provenance.AttributionHuman, "", "2026-01-01T00:00:00Z"),
	}

	from, _ := time.Parse("2006-01-02", "2027-01-01")

	opts := ExportOptions{
		From:   from,
		Format: FormatJSON,
	}

	output, err := GenerateExport(records, opts)
	if err != nil {
		t.Fatalf("GenerateExport() error = %v", err)
	}

	if len(output) < 2 {
		t.Error("expected valid JSON output for empty date range")
	}
}

func TestGenerateExportCSVStructure(t *testing.T) {
	r := provenance.NewRecord(provenance.TargetCommit, "abc123")
	r.SetAttribution(provenance.AgentAttribution("blue"))
	r.SetIntent("fix auth bug", provenance.OriginSpec, "BLUE-204", "hash123")
	r.SetContext("JIRA-42", "add retry logic", "claude-4")

	records := []*provenance.Record{r}

	opts := ExportOptions{Format: FormatCSV}
	output, err := GenerateExport(records, opts)
	if err != nil {
		t.Fatal(err)
	}

	if output == "" {
		t.Fatal("expected CSV output")
	}
}

func TestSummaryStringOutput(t *testing.T) {
	records := []*provenance.Record{
		makeRecord("a", provenance.AttributionHuman, "S-1", ""),
		makeRecord("b", provenance.AgentAttribution("blue"), "S-2", ""),
	}

	summary := GenerateSummary(records)
	s := summary.String()

	if s == "" {
		t.Error("expected non-empty summary string")
	}
}

func TestUnsupportedFormat(t *testing.T) {
	_, err := GenerateExport(nil, ExportOptions{Format: "xml"})
	if err == nil {
		t.Error("expected error for unsupported format")
	}
}
