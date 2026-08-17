package core

import (
	"strings"
	"testing"
)

func TestParseEnvExample_InfersShapesFromKeyNames(t *testing.T) {
	input := `
DATABASE_URL=
API_PORT=
IS_PRODUCTION=
REQUEST_TIMEOUT=
API_KEY=
`
	vars, err := ParseEnvExample(strings.NewReader(input), ".env.example")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := map[string]Shape{
		"DATABASE_URL":    ShapeURL,
		"API_PORT":        ShapePort,
		"IS_PRODUCTION":   ShapeBool,
		"REQUEST_TIMEOUT": ShapeNumber,
		"API_KEY":         ShapeString,
	}
	if len(vars) != len(want) {
		t.Fatalf("got %d vars, want %d: %+v", len(vars), len(want), vars)
	}
	for _, v := range vars {
		if want[v.Key] != v.Shape {
			t.Errorf("key %s: got shape %s, want %s", v.Key, v.Shape, want[v.Key])
		}
	}
}

func TestParseEnvExample_ExplicitHintOverridesInference(t *testing.T) {
	input := `
# hint:string
API_PORT=
`
	vars, err := ParseEnvExample(strings.NewReader(input), ".env.example")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vars) != 1 || vars[0].Shape != ShapeString {
		t.Fatalf("expected explicit hint to override inference, got %+v", vars)
	}
}

func TestParseEnvExample_InlineHint(t *testing.T) {
	input := `WEBHOOK_TARGET=  # hint:url`
	vars, err := ParseEnvExample(strings.NewReader(input), ".env.example")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vars) != 1 || vars[0].Shape != ShapeURL {
		t.Fatalf("expected inline hint to set url shape, got %+v", vars)
	}
}

func TestParseEnvExample_RejectsMalformedLine(t *testing.T) {
	input := "THIS_IS_NOT_VALID"
	_, err := ParseEnvExample(strings.NewReader(input), ".env.example")
	if err == nil {
		t.Fatal("expected error for malformed line, got nil")
	}
	var pe *ParseError
	if !isParseError(err, &pe) {
		t.Fatalf("expected *ParseError, got %T: %v", err, err)
	}
	if pe.Line != 1 {
		t.Errorf("expected error on line 1, got line %d", pe.Line)
	}
}

func TestParseEnvFile_HandlesQuotesAndComments(t *testing.T) {
	input := `
DATABASE_URL="postgres://user:pass@host:5432/db"  # primary db
API_KEY=raw-unquoted-value
# a full-line comment
EMPTY_LINE_ABOVE=1
`
	vals, err := ParseEnvFile(strings.NewReader(input), ".env")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vals["DATABASE_URL"] != "postgres://user:pass@host:5432/db" {
		t.Errorf("got DATABASE_URL=%q", vals["DATABASE_URL"])
	}
	if vals["API_KEY"] != "raw-unquoted-value" {
		t.Errorf("got API_KEY=%q", vals["API_KEY"])
	}
	if vals["EMPTY_LINE_ABOVE"] != "1" {
		t.Errorf("got EMPTY_LINE_ABOVE=%q", vals["EMPTY_LINE_ABOVE"])
	}
}

func TestParseEnvFile_PreservesHashInsideQuotes(t *testing.T) {
	input := `SECRET="value#with#hashes"`
	vals, err := ParseEnvFile(strings.NewReader(input), ".env")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vals["SECRET"] != "value#with#hashes" {
		t.Errorf("got SECRET=%q, want value#with#hashes", vals["SECRET"])
	}
}

func TestParseEnvFile_LastDuplicateWins(t *testing.T) {
	input := "KEY=first\nKEY=second\n"
	vals, err := ParseEnvFile(strings.NewReader(input), ".env")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vals["KEY"] != "second" {
		t.Errorf("got KEY=%q, want second", vals["KEY"])
	}
}

// isParseError is a small helper so the test above can assert the
// concrete error type without importing errors.As boilerplate at every
// call site.
func isParseError(err error, target **ParseError) bool {
	pe, ok := err.(*ParseError)
	if ok {
		*target = pe
	}
	return ok
}
