package audit

import (
	"fmt"
	"strings"

	"github.com/surajsrivastav/gitwhy/pkg/provenance"
)

// CommitInfo is a minimal description of a git commit used for gap detection.
type CommitInfo struct {
	Hash    string
	Date    string
	Subject string
}

// Gap describes a commit that has no provenance record.
type Gap struct {
	Hash    string
	Date    string
	Subject string
}

// FindGaps returns the commits in `commits` that have no corresponding
// provenance record in `records`. A commit is considered covered if any
// record's target ref is a prefix-match (or exact match) of the commit hash,
// since records may be stored keyed by full or abbreviated hash.
func FindGaps(commits []CommitInfo, records []*provenance.Record) []Gap {
	recorded := make(map[string]bool, len(records))
	for _, r := range records {
		recorded[r.Target.Ref] = true
	}

	var gaps []Gap
	for _, c := range commits {
		if commitCovered(c.Hash, recorded) {
			continue
		}
		gaps = append(gaps, Gap{Hash: c.Hash, Date: c.Date, Subject: c.Subject})
	}
	return gaps
}

func commitCovered(hash string, recorded map[string]bool) bool {
	if recorded[hash] {
		return true
	}
	for ref := range recorded {
		if strings.HasPrefix(hash, ref) || strings.HasPrefix(ref, hash) {
			return true
		}
	}
	return false
}

// GapReport renders a human-readable gap report matching `ghw audit gaps` output.
func GapReport(gaps []Gap, total int) string {
	var b strings.Builder
	if len(gaps) == 0 {
		fmt.Fprintf(&b, "  no gaps — all %d commits have provenance records\n", total)
		return b.String()
	}

	fmt.Fprintf(&b, "  %d commits missing provenance records (out of %d total)\n\n", len(gaps), total)
	for _, g := range gaps {
		date := g.Date
		if len(date) > 10 {
			date = date[:10]
		}
		hash := g.Hash
		if len(hash) > 7 {
			hash = hash[:7]
		}
		fmt.Fprintf(&b, "  %s  %s  %s\n", hash, date, g.Subject)
	}
	return b.String()
}
