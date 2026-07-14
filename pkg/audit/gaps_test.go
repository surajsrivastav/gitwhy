package audit

import (
	"strings"
	"testing"

	"github.com/surajsrivastav/gitwhy/pkg/provenance"
)

func TestFindGapsNoRecords(t *testing.T) {
	commits := []CommitInfo{
		{Hash: "abc1234", Date: "2026-01-01T00:00:00Z", Subject: "first"},
		{Hash: "def5678", Date: "2026-01-02T00:00:00Z", Subject: "second"},
	}

	gaps := FindGaps(commits, nil)
	if len(gaps) != 2 {
		t.Fatalf("expected 2 gaps, got %d", len(gaps))
	}
}

func TestFindGapsAllCovered(t *testing.T) {
	commits := []CommitInfo{
		{Hash: "abc1234", Date: "2026-01-01T00:00:00Z", Subject: "first"},
	}
	records := []*provenance.Record{
		makeRecord("abc1234", provenance.AttributionHuman, "", ""),
	}

	gaps := FindGaps(commits, records)
	if len(gaps) != 0 {
		t.Fatalf("expected 0 gaps, got %d", len(gaps))
	}
}

func TestFindGapsPartial(t *testing.T) {
	commits := []CommitInfo{
		{Hash: "abc1234", Date: "2026-01-01T00:00:00Z", Subject: "covered"},
		{Hash: "def5678", Date: "2026-01-02T00:00:00Z", Subject: "missing"},
	}
	records := []*provenance.Record{
		makeRecord("abc1234", provenance.AttributionHuman, "", ""),
	}

	gaps := FindGaps(commits, records)
	if len(gaps) != 1 {
		t.Fatalf("expected 1 gap, got %d", len(gaps))
	}
	if gaps[0].Hash != "def5678" {
		t.Errorf("expected gap for def5678, got %q", gaps[0].Hash)
	}
	if gaps[0].Subject != "missing" {
		t.Errorf("expected subject 'missing', got %q", gaps[0].Subject)
	}
}

func TestFindGapsAbbreviatedHashMatch(t *testing.T) {
	commits := []CommitInfo{
		{Hash: "abc1234567890", Date: "2026-01-01T00:00:00Z", Subject: "covered"},
	}
	records := []*provenance.Record{
		makeRecord("abc1234", provenance.AttributionHuman, "", ""),
	}

	gaps := FindGaps(commits, records)
	if len(gaps) != 0 {
		t.Errorf("expected abbreviated hash to be treated as covered, got %d gaps", len(gaps))
	}
}

func TestGapReportNoGaps(t *testing.T) {
	report := GapReport(nil, 5)
	if !strings.Contains(report, "no gaps") {
		t.Errorf("expected 'no gaps' message, got %q", report)
	}
	if !strings.Contains(report, "5") {
		t.Errorf("expected total count in report, got %q", report)
	}
}

func TestGapReportWithGaps(t *testing.T) {
	gaps := []Gap{
		{Hash: "abc1234567", Date: "2026-01-01T00:00:00Z", Subject: "feat: thing"},
	}
	report := GapReport(gaps, 3)
	if !strings.Contains(report, "1 commits missing provenance records (out of 3 total)") {
		t.Errorf("expected summary line, got %q", report)
	}
	if !strings.Contains(report, "abc1234") {
		t.Errorf("expected truncated hash, got %q", report)
	}
	if !strings.Contains(report, "2026-01-01") {
		t.Errorf("expected date, got %q", report)
	}
	if !strings.Contains(report, "feat: thing") {
		t.Errorf("expected subject, got %q", report)
	}
}
