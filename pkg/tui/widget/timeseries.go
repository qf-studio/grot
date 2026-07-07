package widget

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/qf-studio/grot/pkg/tui/render"
	"github.com/qf-studio/grot/pkg/tui/theme"
)

// TimeSeries renders series as a solid block area chart (btop-dense) with a
// vertical color gradient. Multiple series render as a stacked area. The
// legend and current values are embedded in the panel's top border; y-scale
// labels sit in a left gutter. Braille rendering is opt-in (font-dependent).
type TimeSeries struct {
	data
	title    string
	Unit     string
	Decimals *int
	Stacked  bool
	Braille  bool // high-res braille dots instead of solid blocks
}

// NewTimeSeries creates a time-series chart widget.
func NewTimeSeries(title, unit string) *TimeSeries {
	return &TimeSeries{title: title, Unit: unit}
}

func (t *TimeSeries) Title() string       { return t.title }
func (t *TimeSeries) MinSize() (int, int) { return 24, 5 }

func (t *TimeSeries) Render(w, h int, th theme.Theme, focused bool) string {
	iw, ih := render.InnerSize(w, h)
	ps := t.panelStyle(th, focused)

	switch {
	case t.err != nil:
		return render.Panel(t.title, errorBody(t.err, iw, ih, th), w, h, ps)
	case len(t.res.Series) == 0 || len(t.res.Series[0].Points) == 0:
		return render.Panel(t.title, noDataBody(iw, ih, th), w, h, ps)
	}
	return render.PanelInfo(t.title, t.legendInfo(th), t.body(iw, ih, th), w, h, ps)
}

func (t *TimeSeries) body(iw, ih int, th theme.Theme) string {
	// Shared min/max for the y-gutter labels.
	minV, maxV, vals := t.collect()

	hiLabel := FormatValue(maxV, t.Unit, t.Decimals)
	loLabel := FormatValue(minV, t.Unit, t.Decimals)
	gutter := len(hiLabel)
	if len(loLabel) > gutter {
		gutter = len(loLabel)
	}
	if gutter > iw/3 {
		gutter = iw / 3
	}
	chartW := iw - gutter - 1
	if chartW < 4 {
		chartW = iw
		gutter = 0
	}

	colors := make([]string, len(vals))
	for i := range vals {
		colors[i] = th.SeriesColor(i)
	}

	var rows []string
	switch {
	case t.Braille:
		rows = render.BrailleMulti(vals, chartW, ih, colors)
	case len(vals) == 1:
		gradient := render.GradientStyles([]string{colors[0]}, ih)
		rows = render.BlockArea(vals[0], chartW, ih, gradient)
	default:
		// Multi-series → stacked solid area, one color per series.
		rows = render.BlockStacked(vals, chartW, ih, colors)
	}

	lines := make([]string, 0, ih)
	for i, row := range rows {
		label := strings.Repeat(" ", gutter)
		if gutter > 0 {
			switch i {
			case 0:
				label = render.PadOrTruncate(hiLabel, gutter)
			case len(rows) - 1:
				label = render.PadOrTruncate(loLabel, gutter)
			}
		}
		sep := ""
		if gutter > 0 {
			sep = " "
		}
		lines = append(lines, th.DimStyle().Render(label)+sep+row)
	}
	return strings.Join(lines, "\n")
}

// collect returns the shared min/max and per-series value slices.
func (t *TimeSeries) collect() (minV, maxV float64, vals [][]float64) {
	first := true
	for _, s := range t.res.Series {
		sv := make([]float64, len(s.Points))
		for i, p := range s.Points {
			sv[i] = p.V
			if first {
				minV, maxV = p.V, p.V
				first = false
				continue
			}
			if p.V < minV {
				minV = p.V
			}
			if p.V > maxV {
				maxV = p.V
			}
		}
		vals = append(vals, sv)
	}
	return minV, maxV, vals
}

// legendInfo builds the border-embedded legend: "● opus 2.1K  ● haiku 800".
func (t *TimeSeries) legendInfo(th theme.Theme) string {
	parts := make([]string, 0, len(t.res.Series))
	for i, s := range t.res.Series {
		name := s.Legend
		if name == "" {
			name = "series"
		}
		part := th.SeriesStyle(i).Render("●") + " " + th.DimStyle().Render(name)
		if v, ok := s.Last(); ok {
			part += " " + th.LabelStyle().Render(FormatValue(v, t.Unit, t.Decimals))
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, "  ")
}

var _ = lipgloss.Width // keep lipgloss import if unused in future edits
