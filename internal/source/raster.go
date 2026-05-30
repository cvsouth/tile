package source

import (
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg" // register JPEG decoder
	_ "image/png"  // register PNG decoder
	"math"
	"os"

	"tile/internal/tile"
)

type rasterSource struct {
	img    image.Image
	width  int
	height int
}

func loadRaster(path string) (Source, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decoding %s: %w", path, err)
	}
	b := img.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		return nil, fmt.Errorf("%s has zero size", path)
	}
	return &rasterSource{img: img, width: b.Dx(), height: b.Dy()}, nil
}

func (r *rasterSource) Info() tile.ImageInfo {
	return tile.ImageInfo{
		AspectRatio: float64(r.height) / float64(r.width),
		IsVector:    false,
		PixelWidth:  r.width,
		PixelHeight: r.height,
	}
}

func (r *rasterSource) RenderTile(posterW, _ float64, win tile.Rect) (image.Image, error) {
	// One source pixel per (posterW / imgW) mm; preserved aspect makes this the
	// same density on both axes, so no upscaling is introduced by tiling.
	ppm := float64(r.width) / posterW
	ow := maxInt(1, int(math.Round(win.W*ppm)))
	oh := maxInt(1, int(math.Round(win.H*ppm)))

	dst := newWhite(ow, oh)

	// Top-left source pixel this window starts at (may be negative or beyond the
	// image; draw.Draw clips to the source bounds, leaving the rest white).
	srcX := int(math.Round(win.X * ppm))
	srcY := int(math.Round(win.Y * ppm))
	b := r.img.Bounds()
	sp := image.Pt(b.Min.X+srcX, b.Min.Y+srcY)
	draw.Draw(dst, dst.Bounds(), r.img, sp, draw.Over)
	return dst, nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func newWhite(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := range img.Pix {
		img.Pix[i] = 0xff
	}
	return img
}
