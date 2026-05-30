# Tile

Turn any image into a multi-page, tile-and-glue PDF poster. Print every page,
glue them on their overlaps, and you get one large reproduction of the original.

Borderless (full-bleed) printing is assumed: every page is full paper size and
neighbouring tiles overlap by the glue overlap.

## Build

```sh
go build -o tile .
```

## Use

Interactive (a TUI to tweak options and see the page plan before generating):

```sh
./tile path/to/image.png
```

Non-interactive (generate straight away):

```sh
./tile --non-interactive --width 150 --paper A3 path/to/image.svg
```

Accepted inputs: **JPG, PNG, SVG**. Without `--non-interactive`, the flags below
just seed the TUI's starting values.

## Options

| Flag | Default | Meaning |
|------|---------|---------|
| `--paper` | `A3` | Paper size: `A4` or `A3` |
| `--overlap` | `15` | Glue overlap in millimetres |
| `--width` | `123` | Printed width in centimetres (height follows the image's aspect ratio) |
| `--brushing` | `down` | Brushing direction: `up` or `down` — sets which piece sits on top at a vertical seam |
| `--pasting` | `from-left` | Pasting order: `left` or `right` — sets which strip sits on top at a horizontal seam |
| `--dpi` | `300` | Rasterisation DPI — **SVG inputs only** (rejected for raster) |
| `--labels` | `on` | Faint alignment labels inside the overlap band: `on` or `off` |
| `--output` | `<image>.tiles.pdf` | Output PDF path |
| `--non-interactive` | | Generate without the TUI |

The defaults above are the built-in values; the *actual* defaults are whatever
was last used in the current directory (see below). Any flag still overrides.

## Remembered settings

After each successful run, tile writes the settings it used to a `.tile.json`
file in the current working directory. The next time it runs from that directory
those become the defaults — so each project folder keeps its own preferences.
Command-line flags (and edits in the TUI) always override, and the file is
human-readable if you want to tweak it by hand. On finishing, tile prints the
exact settings used as a reusable command line, so the run stays in your shell
history for reference.

## How it works

The image is laid out as a grid of full-size pages. The orientation that needs
the **fewest vertical columns** is chosen (usually landscape; portrait wins the
occasional column tie when it needs fewer pages). Adjacent pages overlap by
exactly the glue overlap, so the band on the covered piece always repeats the
content the top piece will cover — no blank gap if a glue-up is slightly off.

Which piece sits on top at each seam is set so a brush never meets a raised
edge: vertical seams follow the **brushing** direction, horizontal seams follow
the **pasting** order. Each tile is labelled `R<row>C<col>` — printed twice, in
white and black side by side, so it stays readable on any background — alongside
a dashed seam guide, all inside the hidden overlap band so they disappear once
the poster is assembled.

For raster sources the effective print DPI is shown before generating, with a
warning when the image would be scaled up too far. For SVG sources you choose the
render DPI directly instead.

## Test

```sh
go test ./...
```
