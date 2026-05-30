// Command tiler turns an image into a multi-page, tile-and-glue PDF poster.
//
// Usage:
//
//	tiler [options] <image>          # interactive TUI (flags seed the defaults)
//	tiler --non-interactive [options] <image>
//
// The last-used settings are remembered per working directory in a .tile.json
// file and become the defaults next time, unless overridden by arguments.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tiler/internal/render"
	"tiler/internal/source"
	"tiler/internal/tiler"
	"tiler/internal/tui"
)

const settingsFile = ".tile.json"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "tiler: "+err.Error())
		os.Exit(1)
	}
}

func run(args []string) error {
	cwd, _ := os.Getwd()
	// Built-in defaults, overlaid with any remembered settings for this directory.
	base := loadDefaults(tiler.DefaultOptions(), cwd)

	fs := flag.NewFlagSet("tiler", flag.ContinueOnError)
	fs.Usage = func() {
		const usage = `tiler — image tiler for multi-page tile-and-glue prints

Usage:
  tiler [options] <image.(jpg|jpeg|png|svg)>

With no --non-interactive flag the options below just seed the TUI defaults.
Settings are remembered per directory in ` + settingsFile + ` and reused next time.

Options:
`
		_, _ = fmt.Fprint(fs.Output(), usage)
		fs.PrintDefaults()
	}

	// Flag defaults come from base (built-in defaults + remembered settings), so
	// an unset flag keeps the remembered value and a set flag overrides it.
	paper := fs.String("paper", base.Paper.String(), "paper size: A4 or A3")
	overlap := fs.Float64("overlap", base.OverlapMM, "glue overlap in millimetres")
	width := fs.Float64("width", base.WidthCM, "printed width in centimetres")
	brushing := fs.String("brushing", brushingFlag(base.Brushing), "brushing direction: up or down")
	pasting := fs.String("pasting", pastingFlag(base.Pasting), "pasting order: left or right")
	dpi := fs.Float64("dpi", base.RenderDPI, "render DPI for vector (SVG) sources")
	labels := fs.String("labels", onOff(base.Labels), "alignment labels in the overlap band: on or off")
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

	o := base
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

	if *nonInteractive {
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
	} else {
		used, generated, err := tui.Run(imagePath, src, o)
		if err != nil {
			return err
		}
		if !generated {
			return nil // user quit without generating: nothing to save or report
		}
		o = used
	}

	// Persist the used settings as this directory's defaults, and print them so
	// the run stays in the terminal history for reference.
	layout, err := tiler.ComputeLayout(o, src.Info())
	if err != nil {
		return err
	}
	saved := ""
	if path, serr := saveDefaults(cwd, o); serr != nil {
		fmt.Fprintf(os.Stderr, "tiler: could not save %s: %v\n", settingsFile, serr)
	} else {
		saved = path
	}
	printRun(o, layout, saved)
	return nil
}

// printRun reports the result and the exact settings used (as reusable flags).
func printRun(o tiler.Options, l tiler.Layout, savedPath string) {
	fmt.Printf("Wrote %d pages (%d cols × %d rows, %s %s) to %s\n",
		l.TotalPages(), l.Cols, l.Rows, o.Paper, l.Orientation, o.Output)
	fmt.Printf("Settings: %s\n", settingsArgs(o, l.IsVector))
	if !l.IsVector {
		note := ""
		if l.EffectiveDPI < 150 {
			note = "  (low — the image will look soft at this size)"
		}
		fmt.Printf("Effective resolution: %.0f DPI%s\n", l.EffectiveDPI, note)
	}
	if savedPath != "" {
		fmt.Printf("Defaults saved to %s\n", savedPath)
	}
}

// settingsArgs renders the options as a command line that reproduces the run.
func settingsArgs(o tiler.Options, isVector bool) string {
	parts := []string{
		"--paper " + o.Paper.String(),
		fmt.Sprintf("--overlap %g", o.OverlapMM),
		fmt.Sprintf("--width %g", o.WidthCM),
		"--brushing " + brushingFlag(o.Brushing),
		"--pasting " + pastingFlag(o.Pasting),
		"--labels " + onOff(o.Labels),
	}
	if isVector {
		parts = append(parts, fmt.Sprintf("--dpi %g", o.RenderDPI))
	}
	return strings.Join(parts, " ")
}

// persisted is the on-disk form of the remembered settings (human-editable).
// Output is deliberately not stored: it is derived per image. Labels is a
// pointer so an absent field keeps the built-in default rather than forcing off.
type persisted struct {
	Paper     string  `json:"paper"`
	OverlapMM float64 `json:"overlapMM"`
	WidthCM   float64 `json:"widthCM"`
	Brushing  string  `json:"brushing"`
	Pasting   string  `json:"pasting"`
	RenderDPI float64 `json:"renderDPI"`
	Labels    *bool   `json:"labels"`
}

// loadDefaults overlays any settings stored in dir's .tile.json onto base. A
// missing file is silently ignored; a malformed one is reported but ignored.
func loadDefaults(base tiler.Options, dir string) tiler.Options {
	if dir == "" {
		return base
	}
	data, err := os.ReadFile(filepath.Join(dir, settingsFile))
	if err != nil {
		return base
	}
	var p persisted
	if err := json.Unmarshal(data, &p); err != nil {
		fmt.Fprintf(os.Stderr, "tiler: ignoring malformed %s: %v\n", settingsFile, err)
		return base
	}
	o := base
	if v, err := tiler.ParsePaper(p.Paper); err == nil {
		o.Paper = v
	}
	if v, err := tiler.ParseBrushing(p.Brushing); err == nil {
		o.Brushing = v
	}
	if v, err := tiler.ParsePasting(p.Pasting); err == nil {
		o.Pasting = v
	}
	if p.OverlapMM > 0 {
		o.OverlapMM = p.OverlapMM
	}
	if p.WidthCM > 0 {
		o.WidthCM = p.WidthCM
	}
	if p.RenderDPI > 0 {
		o.RenderDPI = p.RenderDPI
	}
	if p.Labels != nil {
		o.Labels = *p.Labels
	}
	return o
}

// saveDefaults writes the used settings to dir's .tile.json and returns its path.
func saveDefaults(dir string, o tiler.Options) (string, error) {
	if dir == "" {
		return "", fmt.Errorf("no working directory")
	}
	lab := o.Labels
	data, err := json.MarshalIndent(persisted{
		Paper:     o.Paper.String(),
		OverlapMM: o.OverlapMM,
		WidthCM:   o.WidthCM,
		Brushing:  brushingFlag(o.Brushing),
		Pasting:   pastingFlag(o.Pasting),
		RenderDPI: o.RenderDPI,
		Labels:    &lab,
	}, "", "  ")
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, settingsFile)
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

func brushingFlag(b tiler.Brushing) string {
	if b == tiler.Upwards {
		return "up"
	}
	return "down"
}

func pastingFlag(p tiler.Pasting) string {
	if p == tiler.FromRight {
		return "from-right"
	}
	return "from-left"
}
