package passthrough

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsPassthroughCommand(t *testing.T) {
	tests := []struct {
		cmd      string
		expected bool
	}{
		{"commit", false},
		{"init", false},
		{"config", false},
		{"why", false},
		{"log", false},
		{"diff", false},
		{"audit", false},
		{"pr", false},
		{"issue", true},
		{"release", true},
		{"repo", true},
		{"auth", true},
		{"help", true},
		{"status", true},
		{"unknown-command", true},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			if got := IsPassthroughCommand(tt.cmd); got != tt.expected {
				t.Errorf("IsPassthroughCommand(%q) = %v, want %v", tt.cmd, got, tt.expected)
			}
		})
	}
}

func TestIsGHAvailable(t *testing.T) {
	available := IsGHAvailable()
	if ghBin != "" && !available {
		t.Error("gh is in PATH but IsGHAvailable returns false")
	}
}

func TestGHPath(t *testing.T) {
	path := GHPath()
	if ghBin != "" && path != ghBin {
		t.Errorf("GHPath() = %q, want %q", path, ghBin)
	}
}

func TestExecuteWithNoGH(t *testing.T) {
	if IsGHAvailable() {
		t.Skip("gh is available, skipping no-GH test")
	}

	err := Execute([]string{"pr", "list"})
	if err == nil {
		t.Error("expected error when gh is not in PATH")
	}
}

func TestExecuteWithOutputNoGH(t *testing.T) {
	if IsGHAvailable() {
		t.Skip("gh is available, skipping no-GH test")
	}

	_, err := ExecuteWithOutput([]string{"pr", "list"})
	if err == nil {
		t.Error("expected error when gh is not in PATH")
	}
}

func TestExecuteWithMockGH(t *testing.T) {
	tmpDir := t.TempDir()
	mockGh := filepath.Join(tmpDir, "gh")
	mockGhContent := "#!/bin/sh\necho \"mock-gh-output\"\n"
	if err := os.WriteFile(mockGh, []byte(mockGhContent), 0755); err != nil {
		t.Fatal(err)
	}

	origGhBin := ghBin
	ghBin = mockGh
	t.Cleanup(func() { ghBin = origGhBin })

	oldStdout := os.Stdout
	pr, pw, _ := os.Pipe()
	os.Stdout = pw

	err := Execute([]string{"pr", "list"})

	pw.Close()
	os.Stdout = oldStdout
	out, _ := io.ReadAll(pr)
	pr.Close()

	if err != nil {
		t.Errorf("Execute() with mock gh should not error: %v", err)
	}
	if !strings.Contains(string(out), "mock-gh-output") {
		t.Errorf("expected mock gh output in stdout, got: %s", string(out))
	}
}

func TestExecuteWithMockGHError(t *testing.T) {
	origGhBin := ghBin
	ghBin = "/nonexistent/gh"
	t.Cleanup(func() { ghBin = origGhBin })

	err := Execute([]string{"pr", "list"})
	if err == nil {
		t.Error("expected error when gh binary doesn't exist")
	}
}

func TestExecuteWithOutputGHError(t *testing.T) {
	origGhBin := ghBin
	ghBin = "/nonexistent/gh"
	t.Cleanup(func() { ghBin = origGhBin })

	_, err := ExecuteWithOutput([]string{"pr", "list"})
	if err == nil {
		t.Error("expected error when gh binary doesn't exist")
	}
}

func TestExecuteGHBinEmpty(t *testing.T) {
	origGhBin := ghBin
	ghBin = ""
	t.Cleanup(func() { ghBin = origGhBin })

	err := Execute([]string{"pr", "list"})
	if err == nil {
		t.Error("expected error when ghBin is empty")
	}
	if !strings.Contains(err.Error(), "not found in PATH") {
		t.Errorf("expected 'not found in PATH' error, got: %v", err)
	}
}

func TestExecuteWithOutputGHBinEmpty(t *testing.T) {
	origGhBin := ghBin
	ghBin = ""
	t.Cleanup(func() { ghBin = origGhBin })

	_, err := ExecuteWithOutput([]string{"pr", "list"})
	if err == nil {
		t.Error("expected error when ghBin is empty")
	}
	if !strings.Contains(err.Error(), "not found in PATH") {
		t.Errorf("expected 'not found in PATH' error, got: %v", err)
	}
}

func TestExecuteWithOutputMockGH(t *testing.T) {
	tmpDir := t.TempDir()
	mockGh := filepath.Join(tmpDir, "gh")
	mockGhContent := "#!/bin/sh\necho \"mock-gh-output\"\n"
	if err := os.WriteFile(mockGh, []byte(mockGhContent), 0755); err != nil {
		t.Fatal(err)
	}

	origGhBin := ghBin
	ghBin = mockGh
	t.Cleanup(func() { ghBin = origGhBin })

	output, err := ExecuteWithOutput([]string{"pr", "list"})
	if err != nil {
		t.Errorf("ExecuteWithOutput() should not error: %v", err)
	}
	if output != "mock-gh-output\n" {
		t.Errorf("expected 'mock-gh-output\\n', got %q", output)
	}
}

func TestGhBinNotFound(t *testing.T) {
	origLookPath := lookPath
	lookPath = func(_ string) (string, error) {
		return "", fmt.Errorf("not found")
	}
	t.Cleanup(func() { lookPath = origLookPath })

	origGhBin := ghBin
	ghBin = ""
	t.Cleanup(func() { ghBin = origGhBin })

	if IsGHAvailable() {
		t.Error("expected gh not available when lookPath fails")
	}
	if GHPath() != "" {
		t.Errorf("expected empty GHPath, got %q", GHPath())
	}
}

func TestExecuteWithNilArgs(t *testing.T) {
	tmpDir := t.TempDir()
	mockGh := filepath.Join(tmpDir, "gh")
	mockGhContent := "#!/bin/sh\n"
	if err := os.WriteFile(mockGh, []byte(mockGhContent), 0755); err != nil {
		t.Fatal(err)
	}

	origGhBin := ghBin
	ghBin = mockGh
	t.Cleanup(func() { ghBin = origGhBin })

	err := Execute(nil)
	if err != nil {
		t.Errorf("Execute(nil) should not error: %v", err)
	}
}

func TestExecuteStderrPassthrough(t *testing.T) {
	tmpDir := t.TempDir()
	mockGh := filepath.Join(tmpDir, "gh")
	mockGhContent := "#!/bin/sh\necho \"stderr-output\" >&2\n"
	if err := os.WriteFile(mockGh, []byte(mockGhContent), 0755); err != nil {
		t.Fatal(err)
	}

	origGhBin := ghBin
	ghBin = mockGh
	t.Cleanup(func() { ghBin = origGhBin })

	oldStderr := os.Stderr
	er, ew, _ := os.Pipe()
	os.Stderr = ew

	err := Execute([]string{"pr", "list"})

	ew.Close()
	os.Stderr = oldStderr
	out, _ := io.ReadAll(er)
	er.Close()

	if err != nil {
		t.Errorf("Execute() with mock gh should not error: %v", err)
	}
	if !strings.Contains(string(out), "stderr-output") {
		t.Errorf("expected mock gh stderr output, got: %s", string(out))
	}
}

func TestExecuteStdinPassthrough(t *testing.T) {
	tmpDir := t.TempDir()
	mockGh := filepath.Join(tmpDir, "gh")
	mockGhContent := "#!/bin/sh\ncat\n"
	if err := os.WriteFile(mockGh, []byte(mockGhContent), 0755); err != nil {
		t.Fatal(err)
	}

	origGhBin := ghBin
	ghBin = mockGh
	t.Cleanup(func() { ghBin = origGhBin })

	oldStdin := os.Stdin
	oldStdout := os.Stdout
	inr, inw, _ := os.Pipe()
	outr, outw, _ := os.Pipe()
	os.Stdin = inr
	os.Stdout = outw

	inw.Write([]byte("hello-input\n"))
	inw.Close()

	errCh := make(chan error, 1)
	go func() {
		err := Execute([]string{"pr", "list"})
		outw.Close()
		errCh <- err
	}()

	err := <-errCh
	os.Stdin = oldStdin
	os.Stdout = oldStdout
	out, _ := io.ReadAll(outr)
	outr.Close()

	if err != nil {
		t.Errorf("Execute() with mock gh should not error: %v", err)
	}
	if !strings.Contains(string(out), "hello-input") {
		t.Errorf("expected mock gh to echo stdin input, got: %s", string(out))
	}
}
