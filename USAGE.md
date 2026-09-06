# Using s99wails

## Credentials

Set both. `GOPRIVATE` alone makes Go fetch over direct git, which then has no
credentials.

```sh
go env -w GOPRIVATE='github.com/smegg99/*'
git config --global url."ssh://git@github.com/smegg99/".insteadOf "https://github.com/smegg99/"
```

## Import

```sh
go get github.com/smegg99/s99wails/v2
go get -tool github.com/smegg99/s99wails/v2/frontendgen
go get -tool github.com/smegg99/s99wails/traygen/v2
```

| Import | What |
| --- | --- |
| `v2/platform/accent` | OS accent colour and dark mode, with a watcher |
| `v2/platform/autostart` | launch on login, per user |
| `v2/platform/env` | webview workarounds, call before the GUI starts |
| `v2/locale` | native-side strings, plurals, language switching |
| `v2/shell/tray` | theme-aware tray, rebuildable menus, close-to-tray |
| `v2/shell/window` | resize limits |
| `v2/shell/windowstate` | geometry across restarts |
| `v2/services/browser` | open a URL in the real browser |
| `v2/services/logger` | frontend logs into s99logger |
| `v2/services/theme` | theme events to the frontend |
| `v2/services/title` | window title from the frontend |
| `v2/proxy` | a daemon's API at same-origin `/api` over a unix socket |
| `v2/webbench` | render-path benchmark, Linux only |

`platform/*` and `locale` need no Wails and cross-compile. `shell/*`,
`services/*` and `webbench` need the webview stack.

## Build

```sh
go build -tags gtk3 -o bin/app .
```

Needs GTK 3 and WebKit2GTK 4.1 dev headers, CGO and a C compiler. The GTK 4
stack renders a blank window on many GPUs.

Generate bindings with the pinned module, not a `wails3` on `PATH`: another
version writes bindings the runtime does not match.

```sh
go run github.com/wailsapp/wails/v3/cmd/wails3 generate bindings -d frontend/bindings -ts
```

## Generators

```go
//go:generate go tool traygen -config trayicons.cue
//go:generate go tool frontendgen -out ../frontend/app/composables/s99wails
```

`frontendgen -s99ab` also emits the appearance bridge, for an app that extends
the `@smegg99/s99ab` layer.

## Local development

Point at checkouts in your app, never in a release:

```
replace (
	github.com/smegg99/s99wails/v2 => ../S99Wails
	github.com/smegg99/s99wails/traygen/v2 => ../S99Wails/traygen
)
```

A `replace` in a dependency's `go.mod` does not apply to your build, so a
released module must pin real tags.
