// Command tiler turns an image into a multi-page, tile-and-glue PDF poster.
//
// Usage:
//
//	tiler [options] <image>          # interactive TUI (flags seed the defaults)
//	tiler --non-interactive [options] <image>
package main

import (
	"flag"
	"fmt"
	"os"

	"tiler/internal/render"
	"tiler/internal/source"
	"tiler/internal/tiler"
	"tiler/internal/tui"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "tiler: "+err.Error())
		os.Exit(1)
	}
}

func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

func run(args []string) error {
	def := tiler.DefaultOptions()

	fs := flag.NewFlagSet("tiler", flag.ContinueOnError)
	fs.Usage = func() {
		const usage = `tiler — image tiler for multi-page tile-and-glue prints

Usage:
  tiler [options] <image.(jpg|jpeg|png|svg)>

With no --non-interactive flag the options below just seed the TUI defaults.

Options:
`
		_, _ = fmt.Fprint(fs.Output(), usage)
		fs.PrintDefaults()
	}

	paper := fs.String("paper", def.Paper.String(), "paper size: A4 or A3")
	overlap := fs.Float64("overlap", def.OverlapMM, "glue overlap in millimetres")
	width := fs.Float64("width", def.WidthCM, "printed width in centimetres")
	brushing := fs.String("brushing", def.Brushing.String(), "brushing direction: up or down")
	pasting := fs.String("pasting", "from-left", "pasting order: left or right")
	dpi := fs.Float64("dpi", def.RenderDPI, "render DPI for vector (SVG) sources")
	labels := fs.String("labels", onOff(def.Labels), "print faint alignment labels in the overlap band: on or off")
	output := fs.String("output", "", "output PDF path (default: <image>.tiles.pdf)")
	nonInteractive := fs.Bool("non-interactive", false, "generate immediately without the TUI")

	if err := fs.Parse(args); err != nil {
		return err
	}

	imagePath := fs.Arg(0)
	if imagePath == "" {
		fs.Usage()
		return fmt.Errorf("no image given")
	}
	if !source.IsSupported(imagePath) {
		return fmt.Errorf("unsupported file type for %q (supported: jpg, jpeg, png, svg)", imagePath)
	}

	o := def
	var err error
	if o.Paper, err = tiler.ParsePaper(*paper); err != nil {
		return err
	}
	if o.Brushing, err = tiler.ParseBrushing(*brushing); err != nil {
		return err
	}
	if o.Pasting, err = tiler.ParsePasting(*pasting); err != nil {
		return err
	}
	if o.Labels, err = tiler.ParseToggle(*labels); err != nil {
		return err
	}
	o.OverlapMM = *overlap
	o.WidthCM = *width
	o.RenderDPI = *dpi
	o.Output = *output

	src, err := source.Load(imagePath, o.RenderDPI)
	if err != nil {
		return err
	}

	// --dpi only applies to vector (SVG) sources.
	dpiSet := false
	fs.Visit(func(fl *flag.Flag) {
		if fl.Name == "dpi" {
			dpiSet = true
		}
	})
	if dpiSet && !src.Info().IsVector {
		return fmt.Errorf("--dpi only applies to vector (SVG) inputs")
	}

	if !*nonInteractive {
		return tui.Run(imagePath, src, o)
	}

	// Non-interactive: validate, generate, report.
	if o.Output == "" {
		o.Output = tiler.DefaultOutputName(imagePath)
	}
	layout, err := tiler.ComputeLayout(o, src.Info())
	if err != nil {
		return err
	}
	if err := render.Generate(layout, src, o, o.Output); err != nil {
		return err
	}
	fmt.Printf("Wrote %d pages (%d cols × %d rows, %s %s) to %s\n",
		layout.TotalPages(), layout.Cols, layout.Rows, o.Paper, layout.Orientation, o.Output)
	if !layout.IsVector && layout.EffectiveDPI < 150 {
		fmt.Printf("note: effective resolution is %.0f DPI — the image will look soft at this size\n", layout.EffectiveDPI)
	}
	return nil
}
