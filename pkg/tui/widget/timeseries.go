package widget

import (
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/qf-studio/grot/pkg/tui/render"
	"github.com/qf-studio/grot/pkg/tui/theme"
)

// TimeSeries renders series as a braille area chart — the btop texture:
// airy dot-grid fill, one hue per series with a subtle dim→bright vertical
// gradient. Multiple series render stacked with a shared y-scale. The legend
// and current values are embedded in the panel's top border; y-scale labels
// sit in a left gutter. Solid blocks are the fallback for fonts without
// braille coverage.
type TimeSeries struct {
	data
	title    string
	Unit     string
	Decimals *int
	Stacked  bool
	Solid    bool // block-character fallback instead of braille
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

	// Multi-series renders stacked (0..max column total) — labels must match.
	if len(vals) > 1 {
		minV = 0
		maxV = maxStackedTotal(vals)
	}

	hiLabel := FormatValue(maxV, t.Unit, t.Decimals)
	loLabel := FormatValue(minV, t.Unit, t.Decimals)
	gutter := lipgloss.Width(hiLabel)
	if w := lipgloss.Width(loLabel); w > gutter {
		gutter = w
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
	case t.Solid && len(vals) == 1:
		gradient := render.GradientStyles([]string{colors[0]}, ih)
		rows = render.BlockArea(vals[0], chartW, ih, gradient)
	case t.Solid:
		rows = render.BlockStacked(vals, chartW, ih, colors)
	case len(vals) == 1:
		gradient := render.GradientStyles(
			[]string{render.Dim(colors[0], 0.5), colors[0]}, ih)
		rows = render.BrailleArea(vals[0], chartW, ih, gradient)
	default:
		// Multi-series → stacked braille, one flat hue per series.
		rows = render.BrailleStacked(vals, chartW, ih, colors)
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

// maxStackedTotal returns the largest per-index sum across series (series
// from one query_range share timestamps, so index alignment holds).
func maxStackedTotal(vals [][]float64) float64 {
	n := 0
	for _, sv := range vals {
		if len(sv) > n {
			n = len(sv)
		}
	}
	maxT := 0.0
	for i := 0; i < n; i++ {
		total := 0.0
		for _, sv := range vals {
			if i < len(sv) && sv[i] > 0 {
				total += sv[i]
			}
		}
		if total > maxT {
			maxT = total
		}
	}
	return maxT
}

// collect returns the shared min/max (ignoring NaN gaps) and per-series value
// slices. NaN points are kept positionally in the slices so charts preserve the
// time axis; only the scale computation skips them.
func (t *TimeSeries) collect() (minV, maxV float64, vals [][]float64) {
	first := true
	for _, s := range t.res.Series {
		sv := make([]float64, len(s.Points))
		for i, p := range s.Points {
			sv[i] = p.V
			if math.IsNaN(p.V) {
				continue
			}
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
