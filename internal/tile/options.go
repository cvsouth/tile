// Package tile holds the pure domain logic for laying out an image across
// multiple printed pages. It performs no I/O: callers supply image metadata
// and receive a Layout describing every tile.
package tile

import "fmt"

// Paper is a supported paper size.
type Paper int

const (
	A4 Paper = iota
	A3
)

// PortraitDims returns the paper's width and height in millimetres in portrait
// orientation (width <= height).
func (p Paper) PortraitDims() (w, h float64) {
	switch p {
	case A3:
		return 297, 420
	default:
		return 210, 297
	}
}

// ShortSide returns the shorter paper dimension in millimetres. The glue
// overlap must stay below this for every orientation to have a positive step.
func (p Paper) ShortSide() float64 {
	w, _ := p.PortraitDims()
	return w
}

func (p Paper) String() string {
	if p == A3 {
		return "A3"
	}
	return "A4"
}

// Orientation is the orientation a tile (and therefore the page) is printed in.
type Orientation int

const (
	Landscape Orientation = iota
	Portrait
)

func (o Orientation) String() string {
	if o == Portrait {
		return "portrait"
	}
	return "landscape"
}

// Brushing is the direction the assembled vertical strips will be brushed, which
// sets which piece sits on top at a vertical seam (and thus where the band is).
type Brushing int

const (
	Downwards Brushing = iota
	Upwards
)

func (b Brushing) String() string {
	if b == Upwards {
		return "upwards"
	}
	return "downwards"
}

// Pasting is the order finished strips are pasted side by side, which sets which
// strip sits on top at a horizontal seam (and thus where the band is).
type Pasting int

const (
	FromLeft Pasting = iota
	FromRight
)

func (p Pasting) String() string {
	if p == FromRight {
		return "from right"
	}
	return "from left"
}

// Options are the user-configurable inputs for a tiling run.
type Options struct {
	Paper     Paper
	OverlapMM float64 // glue overlap in millimetres
	WidthCM   float64 // target printed width in centimetres
	Brushing  Brushing
	Pasting   Pasting
	RenderDPI float64 // rasterisation DPI for vector (SVG) sources
	Labels    bool    // print faint alignment labels inside the overlap band
	Output    string  // output PDF path; empty means derive from the input name
}

// DefaultOptions returns the documented defaults.
func DefaultOptions() Options {
	return Options{
		Paper:     A3,
		OverlapMM: 15,  // 1.5 cm
		WidthCM:   123, // cm
		Brushing:  Downwards,
		Pasting:   FromLeft,
		RenderDPI: 300,
		Labels:    true,
	}
}

// Validate reports whether the options can produce a sane layout.
func (o Options) Validate() error {
	if o.WidthCM <= 0 {
		return fmt.Errorf("width must be greater than 0 cm (got %g)", o.WidthCM)
	}
	short := o.Paper.ShortSide()
	if o.OverlapMM <= 0 {
		return fmt.Errorf("overlap must be greater than 0 mm (got %g)", o.OverlapMM)
	}
	if o.OverlapMM >= short {
		return fmt.Errorf("overlap (%g mm) must be smaller than the %s short side (%g mm)", o.OverlapMM, o.Paper, short)
	}
	if o.RenderDPI <= 0 {
		return fmt.Errorf("render DPI must be greater than 0 (got %g)", o.RenderDPI)
	}
	return nil
}
