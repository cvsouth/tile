// Package render turns a Layout plus an image Source into a multi-page,
// full-bleed PDF ready for borderless printing.
package render

import (
	"bytes"
	"fmt"
	"image/png"

	"github.com/go-pdf/fpdf"

	"tile/internal/source"
	"tile/internal/tile"
)

// Generate writes the tiled PDF for layout l to outPath, rendering each tile on
// demand from src. Every page is full paper size (borderless); the image is
// placed edge to edge and, when enabled, a faint alignment label and guide are
// drawn inside the hidden overlap band.
func Generate(l tile.Layout, src source.Source, o tile.Options, outPath string) error {
	pdf := fpdf.NewCustom(&fpdf.InitType{
		UnitStr: "mm",
		Size:    fpdf.SizeType{Wd: l.PaperW, Ht: l.PaperH},
	})
	pdf.SetMargins(0, 0, 0)
	pdf.SetAutoPageBreak(false, 0)

	imgOpt := fpdf.ImageOptions{ImageType: "PNG", ReadDpi: false}

	for i, tile := range l.Tiles {
		bmp, err := src.RenderTile(l.PosterW, l.PosterH, tile.Window)
		if err != nil {
			return fmt.Errorf("rendering tile R%dC%d: %w", tile.Row+1, tile.Col+1, err)
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, bmp); err != nil {
			return fmt.Errorf("encoding tile R%dC%d: %w", tile.Row+1, tile.Col+1, err)
		}
		name := fmt.Sprintf("tile%d", i)
		pdf.RegisterImageOptionsReader(name, imgOpt, bytes.NewReader(buf.Bytes()))

		pdf.AddPage()
		pdf.ImageOptions(name, 0, 0, l.PaperW, l.PaperH, false, imgOpt, 0, "")
		if o.Labels {
			drawLabel(pdf, l, o, tile)
		}
		if pdf.Err() {
			return fmt.Errorf("pdf error on tile R%dC%d: %w", tile.Row+1, tile.Col+1, pdf.Error())
		}
	}

	if err := pdf.OutputFileAndClose(outPath); err != nil {
		return fmt.Errorf("writing %s: %w", outPath, err)
	}
	return nil
}
