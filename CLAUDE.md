# Tiler

Image tiler for multi-page prints.

# Product

The purpose of this program is to produce a TUI to be able to take any image from the local computer and then provide a series of options for the user to configure and then submit, at which point the program generates a PDF file with a series of pages such that if the user prints all the pages, the user would then be able to tile the pages together, gluing them on their overlap, to create one large production of the original image. Assume borderless printing.

The way the tiles will be put together is by first sticking together all the columns in vertical strips. This is so they can be pasted like wallpaper rolls. This also means that depending on which direction they are going to be brushed, depends which side top or bottom of the vertical overlap should be ok, to avoid snaring the brush. If they are going to be pasted top to bottom then they should have the overlap on the top edge (with the exception of the top piece of each column which would not include an overlap), and inversely if they are going to be brushed from bottom to top then they should have the overlap on the bottom edge (with the exception of the bottom piece of each column which would not include an overlap). Overlaps should not be blank but should have the repeat of what is going to be glued on top of them on them, to minimise any gaps if the overlap gluing is not perfect.

Once the vertical strips / columns have been prepared then they can be pasted either from left to right or right to left, which determines which edge the left/right overlap is on. For left to right the overlap would be on the right hand edge with the exception of the right most column, and for right to left the overlap would be on the left hand side with the exception of the left most column. Again, Overlaps should not be blank but should have the repeat of what is going to be glued on top of them on them, to minimise any gaps if the overlap gluing is not perfect.

The configurable options are:

- Paper size (A4 or A3)
- Glue overlap (in millimetres; default: 1.5cm)
- Width (in centimetres; default: 123cm)
- Brushing direction (upwards or downwards; default: downwards)
- Pasting order (from left or from right; default: from left)
- Output filename (default: <original file name>.tiles.pdf)

Height is always automatic based on width and a preserved image aspect ratio

The program accept the following file types:

- JPG
- PNG
- SVG

The program should optimize for the minimum number of vertical columns, most commonly this will be by making the tiles landscape. There is probably just a few cases where portrait tiles would be more optimal.

# Technology

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
