package tiler

import (
	"path/filepath"
	"strings"
)

// imageExts are the recognised input extensions stripped when deriving the
// default output name. Only these are removed, so "my.image.png" keeps its dots
// and "poster.tiles.pdf" does not become "poster.tiles.tiles.pdf".
var imageExts = []string{".jpg", ".jpeg", ".png", ".svg"}

// DefaultOutputName derives the default PDF path from the input image path:
// the recognised image extension is stripped and ".tiles.pdf" appended, beside
// the original file.
func DefaultOutputName(input string) string {
	dir := filepath.Dir(input)
	base := filepath.Base(input)
	lower := strings.ToLower(base)
	for _, ext := range imageExts {
		if strings.HasSuffix(lower, ext) && len(base) > len(ext) {
			base = base[:len(base)-len(ext)]
			break
		}
	}
	return filepath.Join(dir, base+".tiles.pdf")
}
