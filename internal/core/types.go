// Package core holds Relay's platform-independent logic: parsing env files,
// validating values against inferred/declared shapes, and diffing expected
// vs. local vs. remote configuration. Nothing in this package touches the
// filesystem, the network, or os.Args — that all lives in the adapters.
package core

// Shape describes the kind of value a variable is expected to hold, either
// declared explicitly via a "# hint:<shape>" comment in .env.example or
// inferred from the variable's name.
type Shape string

const (
	ShapeString Shape = "string"
	ShapeURL    Shape = "url"
	ShapePort   Shape = "port"
	ShapeNumber Shape = "number"
	ShapeBool   Shape = "bool"
)

// ExpectedVar is one entry parsed out of .env.example: a key Relay expects
// to see configured, along with the shape its value should take.
type ExpectedVar struct {
	Key   string
	Shape Shape
}

// Severity classifies how serious a reported Issue is.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// IssueKind identifies which category of problem an Issue represents.
type IssueKind string

const (
	IssueMissingLocal  IssueKind = "missing_local"  // in .env.example, absent from .env
	IssueMissingRemote IssueKind = "missing_remote" // in .env.example, absent from platform
	IssueExtraRemote   IssueKind = "extra_remote"   // on platform, not in .env.example
	IssueValueMismatch IssueKind = "value_mismatch" // local value differs from remote value
	IssueInvalidShape  IssueKind = "invalid_shape"  // value doesn't match its expected shape
	IssueMarkdownLeak  IssueKind = "markdown_leak"  // value looks like a pasted markdown link
)

// Issue is a single finding produced by the diff engine.
type Issue struct {
	Kind     IssueKind
	Severity Severity
	Key      string
	Detail   string
}

// Report is the full result of comparing expected, local, and remote env
// state. Empty Issues means everything checked out.
type Report struct {
	Issues []Issue
}

// HasErrors reports whether the report contains any error-severity issue,
// which callers (e.g. the CLI) can use to decide on a nonzero exit code.
func (r Report) HasErrors() bool {
	for _, iss := range r.Issues {
		if iss.Severity == SeverityError {
			return true
		}
	}
	return false
}
