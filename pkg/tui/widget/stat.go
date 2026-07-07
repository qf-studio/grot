package widget

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/qf-studio/grot/pkg/tui/render"
	"github.com/qf-studio/grot/pkg/tui/theme"
)

// Stat shows a single large value with threshold coloring and an optional
// history sparkline underneath (populated when the result carries a range).
type Stat struct {
	data
	title      string
	Unit       string
	Decimals   *int
	Thresholds []Threshold
}

// NewStat creates a stat widget.
func NewStat(title, unit string) *Stat {
	return &Stat{title: title, Unit: unit}
}

func (s *Stat) Title() string       { return s.title }
func (s *Stat) MinSize() (int, int) { return 14, 4 }

func (s *Stat) Render(w, h int, th theme.Theme, focused bool) string {
	iw, ih := render.InnerSize(w, h)
	ps := s.panelStyle(th, focused)

	var body string
	switch {
	case s.err != nil:
		body = errorBody(s.err, iw, ih, th)
	case len(s.res.Series) == 0:
		body = noDataBody(iw, ih, th)
	default:
		body = s.body(iw, ih, th)
	}
	return render.Panel(s.title, body, w, h, ps)
}

func (s *Stat) body(iw, ih int, th theme.Theme) string {
	ser := s.res.Series[0]
	v, ok := ser.Last()
	if !ok {
		return noDataBody(iw, ih, th)
	}

	color := thresholdColor(v, s.Thresholds, th, th.Label)
	valueStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(color))
	value := valueStyle.Render(FormatValue(v, s.Unit, s.Decimals))

	// With history and room: value on top, subdued solid trend underneath.
	// The trend is context, not the star — it stays dim so the number reads
	// first. Otherwise: centered value.
	if len(ser.Points) > 1 && ih >= 3 {
		vals := make([]float64, len(ser.Points))
		for i, p := range ser.Points {
			vals[i] = p.V
		}
		chartRows := ih - 1
		gradient := render.GradientStyles(
			[]string{render.Dim(color, 0.30), render.Dim(color, 0.65)}, chartRows)
		rows := render.BlockArea(vals, iw, chartRows, gradient)
		lines := append([]string{render.Center(value, iw)}, rows...)
		return strings.Join(lines, "\n")
	}

	return vCenter(render.Center(value, iw), iw, ih)
}
