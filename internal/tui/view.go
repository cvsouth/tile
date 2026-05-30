package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"tiler/internal/tiler"
)

var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	labelStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	focusStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	valueStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	headingStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("117"))
	errStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	okStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("78"))
	warnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	cursorStyle  = focusStyle.SetString("› ")
	blankCursor  = lipgloss.NewStyle().SetString("  ")
)

const labelWidth = 18

// View satisfies tea.Model.
func (m Model) View() string {
	switch m.state {
	case generating:
		return "\n  " + headingStyle.Render("Generating ") +
			fmt.Sprintf("%d pages → %s …\n", m.layout.TotalPages(), m.resultName()) + "\n"
	case done:
		return "\n  " + okStyle.Render("✓ Wrote ") +
			fmt.Sprintf("%d pages to %s\n", m.layout.TotalPages(), m.resultPath) +
			"  " + dimStyle.Render("Press any key to exit.") + "\n"
	case failed:
		return "\n  " + errStyle.Render("✗ "+m.genErr.Error()) + "\n  " +
			dimStyle.Render("esc to edit · q to quit") + "\n"
	}

	var b strings.Builder
	b.WriteString("\n " + titleStyle.Render("tiler") + dimStyle.Render(" · "+m.inputPath) + "\n\n")

	b.WriteString(m.toggleRow(fPaper, "Paper size", m.paper.String()))
	b.WriteString(m.textRow(fOverlap, "Glue overlap", "mm"))
	b.WriteString(m.textRow(fWidth, "Width", "cm"))
	b.WriteString(m.toggleRow(fBrushing, "Brushing", m.brushing.String()))
	b.WriteString(m.toggleRow(fPasting, "Pasting", m.pasting.String()))
	if m.info.IsVector { // render DPI only applies to vector (SVG) sources
		b.WriteString(m.textRow(fDPI, "Render DPI", ""))
	}
	b.WriteString(m.toggleRow(fLabels, "Alignment labels", onOff(m.labels)))
	b.WriteString(m.textRow(fOutput, "Output", ""))

	b.WriteString("\n")
	b.WriteString(m.submitRow())
	b.WriteString("\n")
	b.WriteString(m.summary())
	b.WriteString("\n " + dimStyle.Render("↑/↓ move · ←/→ change · enter generate · esc quit") + "\n")
	return b.String()
}

func (m Model) cursor(f field) string {
	if m.focus == f {
		return cursorStyle.String()
	}
	return blankCursor.String()
}

func (m Model) label(f field, name string) string {
	st := labelStyle
	if m.focus == f {
		st = focusStyle
	}
	return st.Render(fmt.Sprintf("%-*s", labelWidth, name))
}

func (m Model) toggleRow(f field, name, val string) string {
	left, right := dimStyle.Render("‹ "), dimStyle.Render(" ›")
	v := valueStyle.Render(val)
	if m.focus == f {
		left, right = focusStyle.Render("‹ "), focusStyle.Render(" ›")
		v = focusStyle.Render(val)
	}
	return " " + m.cursor(f) + m.label(f, name) + left + v + right + "\n"
}

func (m Model) textRow(f field, name, unit string) string {
	idx := fieldToInput(f)
	view := m.inputs[idx].View()
	suffix := ""
	if unit != "" {
		suffix = " " + dimStyle.Render(unit)
	}
	box := view
	if m.focus != f {
		box = valueStyle.Render(m.inputs[idx].Value())
	}
	return " " + m.cursor(f) + m.label(f, name) + box + suffix + "\n"
}

func (m Model) submitRow() string {
	btn := " Generate PDF "
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Padding(0, 1)
	if m.focus == fSubmit {
		style = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("232")).Background(lipgloss.Color("212")).Padding(0, 1)
	}
	return " " + m.cursor(fSubmit) + style.Render("["+btn+"]") + "\n"
}

func (m Model) summary() string {
	var b strings.Builder
	b.WriteString(" " + headingStyle.Render("Plan") + "\n")
	if m.layoutErr != nil {
		b.WriteString("  " + errStyle.Render(m.layoutErr.Error()) + "\n")
		return b.String()
	}
	l := m.layout
	row := func(k, v string) {
		b.WriteString("  " + dimStyle.Render(fmt.Sprintf("%-13s", k)) + valueStyle.Render(v) + "\n")
	}
	row("Orientation", fmt.Sprintf("%s (%s)", l.Orientation, m.paper))
	row("Grid", fmt.Sprintf("%d cols × %d rows = %d pages", l.Cols, l.Rows, l.TotalPages()))
	row("Poster", fmt.Sprintf("%.1f × %.1f cm", l.PosterW/10, l.PosterH/10))
	// Effective DPI is only meaningful for raster sources (it warns about
	// upscaling). For vector sources the render DPI is a chosen input instead.
	if !l.IsVector {
		b.WriteString("  " + dimStyle.Render(fmt.Sprintf("%-13s", "Effective DPI")) + dpiText(l) + "\n")
	}
	row("Output", m.resultName())
	return b.String()
}

func (m Model) resultName() string {
	o, err := m.options()
	if err != nil {
		return ""
	}
	return o.Output
}

// dpiText renders the effective print DPI for a raster source, warning when it
// is low enough that the image will be visibly upscaled.
func dpiText(l tiler.Layout) string {
	v := fmt.Sprintf("%.0f", l.EffectiveDPI)
	if l.EffectiveDPI < 150 {
		return warnStyle.Render(v + " — low; the image will look soft printed this large")
	}
	return valueStyle.Render(v)
}

func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}
