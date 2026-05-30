package render

import (
	"math"
	"testing"

	"github.com/go-pdf/fpdf"

	"tiler/internal/tiler"
)

// The faint label's actual ink must never leak past a covered band edge. This
// measures the real drawn bounding box (via fpdf string widths) across small
// overlaps, long labels and all four band types — the case the rectangle-only
// test could not catch.
func TestLabelInkNeverLeaksBand(t *testing.T) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	type scenario struct {
		name  string
		bands tiler.Bands
	}
	scenarios := []scenario{
		{"top", tiler.Bands{Top: true}},
		{"bottom", tiler.Bands{Bottom: true}},
		{"left", tiler.Bands{Left: true}},
		{"right", tiler.Bands{Right: true}},
		{"top+right", tiler.Bands{Top: true, Right: true}},
	}
	overlaps := []float64{0.5, 1, 2, 3, 5, 8, 15, 30}
	labels := []string{"R1C1", "R9C9", "R12C34", "R100C100", "R1000C100"}
	const paperW, paperH = 297.0, 210.0
	const eps = 1e-6

	for _, sc := range scenarios {
		for _, ov := range overlaps {
			margin := math.Min(ov*0.15, 1.0)
			strip := chooseHostStrip(sc.bands, tiler.Downwards, tiler.FromLeft, paperW, paperH, ov)
			// band rectangle for this single-band scenario equals the strip.
			bx0, by0 := strip.x, strip.y
			bx1, by1 := strip.x+strip.w, strip.y+strip.h
			for _, lbl := range labels {
				plan := planLabel(pdf, strip, lbl, margin)
				if !plan.draw {
					continue // skipped rather than leaked — acceptable
				}
				g := labelGeometry(strip, plan)
				x0, y0, x1, y1 := g.bbox[0], g.bbox[1], g.bbox[2], g.bbox[3]
				if x0 < bx0-eps || y0 < by0-eps || x1 > bx1+eps || y1 > by1+eps {
					t.Errorf("LEAK %s ov=%g label=%q: ink bbox [%.3f,%.3f,%.3f,%.3f] outside band [%.3f,%.3f,%.3f,%.3f]",
						sc.name, ov, lbl, x0, y0, x1, y1, bx0, by0, bx1, by1)
				}
			}
		}
	}
}

// At a typical overlap the label must actually be drawn (not skipped), so the
// feature stays useful — the fix must not silently drop every label.
func TestLabelDrawnAtTypicalOverlap(t *testing.T) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	for _, bands := range []tiler.Bands{{Top: true}, {Right: true}, {Bottom: true}, {Left: true}} {
		strip := chooseHostStrip(bands, tiler.Downwards, tiler.FromLeft, 297, 210, 15)
		plan := planLabel(pdf, strip, "R3C4", math.Min(15*0.15, 1.0))
		if !plan.draw {
			t.Errorf("label for bands %+v unexpectedly skipped at 15mm overlap", bands)
		}
	}
}
