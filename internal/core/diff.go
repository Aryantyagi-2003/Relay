package core

import "fmt"

// Diff compares the variables declared in .env.example (expected) against
// what's actually configured locally, and optionally against what's live
// on the deploy platform. remote may be nil when the caller only wants to
// check the local file (e.g. no platform credentials configured yet) — in
// that case remote-related issues are simply skipped.
func Diff(expected []ExpectedVar, local map[string]string, remote map[string]string) Report {
	var issues []Issue

	for _, ev := range expected {
		localVal, hasLocal := local[ev.Key]

		if !hasLocal {
			issues = append(issues, Issue{
				Kind:     IssueMissingLocal,
				Severity: SeverityWarning,
				Key:      ev.Key,
				Detail:   "declared in .env.example but not set locally",
			})
		} else {
			issues = append(issues, checkValue(ev.Key, ev.Shape, localVal, "local")...)
		}

		if remote == nil {
			continue
		}

		remoteVal, hasRemote := remote[ev.Key]
		if !hasRemote {
			issues = append(issues, Issue{
				Kind:     IssueMissingRemote,
				Severity: SeverityError,
				Key:      ev.Key,
				Detail:   "declared in .env.example but not configured on the deploy platform",
			})
			continue
		}

		issues = append(issues, checkValue(ev.Key, ev.Shape, remoteVal, "remote")...)

		if hasLocal && localVal != remoteVal {
			issues = append(issues, Issue{
				Kind:     IssueValueMismatch,
				Severity: SeverityWarning,
				Key:      ev.Key,
				Detail:   "local value differs from the value configured on the deploy platform",
			})
		}
	}

	if remote != nil {
		expectedKeys := make(map[string]bool, len(expected))
		for _, ev := range expected {
			expectedKeys[ev.Key] = true
		}
		for key := range remote {
			if !expectedKeys[key] {
				issues = append(issues, Issue{
					Kind:     IssueExtraRemote,
					Severity: SeverityWarning,
					Key:      key,
					Detail:   "configured on the deploy platform but not declared in .env.example",
				})
			}
		}
	}

	return Report{Issues: issues}
}

// checkValue runs shape validation and markdown-link detection for a single
// key/value pair, tagging any resulting issue with where the value came
// from (e.g. "local" or "remote").
func checkValue(key string, shape Shape, value string, origin string) []Issue {
	var issues []Issue

	if LooksLikeMarkdownLink(value) {
		// A markdown-link leak is itself the actionable problem; running
		// shape validation on top of it would just restate the same fault
		// (e.g. "not a valid URL") in a second, less specific issue.
		return append(issues, Issue{
			Kind:     IssueMarkdownLeak,
			Severity: SeverityError,
			Key:      key,
			Detail:   fmt.Sprintf("%s value looks like a pasted markdown link (e.g. \"[text](url)\"), not a raw value", origin),
		})
	}

	if reason, ok := ValidateShape(shape, value); !ok {
		issues = append(issues, Issue{
			Kind:     IssueInvalidShape,
			Severity: SeverityError,
			Key:      key,
			Detail:   fmt.Sprintf("%s value invalid for expected type %q: %s", origin, shape, reason),
		})
	}

	return issues
}
