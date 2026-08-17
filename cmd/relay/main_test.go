package main

import (
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
