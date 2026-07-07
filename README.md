# grot

[![ci](https://github.com/qf-studio/grot/actions/workflows/ci.yml/badge.svg)](https://github.com/qf-studio/grot/actions/workflows/ci.yml)
[![release](https://img.shields.io/github/v/release/qf-studio/grot)](https://github.com/qf-studio/grot/releases)
[![license](https://img.shields.io/github/license/qf-studio/grot)](LICENSE)

**btop-style terminal dashboards for Prometheus & Grafana.**

grot renders Prometheus metrics as polished terminal dashboards — braille
charts, gradient meters, threshold-colored stats. Point it at your existing
Grafana dashboard JSON and get the same layout in your terminal.

![grot demo](docs/demo.gif)

> Status: early release, under active development.

## Why

- [grafterm](https://github.com/slok/grafterm) — the only Grafana-ish TUI —
  is abandoned (2019) and limited to termdash's fixed widgets.
- [btop](https://github.com/aristocratos/btop) proves terminal dashboards can
  be beautiful — but it only shows system metrics.
- Nothing renders *your* Grafana dashboards in the terminal. grot does.

## Install

```bash
brew install --cask qf-studio/tap/grot   # macOS / Linux
go install github.com/qf-studio/grot/cmd/grot@latest
```

Prebuilt binaries for darwin/linux (amd64 + arm64) on the
[releases page](https://github.com/qf-studio/grot/releases).

## Quick start

```bash
# Widget gallery with fake data — no Prometheus needed
grot demo --theme tokyo-night

# Works against ANY Prometheus (only uses `up` + scrape_* metrics)
grot run --config examples/prometheus.yaml --prom http://localhost:9090

# Render your existing Grafana dashboard in the terminal
grot run --grafana-json my-dashboard.json --prom http://localhost:9090

# What maps? Validate an import without running it
grot import my-dashboard.json          # summary + warnings
grot import --check my-dashboard.json  # non-zero exit on any warning (CI)

# One static frame to stdout (snapshots, docs, cron)
grot run -c dashboard.yaml --prom http://localhost:9090 --once
```

## Keys

| Key | Action |
|---|---|
| `hjkl` / arrows | move focus |
| `enter` | zoom the focused panel |
| `esc` | close zoom (or the help overlay) |
| `+` / `-` | widen / narrow the time range (5m → 24h presets) |
| `r` | refresh now |
| `t` | cycle theme |
| `?` | help overlay |
| `q` / `ctrl-c` | quit |

## Dashboards

Native configs are small YAML files. Every widget is one PromQL query (or
several) plus presentation:

```yaml
title: my service
theme: pilot          # pilot | tokyo-night | catppuccin-mocha
refresh: 10s
range: 1h

widgets:
  - type: stat                      # stat | gauge | bargauge | timeseries
    title: error rate
    unit: percentunit               # percent | percentunit | currencyUSD |
    sparkline: true                 #   s | ms | bytes | short | <suffix>
    grid: { x: 0, y: 0, w: 6, h: 4 }   # Grafana-style 24-column placement
    thresholds:                     # color by CURRENT value
      - color: green                #   base (no value)
      - value: 0.01
        color: yellow
      - value: 0.05
        color: red
    queries:
      - expr: sum(rate(errors_total[5m])) / sum(rate(requests_total[5m]))

  - type: timeseries
    title: requests / s
    stacked: true
    grid: { x: 6, y: 0, w: 18, h: 8 }
    queries:
      - expr: sum by (code) (rate(requests_total[5m]))
        legend: "{{code}}"          # Grafana-style label templates
```

Notes:

- `grid` is optional — omit it everywhere and grot auto-flows widgets into
  rows with responsive breakpoints (1 column < 80 cols, 2 < 140, 4 beyond).
- **timeseries** always query a range (step auto-sized to the chart width);
  **stat** joins in with `sparkline: true` (value + trend band); gauges and
  bar gauges stay instant. `instant: true` on a query opts out of ranges.
- Threshold colors are semantic (`green`, `yellow`, `red`, `blue`, `text`,
  `gray` — resolved per theme) or raw hex.
- A failing query marks only its own panel — the rest of the grid stays live.

Working examples: [`examples/prometheus.yaml`](examples/prometheus.yaml)
(runs against any Prometheus) and [`examples/pilot.yaml`](examples/pilot.yaml).

## Grafana import

`grot run --grafana-json` / `grot import` accept a dashboard JSON export —
bare or API-wrapped (`{"dashboard": {...}}`).

| Grafana | grot |
|---|---|
| `stat`, `gauge`, `bargauge`, `timeseries` panels | 1:1 widgets |
| `gridPos` | 24-column layout, scaled to your terminal |
| `fieldConfig` unit / decimals / min / max / thresholds | honored |
| stat `graphMode` | `area` → sparkline, `none` → number only |
| custom stacking | stacked chart |
| `targets[].expr` / `legendFormat` / `instant` | queries |
| `time.from: now-<dur>`, `refresh` | initial range & poll interval |
| rows | dropped; children promoted in place |
| anything else | placeholder panel in its grid slot + a warning |

## Terminal notes

- **Color**: 24-bit when your terminal advertises it, degrading to ANSI-256/16
  otherwise. Piped output is colorless by default (`CLICOLOR_FORCE=1` forces
  the ANSI fallback — for faithful captures use a PTY).
- **Braille**: charts use braille dots (2×4 per cell). If your font lacks
  coverage, `--ascii` switches to solid block characters.

## Development

```bash
make build   # build ./bin/grot
make demo    # render the widget gallery
make test    # go test -race
make lint    # golangci-lint
```

Releases are tag-driven: `make release V=x.y.z` pushes the tag and CI
(goreleaser) publishes binaries + the Homebrew cask. The README GIF regenerates
with `vhs docs/demo.tape` (point it at any live Prometheus).

## Roadmap

- Prometheus auth (basic / bearer / TLS)
- Per-breakpoint restacking of imported layouts; 2-D canvas for
  staircase grids
- Config-free explore mode (metric browser, ad-hoc queries)

## License

MIT © QF Studio
