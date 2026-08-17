package core

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// ParseError reports a problem with a specific line of an env file, so the
// caller can point a user at exactly what to fix.
type ParseError struct {
	Line   int
	Source string
	Reason string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("%s:%d: %s", e.Source, e.Line, e.Reason)
}

// ParseEnvExample reads a .env.example file and returns the set of
// variables it declares, in the order they appear. A hint for a variable's
// expected Shape may be given via a trailing or preceding comment of the
// form "# hint:<shape>" (e.g. "# hint:url"); otherwise the shape is
// inferred from the key name.
func ParseEnvExample(r io.Reader, source string) ([]ExpectedVar, error) {
	scanner := bufio.NewScanner(r)
	var vars []ExpectedVar
	pendingHint := Shape("")

	lineNo := 0
	for scanner.Scan() {
		lineNo++
		raw := scanner.Text()
		trimmed := strings.TrimSpace(raw)

		if trimmed == "" {
			pendingHint = ""
			continue
		}

		if strings.HasPrefix(trimmed, "#") {
			if h, ok := parseHintComment(trimmed); ok {
				pendingHint = h
			}
			continue
		}

		key, value, ok := splitKeyValue(trimmed)
		if !ok {
			return nil, &ParseError{Line: lineNo, Source: source, Reason: fmt.Sprintf("malformed line (expected KEY=value): %q", raw)}
		}
		if key == "" {
			return nil, &ParseError{Line: lineNo, Source: source, Reason: "empty variable name"}
		}

		shape := pendingHint
		if trailing, ok := parseHintComment(inlineComment(value)); ok {
			shape = trailing
		}
		if shape == "" {
			shape = inferShape(key)
		}

		vars = append(vars, ExpectedVar{Key: key, Shape: shape})
		pendingHint = ""
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading %s: %w", source, err)
	}
	return vars, nil
}

// ParseEnvFile reads a .env-style file (real values, not a template) and
// returns its key/value pairs. Later duplicate keys overwrite earlier ones,
// matching how most env-loading tools behave.
func ParseEnvFile(r io.Reader, source string) (map[string]string, error) {
	scanner := bufio.NewScanner(r)
	values := make(map[string]string)

	lineNo := 0
	for scanner.Scan() {
		lineNo++
		raw := scanner.Text()
		trimmed := strings.TrimSpace(raw)

		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		key, rawValue, ok := splitKeyValue(trimmed)
		if !ok {
			return nil, &ParseError{Line: lineNo, Source: source, Reason: fmt.Sprintf("malformed line (expected KEY=value): %q", raw)}
		}
		if key == "" {
			return nil, &ParseError{Line: lineNo, Source: source, Reason: "empty variable name"}
		}

		value := stripInlineComment(rawValue)
		value = unquote(strings.TrimSpace(value))
		values[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading %s: %w", source, err)
	}
	return values, nil
}

// splitKeyValue splits "KEY=value" (optionally prefixed with "export ")
// into its key and raw (unprocessed) value. ok is false if the line has no
// '=' at all, i.e. it isn't a valid assignment.
func splitKeyValue(line string) (key, value string, ok bool) {
	line = strings.TrimPrefix(line, "export ")
	idx := strings.Index(line, "=")
	if idx < 0 {
		return "", "", false
	}
	key = strings.TrimSpace(line[:idx])
	value = line[idx+1:]
	return key, value, true
}

// inlineComment extracts the "# ..." portion trailing a value, if any, so
// hint comments placed on the same line as an assignment can be found.
func inlineComment(value string) string {
	idx := strings.Index(value, "#")
	if idx < 0 {
		return ""
	}
	return strings.TrimSpace(value[idx:])
}

// stripInlineComment removes a trailing "# ..." comment from a raw value,
// but only when the '#' is not inside a quoted string.
func stripInlineComment(value string) string {
	inSingle, inDouble := false, false
	for i, r := range value {
		switch r {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case '#':
			if !inSingle && !inDouble && (i == 0 || value[i-1] == ' ' || value[i-1] == '\t') {
				return value[:i]
			}
		}
	}
	return value
}

// unquote strips a single matching pair of surrounding quotes, if present.
func unquote(value string) string {
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
}

// parseHintComment recognizes comments of the form "# hint:<shape>" and
// returns the declared Shape.
func parseHintComment(comment string) (Shape, bool) {
	comment = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(comment), "#"))
	const prefix = "hint:"
	if !strings.HasPrefix(comment, prefix) {
		return "", false
	}
	shape := Shape(strings.TrimSpace(strings.TrimPrefix(comment, prefix)))
	switch shape {
	case ShapeString, ShapeURL, ShapePort, ShapeNumber, ShapeBool:
		return shape, true
	default:
		return "", false
	}
}

// inferShape guesses a variable's expected Shape from its name when no
// explicit hint comment is given, using common naming conventions.
func inferShape(key string) Shape {
	upper := strings.ToUpper(key)
	switch {
	case strings.HasSuffix(upper, "_URL") || strings.HasSuffix(upper, "_URI") || upper == "URL" || upper == "URI":
		return ShapeURL
	case strings.HasSuffix(upper, "_PORT") || upper == "PORT":
		return ShapePort
	case strings.HasPrefix(upper, "IS_") || strings.HasPrefix(upper, "HAS_") || strings.HasSuffix(upper, "_ENABLED") || strings.HasSuffix(upper, "_DISABLED"):
		return ShapeBool
	case strings.HasSuffix(upper, "_COUNT") || strings.HasSuffix(upper, "_TIMEOUT") || strings.HasSuffix(upper, "_TTL") || strings.HasSuffix(upper, "_LIMIT"):
		return ShapeNumber
	default:
		return ShapeString
	}
}
