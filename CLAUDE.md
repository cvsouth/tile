# Tiler

Image tiler for multi-page prints.

## Product

The purpose of this program is to produce a TUI to be able to take any image from the local computer and then provide a series of options for the user to configure and then submit, at which point the program generates a PDF file with a series of pages such that if the user prints all the pages, the user would then be able to tile the pages together, gluing them on their overlap, to create one large production of the original image. Assume borderless printing.

The program should optimize for the minimum number of vertical columns, most commonly this will be by making the tiles landscape. There is probably just a few cases where portrait tiles would be more optimal.

### Overlap model

Every piece overlaps its neighbours by the glue overlap. At each seam one piece sits on top; the piece underneath carries an *overlap band* — a strip, as wide or tall as the overlap, that repeats the image content the top piece will cover. So if a glue-up is slightly misaligned and a sliver of the lower piece peeks out, it still shows the correct image and there is no blank gap. The band always sits on the covered edge, so once assembled it is hidden.

The columns are assembled into vertical strips first (so they can be pasted like wallpaper rolls), then the strips are pasted side by side. Which piece sits on top is chosen so the brush never meets a raised edge.

Vertical seams (within a column) are set by the brushing direction:

- **Downwards (default):** the upper piece sits on top, and the lower piece's overlap band is on its **top** edge (hidden underneath). Brushing downwards, the brush rides off the upper piece downhill onto the lower one, so it never snags. The **top** piece of each column has no band.
- **Upwards:** the mirror image — the lower piece sits on top, the band is on the **bottom** edge, and the **bottom** piece of each column has no band.

Horizontal seams (between finished strips) are set by the pasting order:

- **From left (default):** each column is laid on top of the column to its left, so the band is on each column's **right** edge (hidden under the next strip). The **rightmost** column has no band.
- **From right:** the mirror image — the band is on each column's **left** edge, and the **leftmost** column has no band.

Layout: every page is full paper size (borderless). The effective step between pieces is `paper-dimension − overlap`, so `columns = ceil((width − overlap) / (paperWidth − overlap))` and `rows = ceil((height − overlap) / (paperHeight − overlap))` for the chosen tile orientation. The last row/column simply runs off the image; the leftover page area is left blank.

### Inputs

The configurable options are:

- Paper size (A4 or A3)
- Glue overlap (in millimetres; default: 1.5cm)
- Width (in centimetres; default: 123cm)
- Brushing direction (upwards or downwards; default: downwards)
- Pasting order (from left or from right; default: from left)
- Render DPI for vector (SVG) sources (default: 300)
- Faint alignment labels (on or off; default: on) — prints each tile's label and a placement guide inside the overlap band, so they are hidden once the tiles are assembled
- Output filename (default: <original file name>.tiles.pdf, with the original extension stripped)

Height is always automatic based on width and a preserved image aspect ratio.

The effective print DPI for the current options is shown before submitting, so the user can tell whether a raster source is being scaled up too far.

The program accept the following file types:

- JPG
- PNG
- SVG

### Technology

Write this program in Golang with Bubbletea. The program should be started using `./tiler <relative or path to image>`

The program should also be able to be run non-interactively by passing all the options as arguments plus --non-interactive. Passing arguments without the --non-interactive just changes the defaults for that run.

## Development practices

### Manual experimentation

- Always experiment for real to figure things out for yourself, rather than relying on assumptions, stale comments or training data.

### Manual verification

- Always do deep and thorough exploratory manual testing
- Kill any existing processes and run them yourself to test
- Kill the processes you have run once done
- Clean up verification screenshots after using them

### Functional Core + Imperative Shell

- Keep domain/business logic in **pure functions** with no I/O or side effects
- Put all database/HTTP/external calls in the **shell layer**
- Fetch external data *before* calling pure domain logic — makes core trivially testable without mocks
- Boundaries are about **knowledge**: code inside knows everything, code outside knows only what's exposed

### Anti-Patterns (Avoid)

- **Premature abstraction**: Don't create abstractions before **three** uses
- **Premature optimization**: Use profiling data to find actual bottlenecks
- **Architecture by buzzword**: Choosing microservices/event sourcing/CQRS because fashionable, not because drivers demand them
- **Big design up front**: Design enough for current drivers, build, learn, iterate
- **Astronaut architecture**: Designing for hypothetical scale, requirements, or users
- **Golden hammer**: Same pattern for every problem
- **Over-layering**: Adding abstraction layers for "flexibility" without a concrete driver

### The Testing Mindset

- Testing is about **finding bugs, not confirming correctness** — a test that always passes teaches nothing
- Ask "how could this fail?" before asking "does this work?"
- Test the boundaries, the errors, the unexpected — not just the golden path
- Think like a user who is distracted, impatient, and doesn't read instructions
