package core

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// markdownLinkPattern matches a markdown-style link, e.g. "[text](https://...)".
// This is the exact shape of the bug that broke Orbit's first Vercel deploy:
// a URL copied out of a rendered markdown doc, brackets and all, pasted
// straight into an env var value.
var markdownLinkPattern = regexp.MustCompile(`\[[^\]]*\]\([^)]+\)`)

// LooksLikeMarkdownLink reports whether value appears to be a markdown link
// that was pasted in place of a raw value.
func LooksLikeMarkdownLink(value string) bool {
	return markdownLinkPattern.MatchString(value)
}

// ValidateShape checks whether value is well-formed for the given Shape.
// It returns ("", true) when the value is valid, or a human-readable reason
// and false when it is not.
func ValidateShape(shape Shape, value string) (reason string, ok bool) {
	if value == "" {
		return "value is empty", false
	}

	switch shape {
	case ShapeURL:
		u, err := url.Parse(value)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return "does not look like a valid absolute URL (missing scheme/host)", false
		}
		return "", true

	case ShapePort:
		n, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return "not a valid integer port number", false
		}
		if n < 1 || n > 65535 {
			return "port out of valid range (1-65535)", false
		}
		return "", true

	case ShapeNumber:
		if _, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err != nil {
			return "not a valid number", false
		}
		return "", true

	case ShapeBool:
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "true", "false", "1", "0", "yes", "no":
			return "", true
		default:
			return "not a recognizable boolean (true/false/1/0/yes/no)", false
		}

	case ShapeString, "":
		return "", true

	default:
		return "", true
	}
}
