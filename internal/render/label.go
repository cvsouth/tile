package render

import (
	"fmt"
	"math"

	"github.com/go-pdf/fpdf"

	"tile/internal/tile"
)

const minLabelPt = 3.0 // below this the label is illegibly small; skip it rather than leak

// Opacities for the in-band marks. They sit in the hidden overlap band, so they
// are kept faint; the black+white construction keeps them readable on any
// background even at low opacity.
const (
	guideAlpha = 0.12 // alternating white/black dashed seam guides — kept very subtle
	labelAlpha = 0.4  // label on a covered (hidden) band
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
func chooseHostStrip(b tile.Bands, brush tile.Brushing, paste tile.Pasting, paperW, paperH, overlap float64) hostStrip {
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
		if brush == tile.Upwards {
			y = paperH - overlap
		}
		x := paperW - overlap // from-left => right edge is the home side
		if paste == tile.FromRight {
			x = 0
		}
		return hostStrip{x: x, y: y, w: overlap, h: overlap, covered: false}
	}
}

// labelPlan is the fitted size of a label. The label is printed twice — a white
// copy and a black copy, side by side — so whichever colour contrasts with the
// background stays readable. The two copies plus the gap between them must fit
// the strip; draw is false when no legible size fits (then it is skipped rather
// than allowed to leak past a covered edge).
type labelPlan struct {
	draw    bool
	rotated bool
	fontPt  float64
	textW   float64
	capMM   float64
	gap     float64
}

// planLabel shrinks the font until both copies (along the strip's long axis) and
// the cap height (across it) fit the strip interior. The block is centred when
// drawn, so fitting guarantees containment.
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
		textW := pdf.GetStringWidth(label)
		gap := capMM * 0.8

		blockLen := 2*textW + gap
		if blockLen <= lengthAvail && capMM <= thickAvail {
			return labelPlan{draw: true, rotated: s.vertical, fontPt: f, textW: textW, capMM: capMM, gap: gap}
		}
		r := math.Min(lengthAvail/blockLen, thickAvail/capMM)
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

// labelGeom is the resolved on-page placement (centred in the strip): the start
// x of each copy, the shared baseline, and the ink bounding box (page mm) used
// to prove the label stays within its covered band.
type labelGeom struct {
	rotated                  bool
	cx, cy                   float64
	whiteX, blackX, baseline float64
	bbox                     [4]float64
}

func labelGeometry(s hostStrip, p labelPlan) labelGeom {
	g := labelGeom{rotated: p.rotated, cx: s.x + s.w/2, cy: s.y + s.h/2}
	blockLen := 2*p.textW + p.gap
	// Local layout (length axis horizontal): both copies on one baseline, centred.
	g.whiteX = g.cx - blockLen/2
	g.blackX = g.whiteX + p.textW + p.gap
	g.baseline = g.cy + p.capMM/2
	if p.rotated {
		// Rotating 90° swaps the axes: length runs down the page, cap across it.
		g.bbox = [4]float64{g.cx - p.capMM/2, g.cy - blockLen/2, g.cx + p.capMM/2, g.cy + blockLen/2}
	} else {
		g.bbox = [4]float64{g.cx - blockLen/2, g.cy - p.capMM/2, g.cx + blockLen/2, g.cy + p.capMM/2}
	}
	return g
}

// drawLabel draws the alignment label (a white copy beside a black copy, so it
// reads on any background) and the seam guide lines, all inside the hidden
// overlap band(s).
func drawLabel(pdf *fpdf.Fpdf, l tile.Layout, o tile.Options, tile tile.Tile) {
	overlap := l.Overlap
	margin := math.Min(overlap*0.15, 1.0)

	// Seam guide lines: where each covering neighbour's edge lands. Drawn as
	// dashes that alternate white and black, so at least one colour always
	// contrasts with the background.
	drawGuides(pdf, l, tile.Bands, math.Max(0.15, overlap*0.02))

	strip := chooseHostStrip(tile.Bands, o.Brushing, o.Pasting, l.PaperW, l.PaperH, overlap)
	if !strip.covered {
		// This tile has no band — nowhere it will be overlapped by a neighbour —
		// so a label here could never be hidden. Don't print one at all.
		return
	}
	label := fmt.Sprintf("R%dC%d", tile.Row+1, tile.Col+1)
	plan := planLabel(pdf, strip, label, margin)
	if !plan.draw {
		return
	}

	pdf.SetAlpha(labelAlpha, "Normal")
	pdf.SetFont("Helvetica", "", plan.fontPt)

	g := labelGeometry(strip, plan)
	if g.rotated {
		pdf.TransformBegin()
		pdf.TransformRotate(90, g.cx, g.cy)
		drawTwoTone(pdf, g.whiteX, g.blackX, g.baseline, label)
		pdf.TransformEnd()
	} else {
		drawTwoTone(pdf, g.whiteX, g.blackX, g.baseline, label)
	}
	pdf.SetAlpha(1, "Normal")
}

// drawTwoTone prints the label in white, then again in black just beside it.
func drawTwoTone(pdf *fpdf.Fpdf, whiteX, blackX, baseline float64, label string) {
	pdf.SetTextColor(255, 255, 255)
	pdf.Text(whiteX, baseline, label)
	pdf.SetTextColor(0, 0, 0)
	pdf.Text(blackX, baseline, label)
}

func drawGuides(pdf *fpdf.Fpdf, l tile.Layout, b tile.Bands, lineW float64) {
	if b.Top {
		drawAltLine(pdf, 0, l.Overlap, l.PaperW, l.Overlap, lineW)
	}
	if b.Bottom {
		drawAltLine(pdf, 0, l.PaperH-l.Overlap, l.PaperW, l.PaperH-l.Overlap, lineW)
	}
	if b.Left {
		drawAltLine(pdf, l.Overlap, 0, l.Overlap, l.PaperH, lineW)
	}
	if b.Right {
		drawAltLine(pdf, l.PaperW-l.Overlap, 0, l.PaperW-l.Overlap, l.PaperH, lineW)
	}
}

// drawAltLine draws a thin guide line whose dashes alternate white and black,
// each at low opacity. The two colours fall on different segments, so whichever
// contrasts with the background shows through — the line is visible on light,
// dark and mid-tone backgrounds alike, with no halo and a single line width.
func drawAltLine(pdf *fpdf.Fpdf, x1, y1, x2, y2, lineW float64) {
	const dash = 1.5
	pdf.SetAlpha(guideAlpha, "Normal")
	pdf.SetLineWidth(lineW)
	pdf.SetDrawColor(255, 255, 255)
	pdf.SetDashPattern([]float64{dash, dash}, 0)
	pdf.Line(x1, y1, x2, y2)
	pdf.SetDrawColor(0, 0, 0)
	pdf.SetDashPattern([]float64{dash, dash}, dash) // phase into the gap => fills white's off-segments
	pdf.Line(x1, y1, x2, y2)
	pdf.SetDashPattern([]float64{}, 0)
	pdf.SetAlpha(1, "Normal")
}
