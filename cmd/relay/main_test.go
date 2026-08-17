package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Aryantyagi-2003/Relay/internal/core"
)

func hasIssueKind(issues []core.Issue, kind core.IssueKind, key string) bool {
	for _, iss := range issues {
		if iss.Kind == kind && iss.Key == key {
			return true
		}
	}
	return false
}

// TestBuildReport_RedactedKeyNotFlaggedMissingRemote reproduces the bug
// found when running against a real Vercel project: every sensitive env
// var was reported as both missing_remote (absent from the platform) and
// remote_redacted (present but unreadable) — a contradiction, since a
// redacted key is by definition present. A key Vercel marks sensitive
// must get exactly the redacted issue, never missing_remote.
func TestBuildReport_RedactedKeyNotFlaggedMissingRemote(t *testing.T) {
	expected := []core.ExpectedVar{
		{Key: "DATABASE_URL", Shape: core.ShapeURL},
		{Key: "API_SECRET", Shape: core.ShapeString},
	}
	local := map[string]string{
		"DATABASE_URL": "postgres://user:pass@host:5432/db",
		"API_SECRET":   "shh",
	}
	remote := map[string]string{
		"DATABASE_URL": "postgres://user:pass@host:5432/db",
		// API_SECRET intentionally absent: Vercel withheld its value.
	}
	redactedKeys := []string{"API_SECRET"}

	report := buildReport(expected, local, remote, redactedKeys)

	if hasIssueKind(report.Issues, core.IssueMissingRemote, "API_SECRET") {
		t.Errorf("redacted key must not also be reported missing_remote; issues: %+v", report.Issues)
	}
	if !hasIssueKind(report.Issues, core.IssueRemoteRedacted, "API_SECRET") {
		t.Errorf("expected remote_redacted issue for API_SECRET; issues: %+v", report.Issues)
	}
	if hasIssueKind(report.Issues, core.IssueMissingRemote, "DATABASE_URL") {
		t.Errorf("non-redacted key incorrectly flagged missing_remote; issues: %+v", report.Issues)
	}
}

// TestBuildReport_RedactedKeyStillGetsLocalChecks ensures pulling a key
// out of the remote comparison doesn't also skip checks that only need
// the local value (e.g. catching it if it's missing locally, or itself a
// pasted markdown link).
func TestBuildReport_RedactedKeyStillGetsLocalChecks(t *testing.T) {
	expected := []core.ExpectedVar{{Key: "API_SECRET", Shape: core.ShapeString}}
	local := map[string]string{} // not set locally
	remote := map[string]string{}
	redactedKeys := []string{"API_SECRET"}

	report := buildReport(expected, local, remote, redactedKeys)

	if !hasIssueKind(report.Issues, core.IssueMissingLocal, "API_SECRET") {
		t.Errorf("expected missing_local issue for redacted key absent from .env; issues: %+v", report.Issues)
	}
	if !hasIssueKind(report.Issues, core.IssueRemoteRedacted, "API_SECRET") {
		t.Errorf("expected remote_redacted issue; issues: %+v", report.Issues)
	}
}

func TestRun_VersionFlagPrintsAndExitsZero(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--version"}, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("expected exit 0, got %d (stderr: %s)", code, stderr.String())
	}
	if !strings.HasPrefix(stdout.String(), "relay ") {
		t.Errorf("expected version output to start with %q, got %q", "relay ", stdout.String())
	}
}

func TestRun_MissingExampleFileExitsUsage(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := run([]string{"--example", filepath.Join(dir, "nope.env.example")}, &stdout, &stderr)
	if code != exitUsage {
		t.Fatalf("expected exit %d for missing .env.example, got %d", exitUsage, code)
	}
	if !strings.Contains(stderr.String(), "not found") {
		t.Errorf("expected a clear 'not found' error, got: %s", stderr.String())
	}
}

func TestRun_ErrorIssueExitsWithIssuesCode(t *testing.T) {
	dir := t.TempDir()
	examplePath := filepath.Join(dir, ".env.example")
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(examplePath, []byte("DATABASE_URL=\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(envPath, []byte("DATABASE_URL=\"[docs](https://example.com)\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"--example", examplePath, "--env", envPath}, &stdout, &stderr)
	if code != exitIssues {
		t.Fatalf("expected exit %d for a markdown-link error, got %d (stdout: %s)", exitIssues, code, stdout.String())
	}
	if !strings.Contains(stdout.String(), "markdown_leak") {
		t.Errorf("expected markdown_leak in output, got: %s", stdout.String())
	}
}

func TestRun_CleanConfigExitsOK(t *testing.T) {
	dir := t.TempDir()
	examplePath := filepath.Join(dir, ".env.example")
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(examplePath, []byte("API_KEY=\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(envPath, []byte("API_KEY=abc123\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"--example", examplePath, "--env", envPath}, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("expected exit %d for clean config, got %d (stderr: %s)", exitOK, code, stderr.String())
	}
}
