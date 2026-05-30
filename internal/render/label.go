package render

import (
	"fmt"
	"math"

	"github.com/go-pdf/fpdf"

	"tiler/internal/tiler"
)

const minLabelPt = 3.0 // below this the label is illegibly small; skip it rather than leak

// Labels are drawn as a light casing under dark ink (a halo) so they read
// against any background — light, dark, or mid-tone — instead of vanishing.
var (
	darkInk  = [3]int{20, 20, 20}
	lightInk = [3]int{248, 248, 248}
	haloDirs = [8][2]float64{{-1, -1}, {0, -1}, {1, -1}, {-1, 0}, {1, 0}, {-1, 1}, {0, 1}, {1, 1}}
)

// hostStrip is the rectangle (page mm) a tile's label is drawn inside. It is
// always fully covered by a neighbour (hidden) unless covered is false, which
// only happens for the single tile that carries no band at all. vertical is true
// for left/right bands (a tall, overlap-wide strip), where the label is rotated.
type hostStrip struct {
	x, y, w, h        float64
	covered, vertical bool
}

// chooseHostStrip picks where a tile's label goes. It is confined to a single
// present band strip so the label can never leak past a covered edge. When the
// tile has no band (the one outer-corner tile), the label is placed in the
// poster's outer corner, derived from the brushing/pasting directions, and
// marked uncovered so the caller can draw it extra faintly.
func chooseHostStrip(b tiler.Bands, brush tiler.Brushing, paste tiler.Pasting, paperW, paperH, overlap float64) hostStrip {
	switch {
	case b.Top:
		return hostStrip{x: 0, y: 0, w: paperW, h: overlap, covered: true}
	case b.Bottom:
		return hostStrip{x: 0, y: paperH - overlap, w: paperW, h: overlap, covered: true}
	case b.Right:
		return hostStrip{x: paperW - overlap, y: 0, w: overlap, h: paperH, covered: true, vertical: true}
	case b.Left:
		return hostStrip{x: 0, y: 0, w: overlap, h: paperH, covered: true, vertical: true}
	default:
		y := 0.0
		if brush == tiler.Upwards {
			y = paperH - overlap
		}
		x := paperW - overlap // from-left => right edge is the home side
		if paste == tiler.FromRight {
			x = 0
		}
		return hostStrip{x: x, y: y, w: overlap, h: overlap, covered: false}
	}
}

// labelPlan is the fitted size of a label, including the halo allowance, sized so
// the whole block fits within its host strip. draw is false when no legible size
// fits (the label is then skipped rather than allowed to leak).
type labelPlan struct {
	draw    bool
	rotated bool
	fontPt  float64
	textW   float64
	capMM   float64
	halo    float64
	arrow   bool
	arrowW  float64
}

// planLabel fits the label (plus halo, plus an optional arrow on horizontal
// bands) to the strip, shrinking the font until the whole block fits within the
// strip's interior. The block is centred when drawn, so fitting guarantees
// containment.
func planLabel(pdf *fpdf.Fpdf, s hostStrip, label string, margin float64) labelPlan {
	lengthAvail, thickAvail := s.w-2*margin, s.h-2*margin
	if s.vertical {
		lengthAvail, thickAvail = s.h-2*margin, s.w-2*margin
	}
	if lengthAvail <= 0 || thickAvail <= 0 {
		return labelPlan{}
	}

	f := 12.0
	for i := 0; i < 8; i++ {
		pdf.SetFont("Helvetica", "", f)
		emMM := f / 72.0 * 25.4
		capMM := emMM * 0.7
		halo := math.Max(0.12, emMM*0.08)
		textW := pdf.GetStringWidth(label)

		arrow, arrowW := false, 0.0
		if !s.vertical {
			aw := capMM * 0.9
			if aw+margin+textW+2*halo <= lengthAvail {
				arrow, arrowW = true, aw
			}
		}
		gap := 0.0
		if arrow {
			gap = margin
		}
		blockLen := arrowW + gap + textW + 2*halo
		blockThick := capMM + 2*halo

		if blockLen <= lengthAvail && blockThick <= thickAvail {
			return labelPlan{draw: true, rotated: s.vertical, fontPt: f, textW: textW, capMM: capMM, halo: halo, arrow: arrow, arrowW: arrowW}
		}
		r := math.Min(lengthAvail/blockLen, thickAvail/blockThick)
		nf := f * r
		if nf >= f {
			nf = f * 0.9
		}
		f = nf
		if f < minLabelPt {
			break
		}
	}
	return labelPlan{}
}

// labelGeom is the resolved on-page placement of a label (centred in its strip),
// including the union ink bounding box (x0,y0,x1,y1 in page mm) used to prove the
// label cannot leak past a covered edge.
type labelGeom struct {
	rotated         bool
	cx, cy          float64
	textX, baseline float64
	arrow           bool
	arrowCx         float64
	bbox            [4]float64
}

func labelGeometry(s hostStrip, p labelPlan, margin float64) labelGeom {
	g := labelGeom{rotated: p.rotated, cx: s.x + s.w/2, cy: s.y + s.h/2}
	if p.rotated {
		hx := p.capMM/2 + p.halo
		hy := p.textW/2 + p.halo
		g.bbox = [4]float64{g.cx - hx, g.cy - hy, g.cx + hx, g.cy + hy}
		return g
	}
	gap := 0.0
	if p.arrow {
		gap = margin
	}
	blockLen := p.arrowW + gap + p.textW
	leftEdge := g.cx - blockLen/2
	if p.arrow {
		g.arrow = true
		g.arrowCx = leftEdge + p.arrowW/2
		g.textX = leftEdge + p.arrowW + gap
	} else {
		g.textX = leftEdge
	}
	g.baseline = g.cy + p.capMM/2
	g.bbox = [4]float64{leftEdge - p.halo, g.cy - p.capMM/2 - p.halo, leftEdge + blockLen + p.halo, g.cy + p.capMM/2 + p.halo}
	return g
}

// drawLabel draws the haloed alignment label, an up arrow (poster-top) on
// horizontal bands, and the seam guide lines, all inside the hidden overlap
// band(s) and legible against any background.
func drawLabel(pdf *fpdf.Fpdf, l tiler.Layout, o tiler.Options, tile tiler.Tile) {
	overlap := l.Overlap
	margin := math.Min(overlap*0.15, 1.0)

	// Seam guide lines (haloed dashed): where each covering neighbour's edge lands.
	guideCore := math.Max(0.15, overlap*0.025)
	pdf.SetAlpha(0.9, "Normal")
	pdf.SetDashPattern([]float64{1.2, 1.2}, 0)
	drawGuides(pdf, l, tile.Bands, guideCore)
	pdf.SetDashPattern([]float64{}, 0)

	strip := chooseHostStrip(tile.Bands, o.Brushing, o.Pasting, l.PaperW, l.PaperH, overlap)
	label := fmt.Sprintf("R%dC%d", tile.Row+1, tile.Col+1)
	plan := planLabel(pdf, strip, label, margin)
	if !plan.draw {
		pdf.SetAlpha(1, "Normal")
		return
	}

	alpha := 0.95
	if !strip.covered {
		alpha = 0.55 // the one unavoidably-visible corner: a touch softer
	}
	pdf.SetAlpha(alpha, "Normal")
	pdf.SetFont("Helvetica", "", plan.fontPt)

	g := labelGeometry(strip, plan, margin)
	if g.rotated {
		pdf.TransformBegin()
		pdf.TransformRotate(90, g.cx, g.cy)
		drawHaloText(pdf, g.cx-plan.textW/2, g.cy+plan.capMM/2, label, plan.halo)
		pdf.TransformEnd()
	} else {
		if g.arrow {
			drawHaloArrow(pdf, g.arrowCx, g.cy, plan.arrowW, plan.capMM, math.Max(0.1, plan.capMM*0.12), plan.halo)
		}
		drawHaloText(pdf, g.textX, g.baseline, label, plan.halo)
	}
	pdf.SetAlpha(1, "Normal")
}

func drawGuides(pdf *fpdf.Fpdf, l tiler.Layout, b tiler.Bands, core float64) {
	if b.Top {
		drawHaloLine(pdf, 0, l.Overlap, l.PaperW, l.Overlap, core)
	}
	if b.Bottom {
		drawHaloLine(pdf, 0, l.PaperH-l.Overlap, l.PaperW, l.PaperH-l.Overlap, core)
	}
	if b.Left {
		drawHaloLine(pdf, l.Overlap, 0, l.Overlap, l.PaperH, core)
	}
	if b.Right {
		drawHaloLine(pdf, l.PaperW-l.Overlap, 0, l.PaperW-l.Overlap, l.PaperH, core)
	}
}

func drawHaloLine(pdf *fpdf.Fpdf, x1, y1, x2, y2, core float64) {
	setDraw(pdf, lightInk)
	pdf.SetLineWidth(core * 3)
	pdf.Line(x1, y1, x2, y2)
	setDraw(pdf, darkInk)
	pdf.SetLineWidth(core)
	pdf.Line(x1, y1, x2, y2)
}

func drawHaloText(pdf *fpdf.Fpdf, x, baseline float64, s string, halo float64) {
	setText(pdf, lightInk)
	for _, d := range haloDirs {
		pdf.Text(x+d[0]*halo, baseline+d[1]*halo, s)
	}
	setText(pdf, darkInk)
	pdf.Text(x, baseline, s)
}

func drawHaloArrow(pdf *fpdf.Fpdf, cx, cy, w, h, core, halo float64) {
	setDraw(pdf, lightInk)
	pdf.SetLineWidth(core + 2*halo)
	arrowLines(pdf, cx, cy, w, h)
	setDraw(pdf, darkInk)
	pdf.SetLineWidth(core)
	arrowLines(pdf, cx, cy, w, h)
}

func arrowLines(pdf *fpdf.Fpdf, cx, cy, w, h float64) {
	top, bot := cy-h/2, cy+h/2
	pdf.Line(cx, bot, cx, top)
	pdf.Line(cx, top, cx-w/2, top+h*0.35)
	pdf.Line(cx, top, cx+w/2, top+h*0.35)
}

func setDraw(pdf *fpdf.Fpdf, c [3]int) { pdf.SetDrawColor(c[0], c[1], c[2]) }
func setText(pdf *fpdf.Fpdf, c [3]int) { pdf.SetTextColor(c[0], c[1], c[2]) }
