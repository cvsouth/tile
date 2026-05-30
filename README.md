# tiler

Turn any image into a multi-page, tile-and-glue PDF poster. Print every page,
glue them on their overlaps, and you get one large reproduction of the original.

Borderless (full-bleed) printing is assumed: every page is full paper size and
neighbouring tiles overlap by the glue overlap.

## Build

```sh
go build -o tiler .
```

## Use

Interactive (a TUI to tweak options and see the page plan before generating):

```sh
./tiler path/to/image.png
```

Non-interactive (generate straight away):

```sh
./tiler --non-interactive --width 150 --paper A3 path/to/image.svg
```

Accepted inputs: **JPG, PNG, SVG**. Without `--non-interactive`, the flags below
just seed the TUI's starting values.

## Options

| Flag | Default | Meaning |
|------|---------|---------|
| `--paper` | `A4` | Paper size: `A4` or `A3` |
| `--overlap` | `15` | Glue overlap in millimetres |
| `--width` | `123` | Printed width in centimetres (height follows the image's aspect ratio) |
| `--brushing` | `down` | Brushing direction: `up` or `down` — sets which piece sits on top at a vertical seam |
| `--pasting` | `from-left` | Pasting order: `left` or `right` — sets which strip sits on top at a horizontal seam |
| `--dpi` | `300` | Rasterisation DPI — **SVG inputs only** (rejected for raster) |
| `--labels` | `on` | Faint alignment labels inside the overlap band: `on` or `off` |
| `--output` | `<image>.tiles.pdf` | Output PDF path |
| `--non-interactive` | | Generate without the TUI |

## How it works

The image is laid out as a grid of full-size pages. The orientation that needs
the **fewest vertical columns** is chosen (usually landscape; portrait wins the
occasional column tie when it needs fewer pages). Adjacent pages overlap by
exactly the glue overlap, so the band on the covered piece always repeats the
content the top piece will cover — no blank gap if a glue-up is slightly off.

Which piece sits on top at each seam is set so a brush never meets a raised
edge: vertical seams follow the **brushing** direction, horizontal seams follow
the **pasting** order. Faint labels (`R<row>C<col>` plus a seam guide and an
orientation arrow) are printed inside the hidden band so they disappear once the
poster is assembled.

For raster sources the effective print DPI is shown before generating, with a
warning when the image would be scaled up too far. For SVG sources you choose the
render DPI directly instead.

## Test

```sh
go test ./...
```
