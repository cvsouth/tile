package main

import (
	"os"
	"path/filepath"
	"testing"

	"tile/internal/tile"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	o := tile.DefaultOptions()
	o.Paper = tile.A4
	o.OverlapMM = 22
	o.WidthCM = 88
	o.Brushing = tile.Upwards
	o.Pasting = tile.FromRight
	o.RenderDPI = 150
	o.Labels = false

	if _, err := saveDefaults(dir, o); err != nil {
		t.Fatal(err)
	}
	got := loadDefaults(tile.DefaultOptions(), dir)
	if got.Paper != tile.A4 || got.OverlapMM != 22 || got.WidthCM != 88 ||
		got.Brushing != tile.Upwards || got.Pasting != tile.FromRight ||
		got.RenderDPI != 150 || got.Labels {
		t.Fatalf("round trip mismatch: %+v", got)
	}
}

func TestLoadMissingReturnsBase(t *testing.T) {
	base := tile.DefaultOptions()
	if got := loadDefaults(base, t.TempDir()); got != base {
		t.Errorf("missing file should return base, got %+v", got)
	}
}

func TestLoadMalformedReturnsBase(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, settingsFile), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	base := tile.DefaultOptions()
	if got := loadDefaults(base, dir); got != base {
		t.Errorf("malformed file should return base, got %+v", got)
	}
}

func TestLoadPartialOverridesOnlyPresentFields(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, settingsFile), []byte(`{"widthCM":50}`), 0o644); err != nil {
		t.Fatal(err)
	}
	base := tile.DefaultOptions() // A3, overlap 15, width 123, labels on
	got := loadDefaults(base, dir)
	if got.WidthCM != 50 {
		t.Errorf("width should be overridden to 50, got %g", got.WidthCM)
	}
	if got.Paper != base.Paper || got.OverlapMM != base.OverlapMM || got.Labels != base.Labels {
		t.Errorf("absent fields should stay at base, got %+v", got)
	}
}

func TestLoadHonoursLabelsFalse(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, settingsFile), []byte(`{"labels":false}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got := loadDefaults(tile.DefaultOptions(), dir) // base labels on
	if got.Labels {
		t.Error("explicit labels:false should be honoured")
	}
}
