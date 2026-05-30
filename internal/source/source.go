// Package source loads image files (JPG, PNG, SVG) and renders arbitrary
// poster-space windows of them on demand, so the PDF renderer can produce one
// tile at a time without ever holding the whole poster in memory.
package source

import (
	"fmt"
	"image"
	"path/filepath"
	"strings"

	"tile/internal/tile"
)

// Source is a loaded image that can render any poster-space window.
type Source interface {
	// Info reports the pure metadata the core needs (aspect ratio, vector flag,
	// intrinsic pixel size for raster sources).
	Info() tile.ImageInfo
	// RenderTile renders the poster-space window win (millimetres) into a fresh
	// bitmap. posterW/posterH are the full printed poster size in millimetres.
	// Any part of the window beyond the image is left blank (white).
	RenderTile(posterW, posterH float64, win tile.Rect) (image.Image, error)
}

// RenderDPISetter is implemented by sources whose rasterisation DPI can change
// after loading (vector sources). Raster sources do not implement it.
type RenderDPISetter interface {
	SetRenderDPI(dpi float64)
}

// SupportedExts lists the accepted input extensions (lower case, with dot).
var SupportedExts = []string{".jpg", ".jpeg", ".png", ".svg"}

// IsSupported reports whether path has a recognised image extension.
func IsSupported(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, e := range SupportedExts {
		if ext == e {
			return true
		}
	}
	return false
}

// Load reads the file at path and returns a Source. renderDPI is the
// rasterisation density used for vector (SVG) sources; it is ignored for raster
// sources.
func Load(path string, renderDPI float64) (Source, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".svg":
		return loadSVG(path, renderDPI)
	case ".jpg", ".jpeg", ".png":
		return loadRaster(path)
	default:
		return nil, fmt.Errorf("unsupported file type %q (supported: jpg, jpeg, png, svg)", ext)
	}
}
