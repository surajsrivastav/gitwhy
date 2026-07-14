package drift

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestIsURL(t *testing.T) {
	cases := map[string]bool{
		"https://example.com/spec.md": true,
		"http://example.com/spec.md":  true,
		"/local/path/spec.yaml":       false,
		"spec.yaml":                   false,
	}
	for spec, want := range cases {
		if got := IsURL(spec); got != want {
			t.Errorf("IsURL(%q) = %v, want %v", spec, got, want)
		}
	}
}

func TestFetchSpecContentLocalFile(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "spec.yaml")
	if err := os.WriteFile(specPath, []byte("hello spec"), 0644); err != nil {
		t.Fatal(err)
	}

	content, err := FetchSpecContent(specPath)
	if err != nil {
		t.Fatalf("FetchSpecContent() error = %v", err)
	}
	if content != "hello spec" {
		t.Errorf("expected %q, got %q", "hello spec", content)
	}
}

func TestFetchSpecContentHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "remote spec body")
	}))
	defer srv.Close()

	content, err := FetchSpecContent(srv.URL)
	if err != nil {
		t.Fatalf("FetchSpecContent() error = %v", err)
	}
	if content != "remote spec body" {
		t.Errorf("expected %q, got %q", "remote spec body", content)
	}
}

func TestFetchSpecContentHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := FetchSpecContent(srv.URL)
	if err == nil {
		t.Error("expected error for non-2xx response")
	}
}

func TestHashSpecAnyURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "same content")
	}))
	defer srv.Close()

	h1, err := HashSpecAny(srv.URL)
	if err != nil {
		t.Fatalf("HashSpecAny() error = %v", err)
	}
	h2 := HashSpecContent("same content")
	if h1 != h2 {
		t.Errorf("expected hash of fetched content to match HashSpecContent, got %q vs %q", h1, h2)
	}
}

func TestHashSpecAnyLocalFile(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "spec.yaml")
	os.WriteFile(specPath, []byte("local content"), 0644)

	h1, err := HashSpecAny(specPath)
	if err != nil {
		t.Fatalf("HashSpecAny() error = %v", err)
	}
	h2, _ := HashSpec(specPath)
	if h1 != h2 {
		t.Errorf("expected HashSpecAny to match HashSpec for local paths, got %q vs %q", h1, h2)
	}
}

func TestParseGitHubIssueURL(t *testing.T) {
	owner, repo, number, ok := parseGitHubIssueURL("https://github.com/acme/widgets/issues/42")
	if !ok {
		t.Fatal("expected match for github issue URL")
	}
	if owner != "acme" || repo != "widgets" || number != "42" {
		t.Errorf("got owner=%q repo=%q number=%q", owner, repo, number)
	}

	_, _, _, ok = parseGitHubIssueURL("https://example.com/not/an/issue")
	if ok {
		t.Error("expected no match for non-github URL")
	}
}

func TestFetchSpecContentGitHubIssue(t *testing.T) {
	orig := ghIssueViewer
	defer func() { ghIssueViewer = orig }()

	var gotOwner, gotRepo, gotNumber string
	ghIssueViewer = func(owner, repo, number string) (string, error) {
		gotOwner, gotRepo, gotNumber = owner, repo, number
		return "issue body content", nil
	}

	content, err := FetchSpecContent("https://github.com/acme/widgets/issues/42")
	if err != nil {
		t.Fatalf("FetchSpecContent() error = %v", err)
	}
	if content != "issue body content" {
		t.Errorf("expected issue body content, got %q", content)
	}
	if gotOwner != "acme" || gotRepo != "widgets" || gotNumber != "42" {
		t.Errorf("ghIssueViewer called with owner=%q repo=%q number=%q", gotOwner, gotRepo, gotNumber)
	}
}

func TestFetchSpecContentNonExistentFile(t *testing.T) {
	_, err := FetchSpecContent("/nonexistent/spec.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}
