// Package tui implements the interactive Bubbletea form for configuring and
// generating a tiled PDF.
package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"tiler/internal/render"
	"tiler/internal/source"
	"tiler/internal/tiler"
)

type state int

const (
	editing state = iota
	generating
	done
	failed
)

type field int

const (
	fPaper field = iota
	fOverlap
	fWidth
	fBrushing
	fPasting
	fDPI
	fLabels
	fOutput
	fSubmit
)

// text input slots
const (
	tiOverlap = iota
	tiWidth
	tiDPI
	tiOutput
	tiCount
)

// fieldToInput maps a field to its text-input slot, or -1 if it is a toggle/button.
func fieldToInput(f field) int {
	switch f {
	case fOverlap:
		return tiOverlap
	case fWidth:
		return tiWidth
	case fDPI:
		return tiDPI
	case fOutput:
		return tiOutput
	default:
		return -1
	}
}

// Model is the Bubbletea form state.
type Model struct {
	inputPath string
	src       source.Source
	info      tiler.ImageInfo

	paper    tiler.Paper
	brushing tiler.Brushing
	pasting  tiler.Pasting
	labels   bool

	inputs [tiCount]textinput.Model
	focus  field

	layout    tiler.Layout
	layoutErr error

	state  state
	genErr error

	generated bool          // a PDF was produced this session
	used      tiler.Options // the options that produced it
}

type genResultMsg struct {
	err error
}

// New builds the form, seeded with defaults (CLI flags may have changed them).
func New(inputPath string, src source.Source, def tiler.Options) Model {
	mk := func(val string, limit int) textinput.Model {
		ti := textinput.New()
		ti.SetValue(val)
		ti.CharLimit = limit
		ti.Width = 24
		ti.Prompt = ""
		return ti
	}
	out := def.Output
	if strings.TrimSpace(out) == "" {
		out = tiler.DefaultOutputName(inputPath)
	}
	m := Model{
		inputPath: inputPath,
		src:       src,
		info:      src.Info(),
		paper:     def.Paper,
		brushing:  def.Brushing,
		pasting:   def.Pasting,
		labels:    def.Labels,
		focus:     fPaper,
		state:     editing,
	}
	m.inputs[tiOverlap] = mk(trimFloat(def.OverlapMM), 8)
	m.inputs[tiWidth] = mk(trimFloat(def.WidthCM), 8)
	m.inputs[tiDPI] = mk(trimFloat(def.RenderDPI), 8)
	m.inputs[tiOutput] = mk(out, 256)
	m.refreshFocus()
	m.recompute()
	return m
}

func trimFloat(f float64) string {
	return strconv.FormatFloat(f, 'g', -1, 64)
}

// options reads the current form into an Options, returning a parse error if a
// numeric field is not a valid number.
func (m Model) options() (tiler.Options, error) {
	o := tiler.Options{Paper: m.paper, Brushing: m.brushing, Pasting: m.pasting, Labels: m.labels}
	parse := func(slot int, name string) (float64, error) {
		v := strings.TrimSpace(m.inputs[slot].Value())
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, fmt.Errorf("%s must be a number", name)
		}
		return f, nil
	}
	var err error
	if o.OverlapMM, err = parse(tiOverlap, "overlap"); err != nil {
		return o, err
	}
	if o.WidthCM, err = parse(tiWidth, "width"); err != nil {
		return o, err
	}
	if o.RenderDPI, err = parse(tiDPI, "render DPI"); err != nil {
		return o, err
	}
	out := strings.TrimSpace(m.inputs[tiOutput].Value())
	if out == "" {
		out = tiler.DefaultOutputName(m.inputPath)
	}
	o.Output = out
	return o, nil
}

func (m *Model) recompute() {
	o, err := m.options()
	if err != nil {
		m.layoutErr = err
		return
	}
	l, err := tiler.ComputeLayout(o, m.info)
	m.layout, m.layoutErr = l, err
}

func (m *Model) refreshFocus() {
	active := fieldToInput(m.focus)
	for i := range m.inputs {
		if i == active {
			m.inputs[i].Focus()
		} else {
			m.inputs[i].Blur()
		}
	}
}

// visibleFields lists the fields shown for the current source: the SVG render
// DPI is only offered for vector inputs.
func (m Model) visibleFields() []field {
	fs := []field{fPaper, fOverlap, fWidth, fBrushing, fPasting}
	if m.info.IsVector {
		fs = append(fs, fDPI)
	}
	return append(fs, fLabels, fOutput, fSubmit)
}

func (m *Model) moveFocus(delta int) {
	vis := m.visibleFields()
	idx := 0
	for i, f := range vis {
		if f == m.focus {
			idx = i
			break
		}
	}
	n := len(vis)
	idx = ((idx+delta)%n + n) % n
	m.focus = vis[idx]
	m.refreshFocus()
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd { return textinput.Blink }

// Update satisfies tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case genResultMsg:
		if msg.err != nil {
			m.state = failed
			m.genErr = msg.err
			return m, nil
		}
		// Success: record what was used and exit immediately (no keypress).
		// The caller prints the settings + result to the terminal.
		m.state = done
		m.generated = true
		return m, tea.Quit

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Forward other messages (e.g. blink) to the focused input.
	if active := fieldToInput(m.focus); active >= 0 && m.state == editing {
		var cmd tea.Cmd
		m.inputs[active], cmd = m.inputs[active].Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case generating:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		return m, nil
	case done:
		return m, tea.Quit // any key closes after success
	case failed:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			m.state = editing
			return m, nil
		}
		return m, nil
	}

	// editing
	switch msg.String() {
	case "ctrl+c", "esc":
		return m, tea.Quit
	case "up", "shift+tab":
		m.moveFocus(-1)
		return m, nil
	case "down", "tab":
		m.moveFocus(1)
		return m, nil
	case "enter":
		return m.submit()
	}

	// Toggle handling for non-text fields.
	if fieldToInput(m.focus) < 0 {
		switch msg.String() {
		case "left", "right", "h", "l", " ":
			m.toggle(msg.String())
			m.recompute()
			return m, nil
		}
		return m, nil
	}

	// Text field: forward to the input, then recompute live.
	active := fieldToInput(m.focus)
	var cmd tea.Cmd
	m.inputs[active], cmd = m.inputs[active].Update(msg)
	m.recompute()
	return m, cmd
}

func (m *Model) toggle(key string) {
	forward := key == "right" || key == "l" || key == " "
	switch m.focus {
	case fPaper:
		if m.paper == tiler.A4 {
			m.paper = tiler.A3
		} else {
			m.paper = tiler.A4
		}
	case fBrushing:
		if m.brushing == tiler.Downwards {
			m.brushing = tiler.Upwards
		} else {
			m.brushing = tiler.Downwards
		}
	case fPasting:
		if m.pasting == tiler.FromLeft {
			m.pasting = tiler.FromRight
		} else {
			m.pasting = tiler.FromLeft
		}
	case fLabels:
		m.labels = !m.labels
	}
	_ = forward // two-option toggles flip regardless of direction
}

func (m Model) submit() (tea.Model, tea.Cmd) {
	o, err := m.options()
	if err != nil {
		m.layoutErr = err
		return m, nil
	}
	l, err := tiler.ComputeLayout(o, m.info)
	if err != nil {
		m.layoutErr = err
		return m, nil
	}
	m.layout, m.layoutErr = l, nil
	m.used = o
	m.state = generating
	return m, generateCmd(l, m.src, o)
}

func generateCmd(l tiler.Layout, src source.Source, o tiler.Options) tea.Cmd {
	return func() tea.Msg {
		// Honour an edited render DPI for vector sources (raster sources ignore it).
		if s, ok := src.(source.RenderDPISetter); ok {
			s.SetRenderDPI(o.RenderDPI)
		}
		err := render.Generate(l, src, o, o.Output)
		return genResultMsg{err: err}
	}
}

// Run launches the interactive TUI. It returns the options that were used to
// generate (valid only when generated is true) so the caller can persist and
// report them. The alt-screen keeps the form off the scrollback, leaving only
// the caller's printed summary behind.
func Run(inputPath string, src source.Source, def tiler.Options) (used tiler.Options, generated bool, err error) {
	fm, err := tea.NewProgram(New(inputPath, src, def), tea.WithAltScreen()).Run()
	if err != nil {
		return def, false, err
	}
	m := fm.(Model)
	return m.used, m.generated, nil
}
