package tui

import (
	"image"
	"path/filepath"
	"testing"

	"tile/internal/tile"
)

// fakeDPISource records the DPI it was told to render at, mimicking a vector
// source.
type fakeDPISource struct {
	dpi      float64
	setCalls int
}

func (f *fakeDPISource) Info() tile.ImageInfo {
	return tile.ImageInfo{AspectRatio: 1, IsVector: true}
}
func (f *fakeDPISource) SetRenderDPI(d float64) { f.dpi = d; f.setCalls++ }
func (f *fakeDPISource) RenderTile(_, _ float64, _ tile.Rect) (image.Image, error) {
	img := image.NewRGBA(image.Rect(0, 0, 80, 80))
	for i := range img.Pix {
		img.Pix[i] = 0xff
	}
	return img, nil
}

// Regression test: an edited render DPI must reach a vector source before the
// PDF is generated (previously the source was loaded once and the edit ignored).
func TestGenerateAppliesEditedDPI(t *testing.T) {
	src := &fakeDPISource{dpi: 300}
	o := tile.DefaultOptions()
	o.WidthCM = 20
	o.RenderDPI = 123
	o.Output = filepath.Join(t.TempDir(), "out.pdf")

	l, err := tile.ComputeLayout(o, src.Info())
	if err != nil {
		t.Fatal(err)
	}
	msg, ok := generateCmd(l, src, o)().(genResultMsg)
	if !ok {
		t.Fatal("unexpected message type")
	}
	if msg.err != nil {
		t.Fatal(msg.err)
	}
	if src.dpi != 123 || src.setCalls == 0 {
		t.Errorf("edited DPI not applied to source: dpi=%g calls=%d", src.dpi, src.setCalls)
	}
}

// submit() must honour the live-edited DPI field end to end.
func TestSubmitHonoursDPIField(t *testing.T) {
	src := &fakeDPISource{dpi: 300}
	def := tile.DefaultOptions()
	def.WidthCM = 20
	def.Output = filepath.Join(t.TempDir(), "out.pdf")
	m := New("art.svg", src, def)
	// edit the DPI text field to 96
	m.inputs[tiDPI].SetValue("96")
	_, cmd := m.submit()
	if cmd == nil {
		t.Fatal("submit produced no command (layout invalid?)")
	}
	if _, ok := cmd().(genResultMsg); !ok {
		t.Fatal("expected genResultMsg")
	}
	if src.dpi != 96 {
		t.Errorf("submit ignored edited DPI: got %g, want 96", src.dpi)
	}
}
