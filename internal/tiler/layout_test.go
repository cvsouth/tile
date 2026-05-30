package tiler

import (
	"math"
	"testing"
)

func raster(aspect float64, pw, ph int) ImageInfo {
	return ImageInfo{AspectRatio: aspect, PixelWidth: pw, PixelHeight: ph}
}

func mustLayout(t *testing.T, o Options, info ImageInfo) Layout {
	t.Helper()
	l, err := ComputeLayout(o, info)
	if err != nil {
		t.Fatalf("ComputeLayout: %v", err)
	}
	return l
}

func TestDefaultsChooseLandscape(t *testing.T) {
	o := DefaultOptions()
	// 123 cm wide, square-ish image. Default paper is A3.
	l := mustLayout(t, o, raster(0.75, 4000, 3000))
	if l.Orientation != Landscape {
		t.Errorf("expected landscape, got %s", l.Orientation)
	}
	// posterW = 1230mm; landscape A3 step = 420-15 = 405; cols = ceil((1230-15)/405)=ceil(3.0)=3
	if l.Cols != 3 {
		t.Errorf("cols = %d, want 3", l.Cols)
	}
	if l.PaperW != 420 || l.PaperH != 297 {
		t.Errorf("paper = %gx%g, want 420x297", l.PaperW, l.PaperH)
	}
}

func TestLandscapeNeverMoreColumnsThanPortrait(t *testing.T) {
	// Sweep a wide range; landscape cols must always be <= portrait cols.
	for _, paper := range []Paper{A4, A3} {
		for wcm := 5.0; wcm <= 600; wcm += 3.7 {
			for _, ov := range []float64{5, 15, 30, 50} {
				o := Options{Paper: paper, OverlapMM: ov, WidthCM: wcm, RenderDPI: 300}
				if o.Validate() != nil {
					continue
				}
				pw, ph := paper.PortraitDims()
				lc := tileCount(wcm*10, ph, ov) // landscape paperW = long side
				pc := tileCount(wcm*10, pw, ov) // portrait paperW = short side
				if lc > pc {
					t.Fatalf("paper=%s w=%g ov=%g: landscape cols %d > portrait cols %d", paper, wcm, ov, lc, pc)
				}
			}
		}
	}
}

func TestPortraitWinsColumnTieFewerPages(t *testing.T) {
	// A4, overlap 10mm, width 189mm (18.9cm), height 1093mm => aspect ~5.783.
	o := Options{Paper: A4, OverlapMM: 10, WidthCM: 18.9, RenderDPI: 300, Brushing: Downwards, Pasting: FromLeft}
	aspect := 1093.0 / 189.0
	l := mustLayout(t, o, raster(aspect, 1890, 10930))
	if l.Orientation != Portrait {
		t.Fatalf("expected portrait (column tie, fewer pages), got %s with %dx%d", l.Orientation, l.Cols, l.Rows)
	}
	if l.Cols != 1 || l.Rows != 4 {
		t.Errorf("got %dx%d, want 1x4", l.Cols, l.Rows)
	}
}

func TestSmallImageStillProducesOneTile(t *testing.T) {
	// width below overlap must not yield zero tiles.
	o := DefaultOptions()
	o.WidthCM = 1 // 10mm, below the 15mm overlap
	l := mustLayout(t, o, raster(1.0, 100, 100))
	if l.Cols < 1 || l.Rows < 1 {
		t.Fatalf("got %dx%d, want at least 1x1", l.Cols, l.Rows)
	}
	if l.TotalPages() < 1 {
		t.Fatalf("total pages = %d", l.TotalPages())
	}
}

func TestValidateRejectsBadOptions(t *testing.T) {
	cases := []struct {
		name string
		o    Options
	}{
		{"zero width", Options{Paper: A4, OverlapMM: 15, WidthCM: 0, RenderDPI: 300}},
		{"negative width", Options{Paper: A4, OverlapMM: 15, WidthCM: -5, RenderDPI: 300}},
		{"zero overlap", Options{Paper: A4, OverlapMM: 0, WidthCM: 100, RenderDPI: 300}},
		{"overlap == short side", Options{Paper: A4, OverlapMM: 210, WidthCM: 100, RenderDPI: 300}},
		{"overlap > short side", Options{Paper: A4, OverlapMM: 250, WidthCM: 100, RenderDPI: 300}},
		{"zero dpi", Options{Paper: A4, OverlapMM: 15, WidthCM: 100, RenderDPI: 0}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := c.o.Validate(); err == nil {
				t.Errorf("expected error for %s", c.name)
			}
			if _, err := ComputeLayout(c.o, raster(1, 10, 10)); err == nil {
				t.Errorf("ComputeLayout accepted invalid options for %s", c.name)
			}
		})
	}
}

func TestA3OverlapBoundary(t *testing.T) {
	// 250mm overlap is invalid for A4 (short side 210) but valid for A3 (297).
	a4 := Options{Paper: A4, OverlapMM: 250, WidthCM: 100, RenderDPI: 300}
	if a4.Validate() == nil {
		t.Error("A4 overlap 250 should be invalid")
	}
	a3 := Options{Paper: A3, OverlapMM: 250, WidthCM: 100, RenderDPI: 300}
	if err := a3.Validate(); err != nil {
		t.Errorf("A3 overlap 250 should be valid: %v", err)
	}
}

func TestRejectsBadAspect(t *testing.T) {
	bad := []float64{0, -1, math.NaN(), math.Inf(1), math.Inf(-1)}
	for _, ar := range bad {
		if _, err := ComputeLayout(DefaultOptions(), ImageInfo{AspectRatio: ar}); err == nil {
			t.Errorf("expected error for aspect ratio %g", ar)
		}
	}
}

func TestEffectiveDPI(t *testing.T) {
	o := DefaultOptions() // width 123cm => 1230mm
	// raster: 4000px wide => 4000*25.4/1230 = 82.6 dpi
	l := mustLayout(t, o, raster(1.0, 4000, 4000))
	if math.Abs(l.EffectiveDPI-82.6) > 0.2 {
		t.Errorf("raster eff dpi = %g, want ~82.6", l.EffectiveDPI)
	}
	// vector: eff dpi == render dpi
	o.RenderDPI = 222
	lv, _ := ComputeLayout(o, ImageInfo{AspectRatio: 1, IsVector: true})
	if lv.EffectiveDPI != 222 {
		t.Errorf("vector eff dpi = %g, want 222", lv.EffectiveDPI)
	}
}

func TestTileWindowsCoverPosterAndOverlapExactly(t *testing.T) {
	o := DefaultOptions()
	l := mustLayout(t, o, raster(0.6, 5000, 3000))
	stepW := l.PaperW - l.Overlap
	stepH := l.PaperH - l.Overlap

	// Every adjacent horizontal pair overlaps by exactly the overlap.
	for _, tile := range l.Tiles {
		if tile.Col < l.Cols-1 {
			next := l.Tiles[tile.Row*l.Cols+tile.Col+1]
			gotOverlap := (tile.Window.X + tile.Window.W) - next.Window.X
			if math.Abs(gotOverlap-l.Overlap) > 1e-9 {
				t.Fatalf("horizontal overlap = %g, want %g", gotOverlap, l.Overlap)
			}
		}
	}
	// Last column/row must cover the far poster edge.
	lastColX := float64(l.Cols-1)*stepW + l.PaperW
	if lastColX < l.PosterW-1e-9 {
		t.Errorf("last column right edge %g does not cover poster width %g", lastColX, l.PosterW)
	}
	lastRowY := float64(l.Rows-1)*stepH + l.PaperH
	if lastRowY < l.PosterH-1e-9 {
		t.Errorf("last row bottom edge %g does not cover poster height %g", lastRowY, l.PosterH)
	}
	// One column short must NOT cover (the grid is minimal, not over-tiled).
	if l.Cols > 1 {
		shortX := float64(l.Cols-2)*stepW + l.PaperW
		if shortX >= l.PosterW {
			t.Errorf("grid is not minimal: %d-1 columns already cover %g", l.Cols, l.PosterW)
		}
	}
}

// bandRules describes the no-band corner expected for each direction combo.
func TestExactlyOneUnbandedTilePerCombo(t *testing.T) {
	combos := []struct {
		b       Brushing
		p       Pasting
		wantRow func(rows int) int
		wantCol func(cols int) int
		label   string
	}{
		{Downwards, FromLeft, func(r int) int { return 0 }, func(c int) int { return c - 1 }, "down+left=top-right"},
		{Downwards, FromRight, func(r int) int { return 0 }, func(c int) int { return 0 }, "down+right=top-left"},
		{Upwards, FromLeft, func(r int) int { return r - 1 }, func(c int) int { return c - 1 }, "up+left=bottom-right"},
		{Upwards, FromRight, func(r int) int { return r - 1 }, func(c int) int { return 0 }, "up+right=bottom-left"},
	}
	grids := []ImageInfo{
		raster(0.6, 5000, 3000), // multi x multi
		raster(3.0, 3000, 9000), // 1 col x many rows (tall)
		raster(0.05, 9000, 450), // many cols x 1 row (wide)
		raster(1.0, 50, 50),     // single tile
	}
	for _, combo := range combos {
		for _, info := range grids {
			o := DefaultOptions()
			o.Brushing, o.Pasting = combo.b, combo.p
			l := mustLayout(t, o, info)
			unbanded := 0
			for _, tile := range l.Tiles {
				if !tile.Bands.Any() {
					unbanded++
					if tile.Row != combo.wantRow(l.Rows) || tile.Col != combo.wantCol(l.Cols) {
						t.Errorf("%s grid %dx%d: unbanded tile at R%dC%d, want R%dC%d",
							combo.label, l.Rows, l.Cols, tile.Row, tile.Col, combo.wantRow(l.Rows), combo.wantCol(l.Cols))
					}
				}
			}
			if unbanded != 1 {
				t.Errorf("%s grid %dx%d: %d unbanded tiles, want exactly 1", combo.label, l.Rows, l.Cols, unbanded)
			}
		}
	}
}

func TestBandEdgesDefaults(t *testing.T) {
	// Downwards + FromLeft: interior tile must have top+right bands only.
	o := DefaultOptions()
	l := mustLayout(t, o, raster(0.6, 5000, 3000))
	if l.Cols < 3 || l.Rows < 3 {
		t.Fatalf("need an interior tile, got %dx%d", l.Cols, l.Rows)
	}
	interior := l.Tiles[1*l.Cols+1] // R1C1
	want := Bands{Top: true, Right: true}
	if interior.Bands != want {
		t.Errorf("interior bands = %+v, want %+v", interior.Bands, want)
	}
	// Top row, not last col: only right band (no top band).
	topMid := l.Tiles[0*l.Cols+1]
	if topMid.Bands != (Bands{Right: true}) {
		t.Errorf("top-mid bands = %+v, want only Right", topMid.Bands)
	}
	// Last col, not top row: only top band.
	lastColMid := l.Tiles[1*l.Cols+(l.Cols-1)]
	if lastColMid.Bands != (Bands{Top: true}) {
		t.Errorf("last-col-mid bands = %+v, want only Top", lastColMid.Bands)
	}
}
