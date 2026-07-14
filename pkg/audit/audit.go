package audit

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/surajsrivastav/gitwhy/pkg/provenance"
)

type ExportFormat string

const (
	FormatJSON ExportFormat = "json"
	FormatCSV  ExportFormat = "csv"
)

type ExportOptions struct {
	From   time.Time
	To     time.Time
	Format ExportFormat
	Filter FilterOptions
}

// FilterOptions narrows the record set returned by an export or search.
// All non-empty fields are combined with AND logic.
type FilterOptions struct {
	By      string // exact match against attribution.by (e.g. "agent:claude-code")
	Model   string // exact match against context.model
	Ticket  string // exact match against context.ticket
	Origin  string // exact match against intent.origin
	Missing string // field name ("model", "ticket", "spec", "origin", "prompt") that must be empty/unknown
}

func (f FilterOptions) isZero() bool {
	return f.By == "" && f.Model == "" && f.Ticket == "" && f.Origin == "" && f.Missing == ""
}

// ApplyFilter returns the subset of records matching opts. An empty
// FilterOptions matches everything.
func ApplyFilter(records []*provenance.Record, opts FilterOptions) []*provenance.Record {
	if opts.isZero() {
		return records
	}

	var out []*provenance.Record
	for _, r := range records {
		if opts.By != "" && string(r.Attribution.By) != opts.By {
			continue
		}
		if opts.Model != "" && r.Context.Model != opts.Model {
			continue
		}
		if opts.Ticket != "" && r.Context.Ticket != opts.Ticket {
			continue
		}
		if opts.Origin != "" && string(r.Intent.Origin) != opts.Origin {
			continue
		}
		if opts.Missing != "" && !fieldIsMissing(r, opts.Missing) {
			continue
		}
		out = append(out, r)
	}
	return out
}

func fieldIsMissing(r *provenance.Record, field string) bool {
	switch field {
	case "model":
		return r.Context.Model == "" || r.Context.Model == string(provenance.OriginUnknown)
	case "ticket":
		return r.Context.Ticket == ""
	case "spec":
		return r.Intent.Spec == ""
	case "origin":
		return r.Intent.Origin == "" || r.Intent.Origin == provenance.OriginUnknown
	case "prompt":
		return r.Context.Prompt == ""
	default:
		return false
	}
}

type Summary struct {
	TotalCommits      int            `json:"total_commits"`
	AttributedCommits int            `json:"attributed_commits"`
	AIPercentage      float64        `json:"ai_percentage"`
	HumanPercentage   float64        `json:"human_percentage"`
	AgentBreakdown    map[string]int `json:"agent_breakdown"`
	SpecCoverage      int            `json:"spec_coverage"`
	DriftFlags        int            `json:"drift_flags"`
	CoverageTotal     int            `json:"coverage_total,omitempty"`
	CoverageCovered   int            `json:"coverage_covered,omitempty"`
}

// SetCoverage records how many of the repo's total commits (covered) have a
// provenance record, out of the overall commit count (total). Pass total <= 0
// to leave the coverage line out of String().
func (s *Summary) SetCoverage(total, covered int) {
	s.CoverageTotal = total
	s.CoverageCovered = covered
}

func GenerateExport(records []*provenance.Record, opts ExportOptions) (string, error) {
	var filtered []*provenance.Record
	for _, r := range records {
		t, err := time.Parse(time.RFC3339, r.Attribution.Timestamp)
		if err != nil {
			continue
		}
		if !opts.From.IsZero() && t.Before(opts.From) {
			continue
		}
		if !opts.To.IsZero() && t.After(opts.To) {
			continue
		}
		filtered = append(filtered, r)
	}

	filtered = ApplyFilter(filtered, opts.Filter)

	switch opts.Format {
	case FormatJSON:
		return exportJSON(filtered)
	case FormatCSV:
		return exportCSV(filtered)
	default:
		return "", fmt.Errorf("unsupported export format: %s", opts.Format)
	}
}

func GenerateSummary(records []*provenance.Record) *Summary {
	summary := &Summary{
		TotalCommits:   len(records),
		AgentBreakdown: make(map[string]int),
	}

	for _, r := range records {
		summary.AttributedCommits++

		by := string(r.Attribution.By)
		if provenance.IsAgentAttribution(r.Attribution.By) {
			summary.AgentBreakdown[by]++
		}

		if r.Intent.Spec != "" {
			summary.SpecCoverage++
		}

		if r.Intent.SpecHash != "" {
			summary.DriftFlags++
		}
	}

	if summary.TotalCommits > 0 {
		aiCount := 0
		for _, count := range summary.AgentBreakdown {
			aiCount += count
		}
		for _, rec := range records {
			if rec.Attribution.By == provenance.AttributionCopilot {
				aiCount++
			}
		}
		summary.AIPercentage = float64(aiCount) / float64(summary.TotalCommits) * 100
		summary.HumanPercentage = 100 - summary.AIPercentage
	}

	return summary
}

func (s *Summary) String() string {
	var b strings.Builder
	b.WriteString("gitwhy audit summary\n")
	b.WriteString(fmt.Sprintf("  Total commits:      %d\n", s.TotalCommits))
	b.WriteString(fmt.Sprintf("  Attributed commits: %d\n", s.AttributedCommits))
	b.WriteString(fmt.Sprintf("  AI-generated:       %.1f%%\n", s.AIPercentage))
	b.WriteString(fmt.Sprintf("  Human-authored:     %.1f%%\n", s.HumanPercentage))
	if len(s.AgentBreakdown) > 0 {
		b.WriteString("  Agent breakdown:\n")
		for agent, count := range s.AgentBreakdown {
			b.WriteString(fmt.Sprintf("    %s: %d\n", agent, count))
		}
	}
	b.WriteString(fmt.Sprintf("  Spec coverage:      %d commits\n", s.SpecCoverage))
	b.WriteString(fmt.Sprintf("  Drift flags:        %d commits\n", s.DriftFlags))
	if s.CoverageTotal > 0 {
		b.WriteString(fmt.Sprintf("  Coverage:           %d/%d commits\n", s.CoverageCovered, s.CoverageTotal))
	}
	return b.String()
}

// TableFormat renders records as a human-readable table for `ghw audit search`.
func TableFormat(records []*provenance.Record) string {
	if len(records) == 0 {
		return "  no matching records\n"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "  %-9s  %-20s  %-19s  %-12s  %-10s  %s\n",
		"COMMIT", "BY", "WHEN", "TICKET", "MODEL", "SPEC")
	for _, r := range records {
		ref := r.Target.Ref
		if len(ref) > 9 {
			ref = ref[:9]
		}
		when := r.Attribution.Timestamp
		if t, err := time.Parse(time.RFC3339, when); err == nil {
			when = t.Format("2006-01-02 15:04")
		}
		fmt.Fprintf(&b, "  %-9s  %-20s  %-19s  %-12s  %-10s  %s\n",
			ref, r.Attribution.By, when, r.Context.Ticket, r.Context.Model, r.Intent.Spec)
	}
	return b.String()
}

func exportJSON(records []*provenance.Record) (string, error) {
	data, _ := json.MarshalIndent(records, "", "  ")
	return string(data), nil
}

func exportCSV(records []*provenance.Record) (string, error) {
	var b strings.Builder
	writer := csv.NewWriter(&b)

	header := []string{"ref", "type", "attribution", "timestamp", "intent", "spec", "spec_hash", "origin", "ticket", "prompt", "model"}
	writer.Write(header)

	for _, r := range records {
		row := []string{
			r.Target.Ref,
			string(r.Target.Type),
			string(r.Attribution.By),
			r.Attribution.Timestamp,
			r.Intent.Summary,
			r.Intent.Spec,
			r.Intent.SpecHash,
			string(r.Intent.Origin),
			r.Context.Ticket,
			r.Context.Prompt,
			r.Context.Model,
		}
		writer.Write(row)
	}

	writer.Flush()
	return b.String(), nil
}
