package tile

import (
	"fmt"
	"strings"
)

func norm(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

// ParsePaper parses a paper size ("A4" or "A3", case-insensitive).
func ParsePaper(s string) (Paper, error) {
	switch norm(s) {
	case "a4":
		return A4, nil
	case "a3":
		return A3, nil
	}
	return A4, fmt.Errorf("invalid paper size %q (want A4 or A3)", s)
}

// ParseBrushing parses a brushing direction ("up"/"upwards", "down"/"downwards").
func ParseBrushing(s string) (Brushing, error) {
	switch norm(s) {
	case "down", "downward", "downwards":
		return Downwards, nil
	case "up", "upward", "upwards":
		return Upwards, nil
	}
	return Downwards, fmt.Errorf("invalid brushing %q (want up or down)", s)
}

// ParseToggle parses an on/off value ("on"/"off", "true"/"false", "yes"/"no",
// "1"/"0"). It exists so boolean-style flags accept a space-separated value like
// every other flag (e.g. "--labels off"), avoiding the Go flag bool footgun.
func ParseToggle(s string) (bool, error) {
	switch norm(s) {
	case "on", "true", "yes", "1":
		return true, nil
	case "off", "false", "no", "0":
		return false, nil
	}
	return false, fmt.Errorf("invalid on/off value %q (want on or off)", s)
}

// ParsePasting parses a pasting order ("left"/"from-left", "right"/"from-right").
func ParsePasting(s string) (Pasting, error) {
	switch norm(s) {
	case "left", "from-left", "from left", "fromleft":
		return FromLeft, nil
	case "right", "from-right", "from right", "fromright":
		return FromRight, nil
	}
	return FromLeft, fmt.Errorf("invalid pasting order %q (want left or right)", s)
}
