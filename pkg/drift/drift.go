package drift

import (
	"crypto/sha256"
	"fmt"
	"os"
	"time"
)

type Severity string

const (
	SeverityNone   Severity = "none"
	SeverityLow    Severity = "low"
	SeverityMedium Severity = "medium"
	SeverityHigh   Severity = "high"
)

type Report struct {
	File             string   `json:"file"`
	Spec             string   `json:"spec"`
	GeneratedAt      string   `json:"generated_at"`
	GeneratedBy      string   `json:"generated_by"`
	SpecHashAtGen    string   `json:"spec_hash_at_generation"`
	CurrentSpecHash  string   `json:"current_spec_hash"`
	DriftDetected    bool     `json:"drift_detected"`
	Severity         Severity `json:"severity"`
	AffectedLines    int      `json:"affected_lines"`
	Recommendation   string   `json:"recommendation"`
}

func HashSpec(specPath string) (string, error) {
	data, err := os.ReadFile(specPath)
	if err != nil {
		return "", fmt.Errorf("read spec file: %w", err)
	}
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h[:8]), nil
}

func HashSpecContent(content string) string {
	h := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", h[:8])
}

func DetectDrift(filePath, specID, specPath, specHashAtGen, generatedBy string) (*Report, error) {
	report := &Report{
		File:          filePath,
		Spec:          specID,
		SpecHashAtGen: specHashAtGen,
		GeneratedBy:   generatedBy,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
	}

	if specPath == "" {
		report.DriftDetected = false
		report.Severity = SeverityNone
		report.Recommendation = "no spec path provided for comparison"
		return report, nil
	}

	currentHash, err := HashSpec(specPath)
	if err != nil {
		return nil, fmt.Errorf("hash current spec: %w", err)
	}
	report.CurrentSpecHash = currentHash

	if specHashAtGen == "" {
		report.DriftDetected = false
		report.Severity = SeverityNone
		report.Recommendation = "no original spec hash recorded for comparison"
		return report, nil
	}

	if currentHash != specHashAtGen {
		report.DriftDetected = true
		report.Severity = SeverityHigh
		report.AffectedLines = estimateAffectedLines(specPath)
		report.Recommendation = fmt.Sprintf(
			"DRIFT DETECTED — spec has changed since code generation\n"+
				"  Spec hash at generation: %s\n"+
				"  Current spec hash:       %s\n"+
				"  Recommendation: regenerate from %s or reconcile manually",
			specHashAtGen, currentHash, specID,
		)
	} else {
		report.DriftDetected = false
		report.Severity = SeverityNone
		report.Recommendation = "no drift detected — spec hash matches"
	}

	return report, nil
}

func estimateAffectedLines(specPath string) int {
	data, err := os.ReadFile(specPath)
	if err != nil {
		return 0
	}
	lines := 0
	for _, b := range data {
		if b == '\n' {
			lines++
		}
	}
	return lines
}

func (r *Report) String() string {
	s := fmt.Sprintf("  gitwhy drift report\n")
	s += fmt.Sprintf("  %s\n", dashes(50))
	s += fmt.Sprintf("  file:       %s\n", r.File)
	s += fmt.Sprintf("  spec:       %s\n", r.Spec)
	s += fmt.Sprintf("  generated:  %s  by %s\n", r.GeneratedAt[:10], r.GeneratedBy)
	s += fmt.Sprintf("  spec hash:  %s  (at generation)\n", r.SpecHashAtGen)
	if r.CurrentSpecHash != "" {
		s += fmt.Sprintf("  spec hash:  %s  (current)\n", r.CurrentSpecHash)
	}
	s += fmt.Sprintf("\n")

	if r.DriftDetected {
		s += fmt.Sprintf("  DRIFT DETECTED — spec has changed since code generation\n")
		if r.AffectedLines > 0 {
			s += fmt.Sprintf("  Lines affected: +%d in spec, not reflected in implementation\n", r.AffectedLines)
		}
		s += fmt.Sprintf("  %s\n", r.Recommendation)
	} else {
		s += fmt.Sprintf("  %s\n", r.Recommendation)
	}

	return s
}

func dashes(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		s += "─"
	}
	return s
}
