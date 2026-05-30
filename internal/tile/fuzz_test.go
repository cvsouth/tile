package tile

import (
	"math"
	"testing"
)

// clampF maps an arbitrary float into [lo, hi], turning non-finite values into
// lo. It keeps the layout fuzzer within sane, finite, non-pathological inputs.
func clampF(v, lo, hi float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// FuzzParsers checks that the option parsers never panic on arbitrary input.
func FuzzParsers(f *testing.F) {
	for _, s := range []string{"A4", "a3", "down", "upwards", "from-left", "right", "on", "off", "", "garbage", "  A4  "} {
		f.Add(s)
	}
	f.Fuzz(func(_ *testing.T, s string) {
		_, _ = ParsePaper(s)
		_, _ = ParseBrushing(s)
		_, _ = ParsePasting(s)
		_, _ = ParseToggle(s)
	})
}

// FuzzDefaultOutputName checks the default-name derivation never returns empty
// and never panics, for any input path.
func FuzzDefaultOutputName(f *testing.F) {
	for _, s := range []string{"photo.jpg", "a.PNG", ".svg", "", "a/b/c.jpeg", "no-ext", "x.tiles.pdf"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, name string) {
		if got := DefaultOutputName(name); got == "" {
			t.Errorf("DefaultOutputName(%q) returned empty", name)
		}
	})
}

// FuzzComputeLayout exercises the layout maths against arbitrary (but bounded)
// inputs: it must never panic, and any successful layout must be internally
// consistent (positive grid, tile count matching cols*rows).
func FuzzComputeLayout(f *testing.F) {
	f.Add(123.0, 15.0, 300.0, 0.75, true)
	f.Add(50.0, 5.0, 96.0, 2.0, false)
	f.Fuzz(func(t *testing.T, widthCM, overlapMM, dpi, aspect float64, a3 bool) {
		o := Options{
			Paper:     A4,
			OverlapMM: clampF(overlapMM, 1, 80),
			WidthCM:   clampF(widthCM, 1, 400),
			RenderDPI: clampF(dpi, 1, 4800),
			Brushing:  Downwards,
			Pasting:   FromLeft,
			Labels:    true,
		}
		if a3 {
			o.Paper = A3
		}
		info := ImageInfo{AspectRatio: clampF(aspect, 0.02, 40)}
		l, err := ComputeLayout(o, info)
		if err != nil {
			return
		}
		if l.Cols < 1 || l.Rows < 1 {
			t.Fatalf("non-positive grid %dx%d", l.Cols, l.Rows)
		}
		if len(l.Tiles) != l.Cols*l.Rows {
			t.Fatalf("tile count %d != %d*%d", len(l.Tiles), l.Cols, l.Rows)
		}
	})
}
