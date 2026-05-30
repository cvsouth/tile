package tiler

import "testing"

func TestParsePaper(t *testing.T) {
	for _, s := range []string{"A4", "a4", " A4 "} {
		if p, err := ParsePaper(s); err != nil || p != A4 {
			t.Errorf("ParsePaper(%q) = %v, %v", s, p, err)
		}
	}
	if p, err := ParsePaper("a3"); err != nil || p != A3 {
		t.Errorf("ParsePaper(a3) = %v, %v", p, err)
	}
	if _, err := ParsePaper("A5"); err == nil {
		t.Error("expected error for A5")
	}
}

func TestParseBrushing(t *testing.T) {
	for _, s := range []string{"down", "downwards", "DOWN"} {
		if b, err := ParseBrushing(s); err != nil || b != Downwards {
			t.Errorf("ParseBrushing(%q) = %v, %v", s, b, err)
		}
	}
	if b, err := ParseBrushing("up"); err != nil || b != Upwards {
		t.Errorf("ParseBrushing(up) = %v, %v", b, err)
	}
	if _, err := ParseBrushing("sideways"); err == nil {
		t.Error("expected error for sideways")
	}
}

func TestParsePasting(t *testing.T) {
	for _, s := range []string{"left", "from-left", "from left"} {
		if p, err := ParsePasting(s); err != nil || p != FromLeft {
			t.Errorf("ParsePasting(%q) = %v, %v", s, p, err)
		}
	}
	if p, err := ParsePasting("right"); err != nil || p != FromRight {
		t.Errorf("ParsePasting(right) = %v, %v", p, err)
	}
	if _, err := ParsePasting("diagonal"); err == nil {
		t.Error("expected error for diagonal")
	}
}

func TestParseToggle(t *testing.T) {
	on := []string{"on", "true", "yes", "1", "ON", " on "}
	off := []string{"off", "false", "no", "0", "OFF"}
	for _, s := range on {
		if v, err := ParseToggle(s); err != nil || !v {
			t.Errorf("ParseToggle(%q) = %v, %v; want true", s, v, err)
		}
	}
	for _, s := range off {
		if v, err := ParseToggle(s); err != nil || v {
			t.Errorf("ParseToggle(%q) = %v, %v; want false", s, v, err)
		}
	}
	if _, err := ParseToggle("maybe"); err == nil {
		t.Error("expected error for maybe")
	}
}

func TestStringRoundTrips(t *testing.T) {
	if A4.String() != "A4" || A3.String() != "A3" {
		t.Error("paper String")
	}
	if Downwards.String() != "downwards" || Upwards.String() != "upwards" {
		t.Error("brushing String")
	}
	if FromLeft.String() != "from left" || FromRight.String() != "from right" {
		t.Error("pasting String")
	}
	if Landscape.String() != "landscape" || Portrait.String() != "portrait" {
		t.Error("orientation String")
	}
}
