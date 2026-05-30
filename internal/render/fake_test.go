package render

import (
	"image"
	"image/color"

	"tile/internal/tile"
)

// fakeSource is an in-memory source for renderer tests (no file I/O).
type fakeSource struct {
	aspect float64
	w, h   int
}

func (f fakeSource) Info() tile.ImageInfo {
	return tile.ImageInfo{AspectRatio: f.aspect, PixelWidth: f.w, PixelHeight: f.h}
}

func (f fakeSource) RenderTile(_, _ float64, win tile.Rect) (image.Image, error) {
	// Small constant-size bitmap with a recognisable colour so the PDF is real.
	img := image.NewRGBA(image.Rect(0, 0, 120, 90))
	for y := 0; y < 90; y++ {
		for x := 0; x < 120; x++ {
			img.Set(x, y, color.RGBA{R: 200, G: 220, B: 240, A: 255})
		}
	}
	return img, nil
}
