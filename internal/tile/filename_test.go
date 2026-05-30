package tile

import (
	"path/filepath"
	"testing"
)

func TestDefaultOutputName(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"photo.jpg", "photo.tiles.pdf"},
		{"photo.JPG", "photo.tiles.pdf"},
		{"photo.jpeg", "photo.tiles.pdf"},
		{"logo.svg", "logo.tiles.pdf"},
		{"art.PNG", "art.tiles.pdf"},
		{"my.image.v2.png", "my.image.v2.tiles.pdf"},
		{filepath.Join("a", "b", "pic.png"), filepath.Join("a", "b", "pic.tiles.pdf")},
		{"noext", "noext.tiles.pdf"},
		{".png", ".png.tiles.pdf"},                         // dotfile-like: keep the name, don't strip to empty
		{"poster.tiles.pdf", "poster.tiles.pdf.tiles.pdf"}, // .pdf is not a recognised image ext
	}
	for _, c := range cases {
		if got := DefaultOutputName(c.in); got != c.want {
			t.Errorf("DefaultOutputName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
