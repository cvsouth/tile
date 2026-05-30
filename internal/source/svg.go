package source

import (
	"fmt"
	"image"
	"math"
	"os"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"

	"tile/internal/tile"
)

type svgSource struct {
	icon      *oksvg.SvgIcon
	vbW, vbH  float64
	vbX, vbY  float64
	renderDPI float64
}

func loadSVG(path string, renderDPI float64) (Source, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	icon, err := oksvg.ReadIconStream(f, oksvg.WarnErrorMode)
	if err != nil {
		return nil, fmt.Errorf("parsing SVG %s: %w", path, err)
	}
	if icon.ViewBox.W <= 0 || icon.ViewBox.H <= 0 {
		return nil, fmt.Errorf("SVG %s has no usable viewBox/size", path)
	}
	return &svgSource{
		icon:      icon,
		vbW:       icon.ViewBox.W,
		vbH:       icon.ViewBox.H,
		vbX:       icon.ViewBox.X,
		vbY:       icon.ViewBox.Y,
		renderDPI: renderDPI,
	}, nil
}

func (s *svgSource) Info() tile.ImageInfo {
	return tile.ImageInfo{
		AspectRatio: s.vbH / s.vbW,
		IsVector:    true,
	}
}

// SetRenderDPI changes the rasterisation density used for subsequent tiles,
// letting the interactive UI honour edits to the DPI field without reloading.
func (s *svgSource) SetRenderDPI(dpi float64) { s.renderDPI = dpi }

// RenderTile is not safe for concurrent use: it sets the transform on the shared
// parsed icon. The PDF renderer calls it serially.
func (s *svgSource) RenderTile(posterW, _ float64, win tile.Rect) (image.Image, error) {
	// Rasterise at the requested DPI measured against the final poster scale, so
	// the effective print DPI genuinely equals the render DPI.
	ppm := s.renderDPI / 25.4
	ow := maxInt(1, int(math.Round(win.W*ppm)))
	oh := maxInt(1, int(math.Round(win.H*ppm)))

	// Map poster millimetres to viewBox units (preserved aspect => one factor).
	vbPerMM := s.vbW / posterW
	// Absolute viewBox coordinate of the window's top-left.
	vAbs0X := s.vbX + win.X*vbPerMM
	vAbs0Y := s.vbY + win.Y*vbPerMM
	// Window size in viewBox units, and the scale onto the output bitmap.
	vw := win.W * vbPerMM
	vh := win.H * vbPerMM
	sx := float64(ow) / vw
	sy := float64(oh) / vh

	dst := newWhite(ow, oh)
	// Build the transform directly (rather than SetTarget) so a non-zero viewBox
	// origin is handled correctly: a viewBox point v maps to (v - vAbs0)*scale.
	s.icon.Transform = rasterx.Identity.Translate(-vAbs0X*sx, -vAbs0Y*sy).Scale(sx, sy)

	scanner := rasterx.NewScannerGV(ow, oh, dst, dst.Bounds())
	raster := rasterx.NewDasher(ow, oh, scanner)
	s.icon.Draw(raster, 1.0)
	return dst, nil
}
