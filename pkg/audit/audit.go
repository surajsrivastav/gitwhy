package audit

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/anomalyco/gitwhy/pkg/provenance"
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
}

type Summary struct {
	TotalCommits      int     `json:"total_commits"`
	AttributedCommits int     `json:"attributed_commits"`
	AIPercentage      float64 `json:"ai_percentage"`
	HumanPercentage   float64 `json:"human_percentage"`
	AgentBreakdown    map[string]int `json:"agent_breakdown"`
	SpecCoverage      int     `json:"spec_coverage"`
	DriftFlags        int     `json:"drift_flags"`
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
	return b.String()
}

func exportJSON(records []*provenance.Record) (string, error) {
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal records: %w", err)
	}
	return string(data), nil
}

func exportCSV(records []*provenance.Record) (string, error) {
	var b strings.Builder
	writer := csv.NewWriter(&b)

	header := []string{"ref", "type", "attribution", "timestamp", "intent", "spec", "spec_hash", "origin", "ticket", "prompt", "model"}
	if err := writer.Write(header); err != nil {
		return "", err
	}

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
		if err := writer.Write(row); err != nil {
			return "", err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", err
	}
	return b.String(), nil
}
