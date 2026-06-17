package cmd

import (
	"os/exec"
	"regexp"
	"strings"

	"github.com/anomalyco/gitwhy/pkg/provenance"
)

var ticketPattern = regexp.MustCompile(`[A-Z]+-\d+`)

var conventionalCommitRE = regexp.MustCompile(`^(\w+)(\([^)]*\))?(!)?\s*:\s*(.*)$`)

func currentBranch(repoPath string) string {
	out, err := exec.Command("git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD").CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func extractTicketFromBranch(branch string) string {
	if branch == "" {
		return ""
	}
	matches := ticketPattern.FindString(branch)
	return matches
}

func commitMessage(repoPath string) string {
	out, err := exec.Command("git", "-C", repoPath, "log", "-1", "--format=%s", "HEAD").CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func autoFillContext(record *provenance.Record, repoPath string, intentSet, ticketSet bool) {
	branch := currentBranch(repoPath)
	record.SetGitContext(branch)

	if !ticketSet && record.Context.Ticket == "" {
		record.Context.Ticket = extractTicketFromBranch(branch)
	}

	if !intentSet && record.Intent.Summary == "" {
		summary, origin := parseIntentFromMessage(commitMessage(repoPath))
		if summary != "" {
			record.Intent.Summary = summary
			if origin != "" {
				record.Intent.Origin = provenance.OriginType(origin)
			}
		}
	}
}

func parseIntentFromMessage(msg string) (summary string, origin string) {
	if msg == "" {
		return "", ""
	}
	firstLine := strings.SplitN(msg, "\n", 2)[0]
	firstLine = strings.TrimSpace(firstLine)

	matches := conventionalCommitRE.FindStringSubmatch(firstLine)
	if matches == nil {
		return "", ""
	}

	_type := matches[1]
	_ = matches[2]
	breaking := matches[3]
	description := strings.TrimSpace(matches[4])

	if description == "" {
		return "", ""
	}

	summary = description
	if breaking == "!" {
		summary = "BREAKING: " + description
	}

	switch _type {
	case "feat", "fix", "perf":
		origin = "spec"
	case "build", "chore", "ci", "docs", "refactor", "style", "test":
		origin = "human"
	default:
		origin = "human"
	}

	return summary, origin
}
