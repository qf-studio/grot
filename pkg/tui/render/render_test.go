package render

import (
	"math"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestPanelExactDimensions(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		content string
		w, h    int
	}{
		{"basic", "Title", "hello", 30, 5},
		{"empty content", "T", "", 20, 4},
		{"overflow content", "Long", strings.Repeat("line\n", 20), 25, 6},
		{"long title", strings.Repeat("VERYLONGTITLE", 5), "x", 20, 4},
		{"styled content", "S", lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Render("red text"), 30, 4},
	}
	ps := PanelStyle{Border: lipgloss.NewStyle(), Title: lipgloss.NewStyle()}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := Panel(tt.title, tt.content, tt.w, tt.h, ps)
			lines := strings.Split(out, "\n")
			if len(lines) != tt.h {
				t.Fatalf("got %d lines, want %d", len(lines), tt.h)
			}
			for i, line := range lines {
				if got := lipgloss.Width(line); got != tt.w {
					t.Errorf("line %d: width %d, want %d: %q", i, got, tt.w, line)
				}
			}
		})
	}
}

func TestPadOrTruncate(t *testing.T) {
	tests := []struct {
		in    string
		width int
	}{
		{"short", 20},
		{"exactly-ten", 11},
		{"this is a much longer string than allowed", 15},
		{"", 5},
	}
	for _, tt := range tests {
		out := PadOrTruncate(tt.in, tt.width)
		if got := lipgloss.Width(out); got != tt.width {
			t.Errorf("PadOrTruncate(%q, %d): width %d", tt.in, tt.width, got)
		}
	}
}

func TestTruncateVisualStyled(t *testing.T) {
	// Raw ANSI rather than lipgloss.Render: lipgloss strips color when stdout
	// is not a TTY (test runs), which would make this test vacuous.
	styled := "\x1b[38;2;126;184;218mstyled long content here\x1b[0m"
	out := TruncateVisual(styled, 12)
	if got := lipgloss.Width(out); got != 12 {
		t.Errorf("styled truncate: width %d, want 12", got)
	}
	// Truncation cuts before the string's own closing reset — the result must
	// re-emit one so the ellipsis and following text don't inherit the style.
	if !strings.Contains(out, "\x1b[0m") {
		t.Errorf("styled truncate leaks open style, no reset: %q", out)
	}
	if !strings.HasSuffix(out, "...") {
		t.Errorf("styled truncate missing ellipsis: %q", out)
	}
}

// TestPanelInfoStyledLegendWidths guards the resize-corruption invariant: a
// styled border legend truncated at any panel width must keep every line at
// exactly w cells. An over-wide (or escape-mangled) top border wraps in the
// terminal and cascade-breaks the whole frame below it. Raw ANSI on purpose —
// lipgloss strips color off-TTY, which would make the sweep vacuous.
func TestPanelInfoStyledLegendWidths(t *testing.T) {
	ps := PanelStyle{Border: lipgloss.NewStyle(), Title: lipgloss.NewStyle()}
	info := "\x1b[38;2;126;184;218m●\x1b[0m \x1b[38;2;139;148;158mcost\x1b[0m \x1b[38;2;201;209;217m$9.78\x1b[0m"
	for w := 6; w <= 80; w++ {
		out := PanelInfo("cumulative cost", info, "body", w, 5, ps)
		if out == "" {
			continue // below minimum chrome
		}
		for i, line := range strings.Split(out, "\n") {
			if got := lipgloss.Width(line); got != w {
				t.Fatalf("w=%d line %d: width %d: %q", w, i, got, line)
			}
		}
	}
}

func TestNormalizeSparkline(t *testing.T) {
	// Levels are 1-8, right-aligned, zeros pad left.
	levels := NormalizeSparkline([]float64{0, 5, 10}, 5)
	if len(levels) != 5 {
		t.Fatalf("len %d", len(levels))
	}
	if levels[0] != 0 || levels[1] != 0 {
		t.Errorf("left padding not zero: %v", levels)
	}
	if levels[2] != 1 {
		t.Errorf("zero value should be baseline 1: %v", levels)
	}
	if levels[4] != 8 {
		t.Errorf("max value should be 8: %v", levels)
	}
}

func TestSparklineWidth(t *testing.T) {
	for _, w := range []int{5, 20, 60} {
		out := Sparkline(NormalizeSparkline([]float64{1, 2, 3}, w), w)
		if got := lipgloss.Width(out); got != w {
			t.Errorf("width %d: got %d", w, got)
		}
	}
}

func TestBlockChartDimensions(t *testing.T) {
	vals := []float64{1, 5, 3, 8, 2, 9, 4}
	rows := BlockChart(vals, 20, 4)
	if len(rows) != 4 {
		t.Fatalf("got %d rows, want 4", len(rows))
	}
	for i, r := range rows {
		if got := lipgloss.Width(r); got != 20 {
			t.Errorf("row %d: width %d, want 20", i, got)
		}
	}
	// Max value column must reach the top row with a filled block.
	if !strings.ContainsAny(rows[0], "▁▂▃▄▅▆▇█") {
		t.Errorf("top row empty for max value: %q", rows[0])
	}
}

func TestMeterWidth(t *testing.T) {
	fill := lipgloss.NewStyle()
	empty := lipgloss.NewStyle()
	for _, frac := range []float64{-0.5, 0, 0.33, 0.5, 0.875, 1, 1.5} {
		out := Meter(frac, 16, fill, empty)
		if got := lipgloss.Width(out); got != 16 {
			t.Errorf("frac %v: width %d, want 16", frac, got)
		}
	}
}

func TestFormatCompact(t *testing.T) {
	tests := []struct {
		in   float64
		want string
	}{
		{0, "0"}, {999, "999"}, {1000, "1.0K"}, {57300, "57.3K"}, {1_200_000, "1.2M"},
	}
	for _, tt := range tests {
		if got := FormatCompact(tt.in); got != tt.want {
			t.Errorf("FormatCompact(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// TestTextHelpersNonPositiveWidth guards against the negative-Repeat panic that
// surfaced when a widget was laid out smaller than its chrome during a resize.
func TestTextHelpersNonPositiveWidth(t *testing.T) {
	for _, w := range []int{0, -1, -4, -100} {
		if got := TruncateVisual("no data", w); got != "" {
			t.Errorf("TruncateVisual(_, %d) = %q, want empty", w, got)
		}
		if got := Center("no data", w); got != "" {
			t.Errorf("Center(_, %d) = %q, want empty", w, got)
		}
		if got := PadOrTruncate("no data", w); got != "" {
			t.Errorf("PadOrTruncate(_, %d) = %q, want empty", w, got)
		}
	}
}

// TestBrailleAreaNaNGap verifies NaN samples render as gaps at their true
// position rather than shifting the series or panicking.
func TestBrailleAreaNaNGap(t *testing.T) {
	nan := math.NaN()
	// Real data only in the middle third; NaN on both sides.
	vals := make([]float64, 60)
	for i := range vals {
		if i < 20 || i >= 40 {
			vals[i] = nan
		} else {
			vals[i] = float64(i)
		}
	}
	gr := GradientStyles([]string{"#7aa2f7"}, 6)
	rows := BrailleArea(vals, 40, 6, gr)
	if len(rows) != 6 {
		t.Fatalf("got %d rows, want 6", len(rows))
	}
	joined := ""
	for _, r := range rows {
		joined += r
	}
	if !containsBraille(joined) {
		t.Error("expected braille dots for the finite middle section")
	}
}

// TestScaleAllNaN ensures an all-NaN series produces no dots (blank), no panic.
func TestScaleAllNaN(t *testing.T) {
	nan := math.NaN()
	vals := []float64{nan, nan, nan, nan}
	rows := BrailleArea(vals, 10, 4, GradientStyles([]string{"#fff"}, 4))
	for _, r := range rows {
		if containsBraille(r) {
			t.Errorf("all-NaN series should be blank, got %q", r)
		}
	}
}

func containsBraille(s string) bool {
	for _, r := range s {
		if r >= 0x2801 && r <= 0x28ff {
			return true
		}
	}
	return false
}
