// Package widget provides grot's renderable dashboard widgets: Stat, Gauge,
// BarGauge, and TimeSeries. Widgets receive query results and render
// themselves into an exact w×h cell rectangle.
package widget

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/qf-studio/grot/pkg/tui/render"
	"github.com/qf-studio/grot/pkg/tui/theme"
)

// Point is a single sample.
type Point struct {
	T time.Time
	V float64
}

// Series is a named sequence of points. Instant queries produce one point.
type Series struct {
	Legend string
	Points []Point
}

// Last returns the most recent finite value, skipping trailing NaN gaps, or 0
// with false when the series has no finite point.
func (s Series) Last() (float64, bool) {
	for i := len(s.Points) - 1; i >= 0; i-- {
		if v := s.Points[i].V; !math.IsNaN(v) {
			return v, true
		}
	}
	return 0, false
}

// QueryResult is the data delivered to a widget.
type QueryResult struct {
	Series    []Series
	FetchedAt time.Time
}

// Threshold colors a value range. Value nil = base color.
type Threshold struct {
	Value *float64
	Color string // semantic name (green/red/...) or hex; resolved via theme
}

// Widget renders into an exact w×h cell rectangle.
type Widget interface {
	Title() string
	SetResult(QueryResult)
	SetError(err error)
	Render(w, h int, th theme.Theme, focused bool) string
	MinSize() (w, h int)
}

// data is the shared state/data-plumbing for all widgets.
type data struct {
	res QueryResult
	err error
}

func (d *data) SetResult(r QueryResult) { d.res = r; d.err = nil }
func (d *data) SetError(err error)      { d.err = err }

// panelStyle picks the frame styles for the current state: focused → accent,
// error → rose, otherwise theme border.
func (d *data) panelStyle(th theme.Theme, focused bool) render.PanelStyle {
	ps := render.PanelStyle{Border: th.BorderStyle(), Title: th.TitleStyle()}
	if d.err != nil {
		ps.Border = th.ErrorStyle()
		ps.Title = th.ErrorStyle().Bold(true)
	}
	if focused {
		ps.Border = th.FocusBorderStyle()
	}
	return ps
}

// errorBody renders an in-panel error message centered in iw×ih.
func errorBody(err error, iw, ih int, th theme.Theme) string {
	msg := render.TruncateVisual(err.Error(), iw)
	return vCenter(th.ErrorStyle().Render(render.Center(msg, iw)), iw, ih)
}

// noDataBody renders the dim "no data" placeholder.
func noDataBody(iw, ih int, th theme.Theme) string {
	return vCenter(th.DimStyle().Render(render.Center("no data", iw)), iw, ih)
}

// vCenter pads content (already ≤ ih lines) with blank lines to vertically
// center it within ih rows.
func vCenter(content string, iw, ih int) string {
	lines := strings.Split(content, "\n")
	if len(lines) >= ih {
		return strings.Join(lines[:ih], "\n")
	}
	top := (ih - len(lines)) / 2
	blank := strings.Repeat(" ", iw)
	out := make([]string, 0, ih)
	for i := 0; i < top; i++ {
		out = append(out, blank)
	}
	out = append(out, lines...)
	for len(out) < ih {
		out = append(out, blank)
	}
	return strings.Join(out, "\n")
}

// thresholdColor resolves the color for value v given thresholds: the color
// of the highest threshold whose Value <= v, regardless of slice order (Grafana
// exports steps sorted, hand-written YAML may not be). A nil-Value threshold is
// the base color, used when no numeric threshold matches. Falls back to fallback.
func thresholdColor(v float64, thresholds []Threshold, th theme.Theme, fallback string) string {
	color := fallback
	matched := false
	bestVal := 0.0
	for _, t := range thresholds {
		if t.Value == nil {
			if !matched {
				color = th.ResolveColor(t.Color)
			}
			continue
		}
		if v >= *t.Value && (!matched || *t.Value >= bestVal) {
			matched = true
			bestVal = *t.Value
			color = th.ResolveColor(t.Color)
		}
	}
	return color
}

// FormatValue formats v per a Grafana-style unit string.
func FormatValue(v float64, unit string, decimals *int) string {
	dec := -1
	if decimals != nil {
		dec = *decimals
	}
	f := func(def int) string {
		d := def
		if dec >= 0 {
			d = dec
		}
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.*f", d, v), "0"), ".")
	}
	switch unit {
	case "percent":
		return f(1) + "%"
	case "percentunit":
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.*f", pick(dec, 1), v*100), "0"), ".") + "%"
	case "currencyUSD":
		return "$" + f(2)
	case "s":
		return formatDuration(v)
	case "ms":
		return formatDuration(v / 1000)
	case "bytes":
		return formatBytes(v)
	case "short", "", "none":
		if dec >= 0 {
			return fmt.Sprintf("%.*f", dec, v)
		}
		return render.FormatCompact(v)
	default:
		return f(2) + " " + unit
	}
}

func pick(dec, def int) int {
	if dec >= 0 {
		return dec
	}
	return def
}

func formatDuration(sec float64) string {
	switch {
	case sec < 0.001:
		return fmt.Sprintf("%.0fµs", sec*1_000_000)
	case sec < 1:
		return fmt.Sprintf("%.0fms", sec*1000)
	case sec < 60:
		return fmt.Sprintf("%.1fs", sec)
	case sec < 3600:
		return fmt.Sprintf("%.1fm", sec/60)
	default:
		return fmt.Sprintf("%.1fh", sec/3600)
	}
}

func formatBytes(b float64) string {
	units := []string{"B", "KiB", "MiB", "GiB", "TiB"}
	i := 0
	for b >= 1024 && i < len(units)-1 {
		b /= 1024
		i++
	}
	return fmt.Sprintf("%.1f%s", b, units[i])
}
