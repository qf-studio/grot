package widget

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/qf-studio/grot/pkg/tui/render"
	"github.com/qf-studio/grot/pkg/tui/theme"
)

// BarGauge shows one horizontal bar per series, scaled to the max value:
//
//	opus/in    ██████████▌     57.3K
//	sonnet/in  ██████          31.0K
type BarGauge struct {
	data
	title      string
	Unit       string
	Decimals   *int
	Max        *float64 // explicit scale; nil → max across series
	Thresholds []Threshold
}

// NewBarGauge creates a bar-row widget.
func NewBarGauge(title, unit string) *BarGauge {
	return &BarGauge{title: title, Unit: unit}
}

func (b *BarGauge) Title() string       { return b.title }
func (b *BarGauge) MinSize() (int, int) { return 20, 4 }

func (b *BarGauge) Render(w, h int, th theme.Theme, focused bool) string {
	iw, ih := render.InnerSize(w, h)
	ps := b.panelStyle(th, focused)

	var body string
	switch {
	case b.err != nil:
		body = errorBody(b.err, iw, ih, th)
	case len(b.res.Series) == 0:
		body = noDataBody(iw, ih, th)
	default:
		body = b.body(iw, ih, th)
	}
	return render.Panel(b.title, body, w, h, ps)
}

func (b *BarGauge) body(iw, ih int, th theme.Theme) string {
	series := b.res.Series
	if len(series) > ih {
		series = series[:ih]
	}

	// Column layout: legend | bar | value
	legendW := 0
	values := make([]float64, len(series))
	labels := make([]string, len(series))
	valueTexts := make([]string, len(series))
	maxV := 0.0
	for i, s := range series {
		v, _ := s.Last()
		values[i] = v
		labels[i] = s.Legend
		valueTexts[i] = FormatValue(v, b.Unit, b.Decimals)
		if lw := lipgloss.Width(s.Legend); lw > legendW {
			legendW = lw
		}
		if v > maxV {
			maxV = v
		}
	}
	if b.Max != nil {
		maxV = *b.Max
	}
	if legendW > iw/3 {
		legendW = iw / 3
	}
	valueW := 0
	for _, vt := range valueTexts {
		if w := lipgloss.Width(vt); w > valueW {
			valueW = w
		}
	}
	barW := iw - legendW - valueW - 4 // 2 gaps of 2 spaces
	if barW < 3 {
		barW = 3
	}

	track := lipgloss.NewStyle().Foreground(lipgloss.Color(render.Dim(th.Border, 0.9)))
	rows := make([]string, len(series))
	for i := range series {
		frac := 0.0
		if maxV > 0 {
			frac = values[i] / maxV
		}
		color := thresholdColor(values[i], b.Thresholds, th, th.SeriesColor(i))

		legend := th.LabelStyle().Render(render.PadOrTruncate(labels[i], legendW))
		// btop disks-style segmented bar with a dark track.
		bar := render.SegmentMeter(frac, barW, []string{render.Dim(color, 0.6), color}, track)
		val := th.DimStyle().Render(render.PadOrTruncate(valueTexts[i], valueW))
		rows[i] = legend + "  " + bar + "  " + val
	}

	return vCenter(strings.Join(rows, "\n"), iw, ih)
}
