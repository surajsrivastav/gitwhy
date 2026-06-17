package cmd

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/anomalyco/gitwhy/pkg/config"
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

func changedFilesList(repoPath string) []string {
	out, err := exec.Command("git", "-C", repoPath, "diff-tree", "--no-commit-id", "-r", "--name-only", "--root", "HEAD").CombinedOutput()
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}
	return lines
}

func generateSummary(summaryCfg *config.SummaryConfig, repoPath string) string {
	if summaryCfg == nil || !summaryCfg.Enabled || summaryCfg.Command == "" {
		return ""
	}

	files := changedFilesList(repoPath)
	if len(files) == 0 {
		return ""
	}

	var prompt string
	switch summaryCfg.Mode {
	case config.SummaryModeDiff:
		var buf bytes.Buffer
		for _, f := range files {
			out, err := exec.Command("git", "-C", repoPath, "diff", "HEAD~1", "HEAD", "--", f).CombinedOutput()
			if err == nil && len(bytes.TrimSpace(out)) > 0 {
				buf.Write(out)
				buf.WriteByte('\n')
			}
		}
		diff := strings.TrimSpace(buf.String())
		if diff == "" {
			diff = strings.Join(files, ", ")
		}
		prompt = fmt.Sprintf("Summarize this code change in one short line:\n\n%s", diff)
	default:
		prompt = fmt.Sprintf("Summarize this change in a few words: %s", strings.Join(files, ", "))
	}

	args := strings.Fields(summaryCfg.Command)
	if len(args) == 0 {
		return ""
	}

	cmd := exec.Command(args[0], append(args[1:], prompt)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}

	result := strings.TrimSpace(string(out))
	if result == "" {
		return ""
	}
	return result
}

func autoFillContext(record *provenance.Record, repoPath string, intentSet, ticketSet bool, summaryCfg *config.SummaryConfig) {
	branch := currentBranch(repoPath)
	record.SetGitContext(branch)

	if !ticketSet && record.Context.Ticket == "" {
		record.Context.Ticket = extractTicketFromBranch(branch)
	}

	if !intentSet && record.Intent.Summary == "" {
		summary := generateSummary(summaryCfg, repoPath)
		if summary != "" {
			record.Intent.Summary = summary
			record.Intent.Origin = provenance.OriginAI
		} else {
			summary, origin := parseIntentFromMessage(commitMessage(repoPath))
			if summary != "" {
				record.Intent.Summary = summary
				if origin != "" {
					record.Intent.Origin = provenance.OriginType(origin)
				}
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
