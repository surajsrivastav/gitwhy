package drift

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// ghIssueViewer is overridable in tests so they don't depend on a real gh
// CLI / GitHub API being reachable.
var ghIssueViewer = execGHIssueView

var httpClient = &http.Client{Timeout: 15 * time.Second}

var githubIssueURLPattern = regexp.MustCompile(`^https?://github\.com/([^/]+)/([^/]+)/issues/(\d+)/?$`)

// IsURL reports whether spec looks like an http(s) URL rather than a local
// file path.
func IsURL(spec string) bool {
	return strings.HasPrefix(spec, "http://") || strings.HasPrefix(spec, "https://")
}

// FetchSpecContent resolves the raw text content of a spec reference. It
// supports:
//   - GitHub issue URLs (https://github.com/<owner>/<repo>/issues/<n>) —
//     fetched via the local `gh` CLI, so no separate API token is needed.
//   - Any other http(s) URL — fetched directly.
//   - A local file path — read from disk.
func FetchSpecContent(spec string) (string, error) {
	if owner, repo, number, ok := parseGitHubIssueURL(spec); ok {
		return ghIssueViewer(owner, repo, number)
	}
	if IsURL(spec) {
		return fetchHTTPContent(spec)
	}
	return readFileContent(spec)
}

// HashSpecAny hashes the content behind a spec reference, dispatching to
// FetchSpecContent for URLs and to HashSpec for local paths.
func HashSpecAny(spec string) (string, error) {
	if IsURL(spec) {
		content, err := FetchSpecContent(spec)
		if err != nil {
			return "", err
		}
		return HashSpecContent(content), nil
	}
	return HashSpec(spec)
}

func parseGitHubIssueURL(spec string) (owner, repo, number string, ok bool) {
	m := githubIssueURLPattern.FindStringSubmatch(spec)
	if m == nil {
		return "", "", "", false
	}
	return m[1], m[2], m[3], true
}

func execGHIssueView(owner, repo, number string) (string, error) {
	cmd := exec.Command("gh", "issue", "view", number, "-R", owner+"/"+repo, "--json", "body")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh issue view %s#%s: %w\n%s", owner+"/"+repo, number, err, string(output))
	}

	var parsed struct {
		Body string `json:"body"`
	}
	if err := json.Unmarshal(output, &parsed); err != nil {
		return "", fmt.Errorf("parse gh issue view output: %w", err)
	}
	return parsed.Body, nil
}

func fetchHTTPContent(rawURL string) (string, error) {
	if _, err := url.ParseRequestURI(rawURL); err != nil {
		return "", fmt.Errorf("invalid spec url: %w", err)
	}

	resp, err := httpClient.Get(rawURL)
	if err != nil {
		return "", fmt.Errorf("fetch spec url: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("fetch spec url: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read spec url body: %w", err)
	}
	return string(body), nil
}

func readFileContent(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read spec file: %w", err)
	}
	return string(data), nil
}
