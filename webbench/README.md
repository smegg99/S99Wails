# webbench

Measures a page inside WebKitGTK, the engine a Wails app renders in on Linux,
and compares render-path configurations.

## Use

```sh
go run github.com/smegg99/s99wails/v2/cmd/s99webbench --dir frontend/.output/public --matrix
```

Excerpt (4 of the 8 `DefaultMatrix` rows; the error column is empty here):

```
+-----------------+------+--------+----------+---------+----------+-------+------------+
| configuration   | fps  | p95 ms | worst ms | load ms | style ms | gpu % | vs slowest |
+-----------------+------+--------+----------+---------+----------+-------+------------+
| auto + x11      | 52.4 | 18.0   | 282.0    | 848     | 0.0      | 0.0   | 5.0x       |
| auto + native   | 48.0 | 18.0   | 309.0    | 1210    | 0.0      | 0.0   | 4.6x       |
| legacy + x11    | 10.8 | 202.0  | 319.0    | 320     | 0.0      | 0.0   | 1.0x       |
| legacy + native | 10.5 | 211.0  | 327.0    | 352     | 0.0      | 0.0   | 1.0x       |
+-----------------+------+--------+----------+---------+----------+-------+------------+
```

| flag | meaning |
|---|---|
| `--dir` | serve a built SPA over loopback and measure it |
| `--url` | measure a URL instead (a dev server, a running app's page) |
| `--matrix` | run `DefaultMatrix()`, one child process per configuration, each a live step |
| `--workload` | `page` drives the page's own scroller; `synthetic` builds a 1500-row list first |
| `--frames` | how long to count frames (default 5s) |
| `--width`, `--height` | window size (default 1280 x 820) |
| `--gpu` | `always` (the Wails default), `ondemand`, `never` |
| `--json` | machine-readable output on stdout |
| `--console` | mirror the page's console to stdout |
| `--lang` | `en` or `pl` for the tables and status lines |

Without `--url` or `--dir` the command exits 2; a failed measurement exits 1.

Needs cgo, gtk3, webkit2gtk-4.1 and a display. The window is shown because an
unmapped webview is not composited.

## Metrics

| metric | what it measures |
|---|---|
| `fps`, `p50/p95/worst` | frame timing under the workload; a good p50 with a bad p95 means stalls |
| `style_recalc` | one full style invalidation, mostly the stylesheet size |
| `layout_thrash` | write-then-read forced synchronous layout in a loop |
| `script` | arithmetic and megamorphic property access in JavaScriptCore |
| `reactivity` | the same access through a `Proxy`, as Vue 3 reactivity does |
| `cpu_percent` | CPU per WebKit process (web, GPU, network) during the frame window |

## Several processes per comparison

`WEBKIT_DISABLE_DMABUF_RENDERER` and `GDK_BACKEND` are read during GTK and WebKit
initialisation. `Child` re-executes the caller's binary for one row, and the
child prints one JSON line the parent collects. `Run` returns an error if
called while another run is active.

## Reading a result

A large gap between the `auto` and `legacy` rows is the cost of disabling the
DMABUF renderer; package `env` selects a cheaper rung. When rows are close and
all are slow, `style_recalc`, `layout_thrash` and `reactivity` point at the page.
