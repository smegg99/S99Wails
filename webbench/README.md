# webbench

Measures a page inside WebKitGTK, the engine a Wails app renders in on Linux , 
and compares render-path configurations against each other.

It exists for one recurring bug: the frontend is smooth in Chromium, sluggish in
the app, and nothing in the frontend explains it. Usually nothing in the frontend
*is* the reason. The shell picked a render path before the window opened, from an
environment variable copied out of an issue thread, and no amount of profiling
the Vue side will show that.

## Use

```sh
go run github.com/smegg99/s99wails/cmd/s99webbench -dir frontend/.output/public -matrix
```

Excerpt (4 of the 8 `DefaultMatrix` rows):

```
configuration                              fps   p95 ms    worst   load ms  style ms     gpu %
----------------------------------------------------------------------------------------------
auto + x11                                52.4     18.0    282.0       848       0.0       0.0  (5.0x)
auto + native                             48.0     18.0    309.0      1210       0.0       0.0  (4.6x)
legacy + x11                              10.8    202.0    319.0       320       0.0       0.0  (1.0x)
legacy + native                           10.5    211.0    327.0       352       0.0       0.0  (1.0x)
```

| flag | meaning |
|---|---|
| `-dir` | serve a built SPA over loopback and measure it |
| `-url` | measure a URL instead (a dev server, a running app's page) |
| `-matrix` | run `DefaultMatrix()`, one child process per configuration |
| `-workload` | `page` drives the page's own scroller; `synthetic` builds a 1500-row list first |
| `-frames` | how long to count frames (default 5s) |
| `-gpu` | `always` (Wails' own default), `ondemand`, `never` |
| `-json` | machine-readable output |
| `-console` | mirror the page's console, for when the page is what is broken |

Needs cgo, gtk3, webkit2gtk-4.1 and a display. It opens a real window on purpose:
an unmapped one is not composited and reports frame rates that mean nothing.

## What it measures, and why each one is separate

A single "it feels slow" is not actionable and these are:

| metric | what moves it |
|---|---|
| `fps`, `p50/p95/worst` | the render path. A p50 at vsync with a bad p95 is a stall, not a slow page, a mean would hide that. |
| `style_recalc` | one full style invalidation, so the cost of the stylesheet itself. This is the number that moves when a sheet carries thousands of rules the app never renders. |
| `layout_thrash` | write-then-read in a loop, the forced synchronous layout every scroll handler that measures something performs. WebKit pays far more for it than Blink. |
| `script` | arithmetic and megamorphic property access: JavaScriptCore against V8, nothing else in the way. |
| `reactivity` | the same access through a `Proxy`. Vue 3's reactivity is Proxy-based, so it is the closest single number to what a component render costs. |
| `cpu_percent` | per WebKit process. The web process pegged at 100% with a low frame rate means the render path is copying every frame through the CPU, not that the page is heavy. |

## Why comparison means several processes

`WEBKIT_DISABLE_DMABUF_RENDERER` and `GDK_BACKEND` are read during GTK and WebKit
initialisation and cannot be changed afterwards. Two configurations therefore
mean two processes: `Matrix` re-executes the caller's own binary once per row,
and the child prints one JSON line the parent collects.

`Run` refuses a second call in the same process for the same reason, it would
silently measure the first call's settings.

## Reading a result

A large gap between the `auto` and `legacy` rows is the cost of disabling the
DMABUF renderer; package `env` selects a cheaper rung. When rows are close and
all are slow, `style_recalc`, `layout_thrash` and `reactivity` point at the page.
