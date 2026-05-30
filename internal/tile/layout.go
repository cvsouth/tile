package tile

import (
	"fmt"
	"math"
)

// Rect is an axis-aligned rectangle in poster millimetre space (origin top-left).
type Rect struct {
	X, Y, W, H float64
}

// ImageInfo is the pure metadata the core needs about the source image. The
// shell extracts this once after decoding so the core stays I/O-free.
type ImageInfo struct {
	AspectRatio float64 // height / width of the source content
	IsVector    bool    // true for SVG (rasterised on demand at RenderDPI)
	PixelWidth  int     // intrinsic pixel width (raster sources only)
	PixelHeight int     // intrinsic pixel height (raster sources only)
}

// Bands records which edges of a tile carry an overlap band — the strip that is
// hidden underneath the neighbouring piece once assembled.
type Bands struct {
	Top, Bottom, Left, Right bool
}

// Any reports whether the tile has at least one hidden band.
func (b Bands) Any() bool { return b.Top || b.Bottom || b.Left || b.Right }

// Tile is one printed page: a full-page window onto the poster plus the band
// edges that will be covered by neighbours.
type Tile struct {
	Row, Col int
	Window   Rect // full page window in poster mm
	Bands    Bands
}

// Layout is the complete tiling plan for a set of options and a source image.
type Layout struct {
	Orientation  Orientation
	PaperW       float64 // page width in mm, in the chosen orientation
	PaperH       float64 // page height in mm, in the chosen orientation
	PosterW      float64 // full printed width in mm
	PosterH      float64 // full printed height in mm
	Overlap      float64 // glue overlap in mm
	Cols, Rows   int
	EffectiveDPI float64 // effective print resolution of the source
	IsVector     bool
	Tiles        []Tile
}

// TotalPages is the number of pages the layout will print.
func (l Layout) TotalPages() int { return l.Cols * l.Rows }

// tileCount returns how many full-size pages of dimension paper are needed to
// cover total, given that adjacent pages overlap by overlap. Always >= 1 for a
// positive image so a small image still yields one tile.
func tileCount(total, paper, overlap float64) int {
	step := paper - overlap
	if step <= 0 {
		return 1 // guarded by Options.Validate; defensive only
	}
	n := int(math.Ceil((total - overlap) / step))
	if n < 1 {
		n = 1
	}
	return n
}

// ComputeLayout builds the tiling plan. It chooses the orientation that
// minimises the number of vertical columns, breaking ties by fewest total
// pages (so a column tie can favour portrait's taller page).
func ComputeLayout(o Options, info ImageInfo) (Layout, error) {
	if err := o.Validate(); err != nil {
		return Layout{}, err
	}
	if info.AspectRatio <= 0 || math.IsNaN(info.AspectRatio) || math.IsInf(info.AspectRatio, 0) {
		return Layout{}, fmt.Errorf("image aspect ratio must be a positive finite number (got %g)", info.AspectRatio)
	}

	posterW := o.WidthCM * 10
	posterH := posterW * info.AspectRatio

	pw, ph := o.Paper.PortraitDims()
	// Landscape is (long, short); portrait is (short, long).
	landW, landH := ph, pw
	portW, portH := pw, ph

	lc := tileCount(posterW, landW, o.OverlapMM)
	lr := tileCount(posterH, landH, o.OverlapMM)
	pc := tileCount(posterW, portW, o.OverlapMM)
	pr := tileCount(posterH, portH, o.OverlapMM)

	// Minimise columns; on a column tie, minimise total pages. Landscape wins
	// every remaining case (it never yields more columns than portrait).
	usePortrait := pc < lc || (pc == lc && pc*pr < lc*lr)

	l := Layout{
		PosterW:  posterW,
		PosterH:  posterH,
		Overlap:  o.OverlapMM,
		IsVector: info.IsVector,
	}
	if usePortrait {
		l.Orientation = Portrait
		l.PaperW, l.PaperH = portW, portH
		l.Cols, l.Rows = pc, pr
	} else {
		l.Orientation = Landscape
		l.PaperW, l.PaperH = landW, landH
		l.Cols, l.Rows = lc, lr
	}

	if info.IsVector {
		l.EffectiveDPI = o.RenderDPI
	} else {
		l.EffectiveDPI = float64(info.PixelWidth) * 25.4 / posterW
	}

	l.Tiles = buildTiles(l, o)
	return l, nil
}

// buildTiles enumerates the pages row-major and assigns each its window and band
// edges from the brushing/pasting directions. The band always sits on the edge
// the neighbouring piece will cover, so it is hidden once assembled.
func buildTiles(l Layout, o Options) []Tile {
	stepW := l.PaperW - l.Overlap
	stepH := l.PaperH - l.Overlap
	tiles := make([]Tile, 0, l.Cols*l.Rows)
	for r := 0; r < l.Rows; r++ {
		for c := 0; c < l.Cols; c++ {
			var b Bands
			// Vertical seams set by brushing direction.
			if o.Brushing == Downwards {
				b.Top = r > 0 // upper piece on top; covered band on lower piece's top edge
			} else {
				b.Bottom = r < l.Rows-1
			}
			// Horizontal seams set by pasting order.
			if o.Pasting == FromLeft {
				b.Right = c < l.Cols-1 // next strip laid on top to the right
			} else {
				b.Left = c > 0
			}
			tiles = append(tiles, Tile{
				Row: r,
				Col: c,
				Window: Rect{
					X: float64(c) * stepW,
					Y: float64(r) * stepH,
					W: l.PaperW,
					H: l.PaperH,
				},
				Bands: b,
			})
		}
	}
	return tiles
}
