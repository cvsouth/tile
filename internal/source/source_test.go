package source

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"tiler/internal/tiler"
)

// makePNG writes a width x height PNG: left half red, right half blue.
func makePNG(t *testing.T, dir string, w, h int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x < w/2 {
				img.Set(x, y, color.RGBA{R: 255, A: 255})
			} else {
				img.Set(x, y, color.RGBA{B: 255, A: 255})
			}
		}
	}
	path := filepath.Join(dir, "test.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	return path
}

func isRed(c color.Color) bool { r, g, b, _ := c.RGBA(); return r > 0x8000 && g < 0x4000 && b < 0x4000 }
func isBlue(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	return b > 0x8000 && r < 0x4000 && g < 0x4000
}
func isWhite(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	return r > 0xc000 && g > 0xc000 && b > 0xc000
}

func TestRasterInfoAndRender(t *testing.T) {
	dir := t.TempDir()
	path := makePNG(t, dir, 100, 80)
	src, err := Load(path, 300)
	if err != nil {
		t.Fatal(err)
	}
	info := src.Info()
	if info.IsVector || info.PixelWidth != 100 || info.PixelHeight != 80 {
		t.Fatalf("info = %+v", info)
	}
	if info.AspectRatio != 0.8 {
		t.Fatalf("aspect = %g, want 0.8", info.AspectRatio)
	}

	// posterW = 100mm => 1 px/mm. Full window 0..100 x 0..80.
	full, err := src.RenderTile(100, 80, tiler.Rect{X: 0, Y: 0, W: 100, H: 80})
	if err != nil {
		t.Fatal(err)
	}
	if full.Bounds().Dx() != 100 || full.Bounds().Dy() != 80 {
		t.Fatalf("full tile size = %v, want 100x80", full.Bounds())
	}
	if !isRed(full.At(10, 40)) {
		t.Errorf("left half should be red, got %v", full.At(10, 40))
	}
	if !isBlue(full.At(90, 40)) {
		t.Errorf("right half should be blue, got %v", full.At(90, 40))
	}

	// Window running off the right edge: x 50..150. Left 50px = source blue,
	// right 50px = beyond image = blank white.
	off, err := src.RenderTile(100, 80, tiler.Rect{X: 50, Y: 0, W: 100, H: 80})
	if err != nil {
		t.Fatal(err)
	}
	if !isBlue(off.At(10, 40)) {
		t.Errorf("on-image part should be blue, got %v", off.At(10, 40))
	}
	if !isWhite(off.At(80, 40)) {
		t.Errorf("off-image part should be blank white, got %v", off.At(80, 40))
	}
}

const halfRedSVG = `<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100" viewBox="0 0 100 100"><rect x="0" y="0" width="50" height="100" fill="#ff0000"/></svg>`

func makeSVG(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "test.svg")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSVGInfoAndRender(t *testing.T) {
	dir := t.TempDir()
	path := makeSVG(t, dir, halfRedSVG)
	src, err := Load(path, 25.4) // 25.4 dpi => 1 px/mm
	if err != nil {
		t.Fatal(err)
	}
	info := src.Info()
	if !info.IsVector || info.AspectRatio != 1.0 {
		t.Fatalf("info = %+v, want vector aspect 1.0", info)
	}

	// posterW = 100mm, 1px/mm => 100x100 tile. Left half red, right half blank.
	full, err := src.RenderTile(100, 100, tiler.Rect{X: 0, Y: 0, W: 100, H: 100})
	if err != nil {
		t.Fatal(err)
	}
	if full.Bounds().Dx() != 100 || full.Bounds().Dy() != 100 {
		t.Fatalf("svg tile size = %v, want 100x100", full.Bounds())
	}
	if !isRed(full.At(20, 50)) {
		t.Errorf("svg left half should be red, got %v", full.At(20, 50))
	}
	if !isWhite(full.At(80, 50)) {
		t.Errorf("svg right half should be blank white, got %v", full.At(80, 50))
	}

	// Right-half window (x 50..100) maps to the blank right portion.
	right, err := src.RenderTile(100, 100, tiler.Rect{X: 50, Y: 0, W: 50, H: 100})
	if err != nil {
		t.Fatal(err)
	}
	if !isWhite(right.At(25, 50)) {
		t.Errorf("svg right-half window should be blank, got %v", right.At(25, 50))
	}
}

func TestLoadRejectsUnsupported(t *testing.T) {
	if _, err := Load("foo.gif", 300); err == nil {
		t.Error("expected error for unsupported extension")
	}
	if IsSupported("foo.bmp") {
		t.Error("bmp should not be supported")
	}
	for _, ext := range []string{"a.jpg", "a.JPG", "a.png", "a.svg", "a.jpeg"} {
		if !IsSupported(ext) {
			t.Errorf("%s should be supported", ext)
		}
	}
}
