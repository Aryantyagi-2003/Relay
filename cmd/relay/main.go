// Command relay diffs a project's .env.example against what's actually
// configured locally and (optionally) on Vercel, flagging missing
// variables and values that don't look right — e.g. a URL pasted in as a
// markdown link — before they break a deploy.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"sort"

	"github.com/Aryantyagi-2003/Relay/internal/core"
	"github.com/Aryantyagi-2003/Relay/internal/vercel"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

const (
	exitOK     = 0
	exitIssues = 1
	exitUsage  = 2
)

// version is set at build time via -ldflags "-X main.version=vX.Y.Z" by
// the release workflow. Left empty for `go build`/`go run`, in which case
// versionString falls back to module version info that `go install
// pkg@version` embeds automatically.
var version string

func versionString() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("relay", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: relay [flags]\n\n")
		fmt.Fprintf(stderr, "Diffs .env.example against your local .env and (optionally) a Vercel project's\nconfigured environment variables.\n\nFlags:\n")
		fs.PrintDefaults()
		fmt.Fprintf(stderr, "\nExit codes:\n  0  no issues (or only warnings)\n  1  at least one error-level issue found\n  2  usage error (bad flags, missing files, API failure)\n")
	}

	examplePath := fs.String("example", ".env.example", "path to the .env.example template")
	envPath := fs.String("env", ".env", "path to the local .env file")
	project := fs.String("project", "", "Vercel project ID or name (enables remote checks)")
	token := fs.String("token", os.Getenv("VERCEL_TOKEN"), "Vercel API token (defaults to $VERCEL_TOKEN)")
	teamID := fs.String("team", os.Getenv("VERCEL_TEAM_ID"), "Vercel team ID, for team-owned projects (defaults to $VERCEL_TEAM_ID)")
	target := fs.String("target", "production", "deploy target to check against: production, preview, or development")
	showVersion := fs.Bool("version", false, "print the relay version and exit")

	if err := fs.Parse(args); err != nil {
		return exitUsage
	}

	if *showVersion {
		fmt.Fprintf(stdout, "relay %s\n", versionString())
		return exitOK
	}

	expected, err := loadExpected(*examplePath)
	if err != nil {
		fmt.Fprintf(stderr, "relay: %v\n", err)
		return exitUsage
	}

	local, err := loadLocal(*envPath, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "relay: %v\n", err)
		return exitUsage
	}

	var remote map[string]string
	var redactedKeys []string
	if *project != "" {
		if *token == "" {
			fmt.Fprintf(stderr, "relay: --project was given but no Vercel API token was found (pass --token or set $VERCEL_TOKEN)\n")
			return exitUsage
		}
		client, err := vercel.NewClient(*token)
		if err != nil {
			fmt.Fprintf(stderr, "relay: %v\n", err)
			return exitUsage
		}
		client.TeamID = *teamID

		remote, redactedKeys, err = client.FetchEnv(context.Background(), *project, *target)
		if err != nil {
			fmt.Fprintf(stderr, "relay: fetching Vercel env for project %q (%s): %v\n", *project, *target, err)
			return exitUsage
		}
	}

	report := buildReport(expected, local, remote, redactedKeys)
	printReport(stdout, report, *project != "")

	if report.HasErrors() {
		return exitIssues
	}
	return exitOK
}

func loadExpected(path string) ([]core.ExpectedVar, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%s not found — Relay needs a .env.example to know what variables are expected (use --example to point at a different path)", path)
		}
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	expected, err := core.ParseEnvExample(f, path)
	if err != nil {
		return nil, err
	}
	if len(expected) == 0 {
		return nil, fmt.Errorf("%s declares no variables — nothing to check", path)
	}
	return expected, nil
}

func loadLocal(path string, stderr io.Writer) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(stderr, "relay: note: %s not found, treating local config as empty\n", path)
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	return core.ParseEnvFile(f, path)
}

// buildReport runs the core diff and layers on adapter-specific findings
// (redacted remote values) that core.Diff has no platform concept of.
//
// Variables Vercel reports as present-but-redacted (sensitive) are pulled
// out of the remote-comparison pass entirely: core.Diff has no way to know
// they exist on the platform (they're absent from remote precisely
// because their value was withheld), so running the full expected list
// through it would flag them as missing_remote — contradicting the
// separate remote_redacted issue that correctly says they *are* present.
// Those keys still get a local-only pass (missing/invalid-shape/markdown
// checks against .env), just no remote presence/value comparison.
func buildReport(expected []core.ExpectedVar, local, remote map[string]string, redactedKeys []string) core.Report {
	redacted := make(map[string]bool, len(redactedKeys))
	for _, key := range redactedKeys {
		redacted[key] = true
	}

	var normal, redactedExpected []core.ExpectedVar
	for _, ev := range expected {
		if redacted[ev.Key] {
			redactedExpected = append(redactedExpected, ev)
		} else {
			normal = append(normal, ev)
		}
	}

	report := core.Diff(normal, local, remote)

	if len(redactedExpected) > 0 {
		localOnly := core.Diff(redactedExpected, local, nil)
		report.Issues = append(report.Issues, localOnly.Issues...)
		for _, ev := range redactedExpected {
			report.Issues = append(report.Issues, core.Issue{
				Kind:     core.IssueRemoteRedacted,
				Severity: core.SeverityWarning,
				Key:      ev.Key,
				Detail:   "present on Vercel but marked sensitive; its value could not be retrieved to verify",
			})
		}
	}

	return report
}

func printReport(w io.Writer, report core.Report, checkedRemote bool) {
	if len(report.Issues) == 0 {
		fmt.Fprintln(w, "relay: no issues found — local config matches .env.example"+remoteSuffix(checkedRemote))
		return
	}

	sorted := append([]core.Issue(nil), report.Issues...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Severity != sorted[j].Severity {
			return sorted[i].Severity == core.SeverityError
		}
		return sorted[i].Key < sorted[j].Key
	})

	var errors, warnings int
	for _, iss := range sorted {
		label := "WARNING"
		if iss.Severity == core.SeverityError {
			label = "ERROR"
			errors++
		} else {
			warnings++
		}
		fmt.Fprintf(w, "[%s] %s: %s (%s)\n", label, iss.Key, iss.Detail, iss.Kind)
	}
	fmt.Fprintf(w, "\n%d error(s), %d warning(s)\n", errors, warnings)
}

func remoteSuffix(checkedRemote bool) string {
	if checkedRemote {
		return " and Vercel"
	}
	return ""
}
