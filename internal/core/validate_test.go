package core

import "testing"

func TestLooksLikeMarkdownLink(t *testing.T) {
	cases := []struct {
		value string
		want  bool
	}{
		// The exact class of value that broke Orbit's first Vercel deploy:
		// a URL copied out of rendered markdown docs, brackets and all.
		{"[Read the docs](https://example.com/docs)", true},
		{"[Vercel Postgres](https://vercel.com/docs/storage/vercel-postgres)", true},
		{"https://example.com", false},
		{"postgres://user:pass@host:5432/db", false},
		{"just a plain string", false},
		{"", false},
	}
	for _, c := range cases {
		if got := LooksLikeMarkdownLink(c.value); got != c.want {
			t.Errorf("LooksLikeMarkdownLink(%q) = %v, want %v", c.value, got, c.want)
		}
	}
}

func TestValidateShape_URL(t *testing.T) {
	if _, ok := ValidateShape(ShapeURL, "https://example.com"); !ok {
		t.Error("expected valid https URL to pass")
	}
	if _, ok := ValidateShape(ShapeURL, "postgres://user:pass@host:5432/db"); !ok {
		t.Error("expected valid postgres URL to pass")
	}
	if _, ok := ValidateShape(ShapeURL, "not a url"); ok {
		t.Error("expected plain string to fail URL validation")
	}
	if _, ok := ValidateShape(ShapeURL, "[docs](https://example.com)"); ok {
		t.Error("expected markdown link to fail URL validation")
	}
}

func TestValidateShape_Port(t *testing.T) {
	if _, ok := ValidateShape(ShapePort, "8080"); !ok {
		t.Error("expected 8080 to be a valid port")
	}
	if _, ok := ValidateShape(ShapePort, "0"); ok {
		t.Error("expected 0 to be an invalid port")
	}
	if _, ok := ValidateShape(ShapePort, "70000"); ok {
		t.Error("expected 70000 to be an invalid port")
	}
	if _, ok := ValidateShape(ShapePort, "not-a-number"); ok {
		t.Error("expected non-numeric string to be invalid port")
	}
}

func TestValidateShape_Number(t *testing.T) {
	if _, ok := ValidateShape(ShapeNumber, "42"); !ok {
		t.Error("expected 42 to be valid")
	}
	if _, ok := ValidateShape(ShapeNumber, "3.14"); !ok {
		t.Error("expected 3.14 to be valid")
	}
	if _, ok := ValidateShape(ShapeNumber, "not-a-number"); ok {
		t.Error("expected non-numeric string to be invalid")
	}
}

func TestValidateShape_Bool(t *testing.T) {
	for _, v := range []string{"true", "false", "1", "0", "yes", "no", "TRUE"} {
		if _, ok := ValidateShape(ShapeBool, v); !ok {
			t.Errorf("expected %q to be a valid bool", v)
		}
	}
	if _, ok := ValidateShape(ShapeBool, "maybe"); ok {
		t.Error("expected 'maybe' to be invalid bool")
	}
}

func TestValidateShape_EmptyValueAlwaysInvalid(t *testing.T) {
	for _, s := range []Shape{ShapeString, ShapeURL, ShapePort, ShapeNumber, ShapeBool} {
		if _, ok := ValidateShape(s, ""); ok {
			t.Errorf("expected empty value to be invalid for shape %s", s)
		}
	}
}

func TestValidateShape_StringAcceptsAnyNonEmptyValue(t *testing.T) {
	if _, ok := ValidateShape(ShapeString, "literally anything"); !ok {
		t.Error("expected plain string shape to accept any non-empty value")
	}
}
