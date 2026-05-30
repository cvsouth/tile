package render

import (
	"os"
	"path/filepath"
	"testing"

	"tile/internal/tile"
)

type rect struct{ x, y, w, h float64 }

func presentBandRects(b tile.Bands, pw, ph, ov float64) []rect {
	var rs []rect
	if b.Top {
		rs = append(rs, rect{0, 0, pw, ov})
	}
	if b.Bottom {
		rs = append(rs, rect{0, ph - ov, pw, ov})
	}
	if b.Left {
		rs = append(rs, rect{0, 0, ov, ph})
	}
	if b.Right {
		rs = append(rs, rect{pw - ov, 0, ov, ph})
	}
	return rs
}

func within(inner, outer rect) bool {
	const eps = 1e-6
	return inner.x >= outer.x-eps &&
		inner.y >= outer.y-eps &&
		inner.x+inner.w <= outer.x+outer.w+eps &&
		inner.y+inner.h <= outer.y+outer.h+eps
}

// The host strip for every banded tile must lie entirely within one of its
// present bands, across all four brushing/pasting combinations and grid shapes.
// This is the leak-proofness guarantee: a label confined to the strip is hidden.
func TestHostStripInsideABandForEveryTile(t *testing.T) {
	combos := []struct {
		b tile.Brushing
		p tile.Pasting
	}{
		{tile.Downwards, tile.FromLeft},
		{tile.Downwards, tile.FromRight},
		{tile.Upwards, tile.FromLeft},
		{tile.Upwards, tile.FromRight},
	}
	infos := []tile.ImageInfo{
		{AspectRatio: 0.6, PixelWidth: 5000, PixelHeight: 3000},
		{AspectRatio: 3.0, PixelWidth: 3000, PixelHeight: 9000},
		{AspectRatio: 0.05, PixelWidth: 9000, PixelHeight: 450},
		{AspectRatio: 1.0, PixelWidth: 50, PixelHeight: 50},
	}
	for _, combo := range combos {
		for _, info := range infos {
			o := tile.DefaultOptions()
			o.Brushing, o.Pasting = combo.b, combo.p
			l, err := tile.ComputeLayout(o, info)
			if err != nil {
				t.Fatal(err)
			}
			uncovered := 0
			for _, tile := range l.Tiles {
				s := chooseHostStrip(tile.Bands, combo.b, combo.p, l.PaperW, l.PaperH, l.Overlap)
				sr := rect{s.x, s.y, s.w, s.h}
				if !s.covered {
					uncovered++
					if tile.Bands.Any() {
						t.Errorf("tile R%dC%d marked uncovered but has bands %+v", tile.Row, tile.Col, tile.Bands)
					}
					continue
				}
				bands := presentBandRects(tile.Bands, l.PaperW, l.PaperH, l.Overlap)
				ok := false
				for _, br := range bands {
					if within(sr, br) {
						ok = true
						break
					}
				}
				if !ok {
					t.Errorf("brush=%v paste=%v grid %dx%d tile R%dC%d: host strip %+v not inside any band %+v",
						combo.b, combo.p, l.Rows, l.Cols, tile.Row, tile.Col, sr, bands)
				}
			}
			if uncovered != 1 {
				t.Errorf("brush=%v paste=%v grid %dx%d: %d uncovered host strips, want exactly 1",
					combo.b, combo.p, l.Rows, l.Cols, uncovered)
			}
		}
	}
}

func TestGenerateProducesPDF(t *testing.T) {
	dir := t.TempDir()
	// 100mm wide poster, square-ish, default A4 => a few tiles.
	src := fakeSource{aspect: 0.7, w: 1200, h: 840}
	o := tile.DefaultOptions()
	o.WidthCM = 60
	l, err := tile.ComputeLayout(o, src.Info())
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out.tiles.pdf")
	if err := Generate(l, src, o, out); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 1000 {
		t.Fatalf("pdf suspiciously small: %d bytes", len(data))
	}
	if string(data[:5]) != "%PDF-" {
		t.Fatalf("not a PDF: %q", data[:5])
	}
}
