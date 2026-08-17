package core

import (
	"strings"
	"testing"
)

func hasIssue(issues []Issue, kind IssueKind, key string) bool {
	for _, iss := range issues {
		if iss.Kind == kind && iss.Key == key {
			return true
		}
	}
	return false
}

func TestDiff_MissingLocal(t *testing.T) {
	expected := []ExpectedVar{{Key: "API_KEY", Shape: ShapeString}}
	report := Diff(expected, map[string]string{}, nil)
	if !hasIssue(report.Issues, IssueMissingLocal, "API_KEY") {
		t.Fatalf("expected missing_local issue, got %+v", report.Issues)
	}
	if report.HasErrors() {
		t.Error("missing_local alone should not count as an error (warning only)")
	}
}

func TestDiff_MissingRemote(t *testing.T) {
	expected := []ExpectedVar{{Key: "API_KEY", Shape: ShapeString}}
	local := map[string]string{"API_KEY": "abc123"}
	remote := map[string]string{}
	report := Diff(expected, local, remote)
	if !hasIssue(report.Issues, IssueMissingRemote, "API_KEY") {
		t.Fatalf("expected missing_remote issue, got %+v", report.Issues)
	}
	if !report.HasErrors() {
		t.Error("missing_remote should count as an error")
	}
}

func TestDiff_ExtraRemote(t *testing.T) {
	expected := []ExpectedVar{{Key: "API_KEY", Shape: ShapeString}}
	local := map[string]string{"API_KEY": "abc123"}
	remote := map[string]string{"API_KEY": "abc123", "LEGACY_FLAG": "true"}
	report := Diff(expected, local, remote)
	if !hasIssue(report.Issues, IssueExtraRemote, "LEGACY_FLAG") {
		t.Fatalf("expected extra_remote issue for LEGACY_FLAG, got %+v", report.Issues)
	}
}

func TestDiff_ValueMismatchBetweenLocalAndRemote(t *testing.T) {
	expected := []ExpectedVar{{Key: "API_KEY", Shape: ShapeString}}
	local := map[string]string{"API_KEY": "local-value"}
	remote := map[string]string{"API_KEY": "different-remote-value"}
	report := Diff(expected, local, remote)
	if !hasIssue(report.Issues, IssueValueMismatch, "API_KEY") {
		t.Fatalf("expected value_mismatch issue, got %+v", report.Issues)
	}
}

func TestDiff_InvalidShape(t *testing.T) {
	expected := []ExpectedVar{{Key: "API_PORT", Shape: ShapePort}}
	local := map[string]string{"API_PORT": "not-a-port"}
	report := Diff(expected, local, nil)
	if !hasIssue(report.Issues, IssueInvalidShape, "API_PORT") {
		t.Fatalf("expected invalid_shape issue, got %+v", report.Issues)
	}
	if !report.HasErrors() {
		t.Error("invalid_shape should count as an error")
	}
}

func TestDiff_NoRemoteMeansRemoteChecksSkipped(t *testing.T) {
	expected := []ExpectedVar{{Key: "API_KEY", Shape: ShapeString}}
	local := map[string]string{"API_KEY": "abc123"}
	report := Diff(expected, local, nil)
	for _, iss := range report.Issues {
		if iss.Kind == IssueMissingRemote || iss.Kind == IssueExtraRemote || iss.Kind == IssueValueMismatch {
			t.Errorf("did not expect remote-related issue when remote is nil, got %+v", iss)
		}
	}
}

func TestDiff_CleanConfigProducesNoIssues(t *testing.T) {
	expected := []ExpectedVar{
		{Key: "DATABASE_URL", Shape: ShapeURL},
		{Key: "API_PORT", Shape: ShapePort},
	}
	local := map[string]string{
		"DATABASE_URL": "postgres://user:pass@host:5432/db",
		"API_PORT":     "5432",
	}
	remote := map[string]string{
		"DATABASE_URL": "postgres://user:pass@host:5432/db",
		"API_PORT":     "5432",
	}
	report := Diff(expected, local, remote)
	if len(report.Issues) != 0 {
		t.Fatalf("expected no issues for clean matching config, got %+v", report.Issues)
	}
}

// TestDiff_ReproducesOrbitMarkdownLinkBug is an end-to-end reproduction of
// the actual incident that motivated Relay: a database connection string
// was copied out of rendered markdown docs (brackets and all) and pasted
// straight into DATABASE_URL, which then broke the Vercel build with a
// cryptic error instead of a clear "this isn't a valid URL" message.
func TestDiff_ReproducesOrbitMarkdownLinkBug(t *testing.T) {
	exampleFile := `
DATABASE_URL=
API_PORT=
`
	expected, err := ParseEnvExample(strings.NewReader(exampleFile), ".env.example")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	envFile := `
DATABASE_URL="[Vercel Postgres](https://vercel.com/docs/storage/vercel-postgres/quickstart)"
API_PORT=5432
`
	local, err := ParseEnvFile(strings.NewReader(envFile), ".env")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	report := Diff(expected, local, nil)

	if !hasIssue(report.Issues, IssueMarkdownLeak, "DATABASE_URL") {
		t.Fatalf("expected markdown_leak issue for DATABASE_URL, got %+v", report.Issues)
	}
	if !report.HasErrors() {
		t.Fatal("expected the markdown-link leak to be reported as an error")
	}
	// The malformed value should be caught as a markdown leak specifically,
	// not just a generic invalid-shape failure — that's the whole point:
	// a message that names the actual mistake, not just "invalid URL".
	if hasIssue(report.Issues, IssueInvalidShape, "DATABASE_URL") {
		t.Error("expected markdown_leak to supersede a separate invalid_shape issue for the same value")
	}
}
