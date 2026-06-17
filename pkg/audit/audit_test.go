package audit

import (
	"strings"
	"testing"
	"time"

	"github.com/surajsrivastav/gitwhy/pkg/provenance"
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
	if !strings.Contains(output, "SPEC-1") {
		t.Error("expected SPEC-1 in JSON output")
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
	if !strings.Contains(output, "ref,type") {
		t.Error("expected CSV header in output")
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
	if strings.Contains(output, `"ref": "a"`) {
		t.Error("expected record a (before From) to be filtered out")
	}
	if !strings.Contains(output, `"ref": "b"`) {
		t.Error("expected record b (in range) to be included")
	}
	if !strings.Contains(output, `"ref": "c"`) {
		t.Error("expected record c (in range) to be included")
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
	if strings.Contains(output, `"ref": "a"`) {
		t.Error("expected record a (before From) to be filtered out")
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
	if !strings.Contains(output, "BLUE-204") {
		t.Error("expected spec in CSV output")
	}
	if !strings.Contains(output, "JIRA-42") {
		t.Error("expected ticket in CSV output")
	}
}

func TestSummaryStringOutput(t *testing.T) {
	records := []*provenance.Record{
		makeRecord("a", provenance.AttributionHuman, "S-1", ""),
		makeRecord("b", provenance.AgentAttribution("blue"), "S-2", ""),
	}

	summary := GenerateSummary(records)
	s := summary.String()

	expectedLines := []string{
		"gitwhy audit summary",
		"Total commits:",
		"Attributed commits:",
		"AI-generated:",
		"Human-authored:",
		"Agent breakdown:",
		"agent:blue",
		"Spec coverage:",
		"Drift flags:",
	}
	for _, line := range expectedLines {
		if !strings.Contains(s, line) {
			t.Errorf("expected summary to contain %q", line)
		}
	}
}

func TestUnsupportedFormat(t *testing.T) {
	_, err := GenerateExport(nil, ExportOptions{Format: "xml"})
	if err == nil {
		t.Error("expected error for unsupported format")
	}
}

func TestGenerateExportAllAIData(t *testing.T) {
	records := []*provenance.Record{
		makeRecord("a", provenance.AttributionCopilot, "SPEC-1", "2026-01-15T10:00:00Z"),
		makeRecord("b", provenance.AgentAttribution("blue"), "SPEC-2", "2026-01-16T10:00:00Z"),
	}

	summary := GenerateSummary(records)
	if summary.AIPercentage != 100 {
		t.Errorf("expected 100%% AI, got %f", summary.AIPercentage)
	}
	if summary.HumanPercentage != 0 {
		t.Errorf("expected 0%% human, got %f", summary.HumanPercentage)
	}
	if summary.SpecCoverage != 2 {
		t.Errorf("expected 2 spec coverage, got %d", summary.SpecCoverage)
	}
}

func TestGenerateExportMixedAIData(t *testing.T) {
	records := []*provenance.Record{
		makeRecord("a", provenance.AttributionHuman, "SPEC-1", "2026-01-15T10:00:00Z"),
		makeRecord("b", provenance.AttributionCopilot, "SPEC-2", "2026-01-16T10:00:00Z"),
	}

	summary := GenerateSummary(records)
	if summary.AIPercentage != 50 {
		t.Errorf("expected 50%% AI, got %f", summary.AIPercentage)
	}
	if summary.HumanPercentage != 50 {
		t.Errorf("expected 50%% human, got %f", summary.HumanPercentage)
	}
}

func TestGenerateExportSingleCommit(t *testing.T) {
	records := []*provenance.Record{
		makeRecord("a", provenance.AttributionHuman, "SPEC-1", ""),
	}

	summary := GenerateSummary(records)
	if summary.TotalCommits != 1 {
		t.Errorf("expected 1 commit, got %d", summary.TotalCommits)
	}
	if summary.AIPercentage != 0 {
		t.Errorf("expected 0%% AI, got %f", summary.AIPercentage)
	}
	if summary.HumanPercentage != 100 {
		t.Errorf("expected 100%% human, got %f", summary.HumanPercentage)
	}
}

func TestGenerateExportAgentCountSum(t *testing.T) {
	records := []*provenance.Record{
		makeRecord("a", provenance.AgentAttribution("blue"), "SPEC-1", ""),
		makeRecord("b", provenance.AgentAttribution("blue"), "SPEC-2", ""),
		makeRecord("c", provenance.AgentAttribution("red"), "SPEC-3", ""),
		makeRecord("d", provenance.AttributionCopilot, "SPEC-4", ""),
		makeRecord("e", provenance.AttributionHuman, "SPEC-5", ""),
	}

	summary := GenerateSummary(records)
	if summary.AIPercentage != 80 {
		t.Errorf("expected 80%% AI (2 blue + 1 red + 1 copilot = 4/5), got %f", summary.AIPercentage)
	}
	if summary.AgentBreakdown["agent:blue"] != 2 {
		t.Errorf("expected 2 agent:blue, got %d", summary.AgentBreakdown["agent:blue"])
	}
	if summary.AgentBreakdown["agent:red"] != 1 {
		t.Errorf("expected 1 agent:red, got %d", summary.AgentBreakdown["agent:red"])
	}
}

func TestGenerateExportWithDriftFlags(t *testing.T) {
	r := makeRecord("a", provenance.AttributionHuman, "SPEC-1", "2026-01-15T10:00:00Z")
	r.Intent.SpecHash = "abc123"

	summary := GenerateSummary([]*provenance.Record{r})
	if summary.DriftFlags != 1 {
		t.Errorf("expected 1 drift flag, got %d", summary.DriftFlags)
	}
}

func TestGenerateExportWithInvalidTimestamp(t *testing.T) {
	r1 := makeRecord("a", provenance.AttributionHuman, "SPEC-1", "2026-01-01T00:00:00Z")
	r2 := makeRecord("bad", provenance.AttributionHuman, "SPEC-1", "not-a-timestamp")
	r3 := makeRecord("c", provenance.AttributionHuman, "SPEC-1", "2026-03-01T00:00:00Z")

	opts := ExportOptions{Format: FormatJSON}
	output, err := GenerateExport([]*provenance.Record{r1, r2, r3}, opts)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output, `"ref": "bad"`) {
		t.Error("expected record with invalid timestamp to be filtered out")
	}
	if !strings.Contains(output, `"ref": "a"`) {
		t.Error("expected record a (before invalid timestamp) to be included")
	}
	if !strings.Contains(output, `"ref": "c"`) {
		t.Error("expected record c (after invalid timestamp) to be included")
	}
}

func TestGenerateExportWithToFilter(t *testing.T) {
	records := []*provenance.Record{
		makeRecord("a", provenance.AttributionHuman, "", "2026-05-15T00:00:00Z"),
		makeRecord("b", provenance.AttributionHuman, "", "2026-06-15T00:00:00Z"),
		makeRecord("c", provenance.AttributionHuman, "", "2026-05-20T00:00:00Z"),
	}

	to, _ := time.Parse("2006-01-02", "2026-06-01")
	opts := ExportOptions{To: to, Format: FormatCSV}

	output, err := GenerateExport(records, opts)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output, "\nb,") {
		t.Error("expected record b (after To date) to be filtered out")
	}
	if !strings.Contains(output, "\na,") {
		t.Error("expected record a (before To date) to be included")
	}
	if !strings.Contains(output, "\nc,") {
		t.Error("expected record c (before To date, after b) to be included")
	}
}

func TestGenerateExportWithFromFilter(t *testing.T) {
	records := []*provenance.Record{
		makeRecord("a", provenance.AttributionHuman, "", "2026-01-01T00:00:00Z"),
	}

	from, _ := time.Parse("2006-01-02", "2026-06-01")
	opts := ExportOptions{From: from, Format: FormatJSON}

	output, err := GenerateExport(records, opts)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output, `"ref": "a"`) {
		t.Error("expected record before From date to be filtered out")
	}
}

func TestSummaryStringNoAgents(t *testing.T) {
	records := []*provenance.Record{
		makeRecord("a", provenance.AttributionHuman, "", ""),
	}

	summary := GenerateSummary(records)
	s := summary.String()

	if strings.Contains(s, "Agent breakdown") {
		t.Error("expected no Agent breakdown when there are no agents")
	}
}

func TestExportFormats(t *testing.T) {
	if FormatJSON != "json" {
		t.Errorf("expected FormatJSON to be 'json', got %q", FormatJSON)
	}
	if FormatCSV != "csv" {
		t.Errorf("expected FormatCSV to be 'csv', got %q", FormatCSV)
	}
}
