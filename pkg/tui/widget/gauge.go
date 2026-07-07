package widget

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/qf-studio/grot/pkg/tui/render"
	"github.com/qf-studio/grot/pkg/tui/theme"
)

// Gauge shows a value as a horizontal meter with threshold coloring:
//
//	  66.0%
//	████████▓░░░░░░░
//	0            100
type Gauge struct {
	data
	title      string
	Unit       string
	Decimals   *int
	Min, Max   float64
	Thresholds []Threshold
}

// NewGauge creates a gauge widget with the given range (max 0 → 100).
func NewGauge(title, unit string, min, max float64) *Gauge {
	if max == 0 && min == 0 {
		max = 100
	}
	return &Gauge{title: title, Unit: unit, Min: min, Max: max}
}

func (g *Gauge) Title() string       { return g.title }
func (g *Gauge) MinSize() (int, int) { return 16, 5 }

func (g *Gauge) Render(w, h int, th theme.Theme, focused bool) string {
	iw, ih := render.InnerSize(w, h)
	ps := g.panelStyle(th, focused)

	var body string
	switch {
	case g.err != nil:
		body = errorBody(g.err, iw, ih, th)
	case len(g.res.Series) == 0:
		body = noDataBody(iw, ih, th)
	default:
		body = g.body(iw, ih, th)
	}
	return render.Panel(g.title, body, w, h, ps)
}

func (g *Gauge) body(iw, ih int, th theme.Theme) string {
	v, ok := g.res.Series[0].Last()
	if !ok {
		return noDataBody(iw, ih, th)
	}

	span := g.Max - g.Min
	frac := 0.0
	if span > 0 {
		frac = (v - g.Min) / span
	}

	color := thresholdColor(v, g.Thresholds, th, th.Accent)
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Bold(true)

	// The meter fills in the CURRENT value's threshold color (dim→bright
	// sweep of that one color). A rainbow sweep would read as "the left part
	// of my metric is failing" — the color must answer "how is it now?".
	stops := []string{render.Dim(color, 0.55), color}
	lines := []string{
		render.Center(valueStyle.Render(FormatValue(v, g.Unit, g.Decimals)), iw),
		render.GradientMeter(frac, iw, stops, th.DimMoreStyle()),
	}

	// Min/max scale line when there's room.
	if ih >= 4 {
		lo := FormatValue(g.Min, g.Unit, g.Decimals)
		hi := FormatValue(g.Max, g.Unit, g.Decimals)
		gap := iw - len(lo) - len(hi)
		if gap > 0 {
			lines = append(lines, th.DimStyle().Render(lo+strings.Repeat(" ", gap)+hi))
		}
	}

	return vCenter(strings.Join(lines, "\n"), iw, ih)
}
