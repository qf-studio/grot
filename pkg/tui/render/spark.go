package render

import (
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// lipglossStyle is a local alias to keep chart signatures compact.
type lipglossStyle = lipgloss.Style

func newFgStyle(hex string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(hex))
}

// SparkBlocks maps normalized levels (0-8) to Unicode block elements.
// Index 0 is a space, reserved for padding.
var SparkBlocks = []rune{' ', '▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// NormalizeSparkline scales values to 1-8 levels for sparkline rendering,
// right-aligned within width (left-padded with 0 = blank). More values than
// width keeps the most recent.
func NormalizeSparkline(values []float64, width int) []int {
	result := make([]int, width)
	if len(values) == 0 || width <= 0 {
		return result
	}

	offset := width - len(values)
	if offset < 0 {
		values = values[len(values)-width:]
		offset = 0
	}

	minVal, maxVal := values[0], values[0]
	for _, v := range values[1:] {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	span := maxVal - minVal
	if span == 0 {
		level := 1
		if values[0] > 0 {
			level = 4
		}
		for i := range values {
			result[offset+i] = level
		}
		return result
	}

	for i, v := range values {
		normalized := (v - minVal) / span * 7
		level := int(math.Round(normalized)) + 1
		if v == 0 {
			level = 1
		}
		if level < 1 {
			level = 1
		}
		if level > 8 {
			level = 8
		}
		result[offset+i] = level
	}
	return result
}

// Sparkline renders levels (0-8) as block characters, exactly width cells.
func Sparkline(levels []int, width int) string {
	out := make([]rune, width)
	start := 0
	if len(levels) > width {
		start = len(levels) - width
	}
	pad := width - (len(levels) - start)
	for i := 0; i < pad; i++ {
		out[i] = SparkBlocks[0]
	}
	for i := start; i < len(levels); i++ {
		idx := levels[i]
		if idx < 0 {
			idx = 0
		}
		if idx >= len(SparkBlocks) {
			idx = len(SparkBlocks) - 1
		}
		out[pad+i-start] = SparkBlocks[idx]
	}
	return string(out)
}

// BlockChart renders values as a multi-row block area chart of exactly
// width×rows cells, one string per row (top first). Each column is scaled to
// rows*8 sub-levels; partial tops use ▁▂▃▄▅▆▇, filled cells use █.
func BlockChart(values []float64, width, rows int) []string {
	if rows < 1 {
		rows = 1
	}
	levels := normalizeTo(values, width, rows*8)
	out := make([]string, rows)
	for r := 0; r < rows; r++ {
		rowRunes := make([]rune, width)
		// Sub-level threshold below this row (rows are top-first).
		base := (rows - 1 - r) * 8
		for c := 0; c < width; c++ {
			fill := levels[c] - base
			if fill < 0 {
				fill = 0
			}
			if fill > 8 {
				fill = 8
			}
			rowRunes[c] = SparkBlocks[fill]
		}
		out[r] = string(rowRunes)
	}
	return out
}

// BlockArea renders values as a solid block area chart of exactly width×rows
// cells with a vertical color gradient (btop-style, but dense blocks instead
// of braille dots). Row 0 is the top.
func BlockArea(values []float64, width, rows int, gradient []lipglossStyle) []string {
	chart := BlockChart(values, width, rows)
	for i := range chart {
		chart[i] = rowStyle(gradient, i, rows).Render(chart[i])
	}
	return chart
}

// BlockStacked renders multiple series as a stacked solid area chart of
// exactly width×rows cells. Column totals share one y-scale; each cell is
// colored by the topmost series visible in it. colors cycle per series.
func BlockStacked(series [][]float64, width, rows int, colors []string) []string {
	if width <= 0 || rows <= 0 || len(series) == 0 {
		return blankRows(width, rows)
	}
	maxLevel := rows * 8

	// Resample each series onto width columns.
	cols := make([][]float64, len(series))
	for i, vals := range series {
		cols[i] = resampleCols(vals, width)
	}

	// Shared scale: max column total (stacked charts baseline at 0).
	maxTotal := 0.0
	for c := 0; c < width; c++ {
		total := 0.0
		for i := range cols {
			if cols[i][c] > 0 {
				total += cols[i][c]
			}
		}
		if total > maxTotal {
			maxTotal = total
		}
	}
	if maxTotal == 0 {
		return blankRows(width, rows)
	}

	// Cumulative sub-level heights per column per series.
	cum := make([][]int, width)
	for c := 0; c < width; c++ {
		cum[c] = make([]int, len(series))
		running := 0.0
		for i := range cols {
			v := cols[i][c]
			if v < 0 {
				v = 0
			}
			running += v
			cum[c][i] = int(running / maxTotal * float64(maxLevel))
		}
	}

	styles := make([]lipglossStyle, len(series))
	for i := range series {
		styles[i] = newFgStyle(colors[i%len(colors)])
	}

	out := make([]string, rows)
	for r := 0; r < rows; r++ {
		cellBase := (rows - 1 - r) * 8 // sub-levels below this row
		var b strings.Builder
		c := 0
		for c < width {
			// Determine rune + owning series for this cell.
			ru, owner := stackedCell(cum[c], cellBase)
			start := c
			for c < width {
				r2, o2 := stackedCell(cum[c], cellBase)
				if r2 != ru || o2 != owner {
					break
				}
				c++
			}
			run := strings.Repeat(string(ru), c-start)
			if owner < 0 {
				b.WriteString(run)
			} else {
				b.WriteString(styles[owner].Render(run))
			}
			_ = start
		}
		out[r] = b.String()
	}
	return out
}

// stackedCell computes the block rune and owning series index for a cell
// whose sub-level range is [cellBase, cellBase+8). Returns owner -1 if empty.
func stackedCell(cum []int, cellBase int) (rune, int) {
	total := cum[len(cum)-1]
	fill := total - cellBase
	if fill <= 0 {
		return ' ', -1
	}
	if fill > 8 {
		fill = 8
	}
	// Owner: topmost series covering the midpoint of the filled portion.
	mid := cellBase + (fill+1)/2
	owner := len(cum) - 1
	for i, cv := range cum {
		if cv >= mid {
			owner = i
			break
		}
	}
	return SparkBlocks[fill], owner
}

// resampleCols maps values onto exactly width columns (nearest index),
// right-aligned; missing history pads left with 0.
func resampleCols(values []float64, width int) []float64 {
	out := make([]float64, width)
	n := len(values)
	if n == 0 {
		return out
	}
	cols := width
	if n < cols {
		cols = n
	}
	offset := width - cols
	for i := 0; i < cols; i++ {
		var idx int
		if cols == 1 {
			idx = n - 1
		} else {
			idx = int(math.Round(float64(i) / float64(cols-1) * float64(n-1)))
		}
		out[offset+i] = values[idx]
	}
	return out
}

// normalizeTo scales values to 0..maxLevel, right-aligned within width.
func normalizeTo(values []float64, width, maxLevel int) []int {
	result := make([]int, width)
	if len(values) == 0 || width <= 0 {
		return result
	}
	offset := width - len(values)
	if offset < 0 {
		values = values[len(values)-width:]
		offset = 0
	}
	minVal, maxVal := values[0], values[0]
	for _, v := range values[1:] {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}
	span := maxVal - minVal
	for i, v := range values {
		var level int
		if span == 0 {
			if v > 0 {
				level = maxLevel / 2
			} else {
				level = 1
			}
		} else {
			level = int(math.Round((v-minVal)/span*float64(maxLevel-1))) + 1
			if v == 0 && minVal == 0 {
				level = 1
			}
		}
		if level < 0 {
			level = 0
		}
		if level > maxLevel {
			level = maxLevel
		}
		result[offset+i] = level
	}
	return result
}
