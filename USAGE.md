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
| `v2/platform/notify` | lazy native notification sender, Linux implemented |
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

An `appIcon` block in `trayicons.cue` adds the window icon as `AppIcon`, a PNG
for `application.Options.Icon` (README, "Code generation").

## Tray artwork

The generated `ForStatus` supplies SVG icon themes and native bitmap assets
to `shell/tray.Manager`. Linux sets `IconName` and `IconThemePath` and keeps
`IconPixmap` populated as a fallback. Flat SVG names support hosts without
theme indexing; `index.theme`, scalable and fixed-size status directories
support hosts that select artwork by size. The manager extracts embedded
themes to its own private temporary directory and removes it on close.

| Platform | Default physical sizes | Native asset | Transparent inset |
|---|---|---|---|
| Linux | 32, 22, 24, 48, 64 | SVG theme; 32px PNG pixmap fallback | 0 |
| Windows | 16, 20, 24, 32, 40, 48 | ICO containing every size; PNGs also emitted | 6.25% per edge |
| macOS (`darwin`) | 44, 22 | 44px PNG, downsampled by AppKit on standard-density displays | 9% per edge |

Each profile can set `sizes` and `padding`; the first size supplies its
primary PNG. ICO includes all Windows sizes so Windows can pick its native
resolution. Set macOS `sizes: [22, 44]` if a 22px primary is desired instead.

Keep editable SVGs beside the config, outside generated theme directories:

```cue
platforms: {
    linux: sizes: [32, 22, 24, 48, 64]
    windows: {sizes: [16, 20, 24, 32, 40, 48], padding: 0}
    darwin: {sizes: [44, 22], padding: 0.09, template: false}
}
themes: {
    dark: {
        source: "logo-dark.svg"
        platforms: windows: sizeSources: {
            "16": "logo-dark-small.svg"
            "20": "logo-dark-small.svg"
            "32": "logo-dark-medium.svg"
        }
    }
    light: {
        source: "logo-light.svg"
        platforms: windows: sizeSources: "16": "logo-light-small.svg"
    }
}
```

Reuse one simpler source at several sizes or supply an SVG for each size.
Unspecified sizes use the main artwork. Sources are relative to the config;
all sources must retain the declared `inkscape:label` slots so state colors
and hiding still work. SVGs need a `viewBox` or positive pixel dimensions.
Padding applies after source selection, so set it to zero if the source
already includes the desired margin.
The generator fits artwork by expanding the root `viewBox`, preserving a
single SVG viewport. Nested SVG viewports can parse successfully while
rendering completely transparent in Qt (including KDE's tray renderer).
`TestFitSVGWithQtRenderer` checks the native Qt renderer when a C++ compiler,
pkg-config and Qt6Svg/Qt6Gui development files are installed; it does not
require a display or desktop session.

Source precedence, from lowest to highest: `theme.source`, `state.source`,
`theme.stateSources[state]`, `theme.platforms[platform].source`, platform
`stateSources[state]`, platform `sizeSources[size]`. Linux's scalable/flat
SVG uses the platform source; size-specific SVGs live in fixed-size directories.
A host using flat lookup scales the main SVG and cannot select size overrides.

`platforms.darwin.template: true` generates black pixels preserving alpha.
Pass `TemplateIcon: trayicons.TemplateIcon` to the manager so AppKit tints
them. Template mode removes status colors; use distinct artwork for states
if their meaning depends on color. Default mode preserves colored badges.

Run `go generate` after editing. Inspect `preview-linux.png`,
`preview-windows.png` and `preview-darwin.png` at 100% zoom: every size and
state is shown on both panel themes at actual pixel dimensions. `preview.png`
keeps the primary Linux size overview. `examples/traygen` includes two simple
16px masters that override the Windows source. Never edit generated assets:
regeneration removes stale theme directories and refuses sources inside them.

## Local development

Point at checkouts in your app, never in a release:

```
replace (
	github.com/smegg99/s99wails/v2 => ../S99Wails
	github.com/smegg99/s99wails/traygen/v2 => ../S99Wails/traygen
	github.com/wailsapp/wails/v3 => ../wails/v3
)
```

A `replace` in a dependency's `go.mod` does not apply to your build, so a
released module must pin real tags.
The Wails replacement is currently required for `SystemTray.SetThemedIcon`,
provided by the local fork. Wails beta.20 itself exposes only bitmap tray
updates on Linux; remove the replacement only after a published dependency
provides the native icon-theme API.

## Native notifications

Use `platform/notify` without registering it as a Wails service. Share one
sender between native callers and close it after those callers stop:

```go
sender := notify.New("Example")
defer sender.Close()
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
err := sender.Send(ctx, title, body)
```

The application localizes `title` and `body` and handles `err`. Linux opens a
private connection to the existing session bus on first send, without
starting one, and reconnects after a failed send. `Close` is final. Other
platforms return `errors.ErrUnsupported`.

The generated frontend log bridge takes application message keys:

```ts
const stopLogs = installFrontendLogBridge(LoggerService, {
  error: 'log.frontendUnhandledError',
  rejection: 'log.frontendUnhandledRejection',
})
```

The native logger translates those keys. Diagnostic text stays in structured
fields; call `stopLogs()` when disposing the bridge.

`platform/notify.Desktop.SendAction(ctx, title, body, label, activate)` adds a
default notification action. The callback runs when the desktop emits that
notification's activation; closing it removes the callback. Call `Close` at
application shutdown. The connection and signal reader are shared with `Send`.

## Native workspaces

`shell/windows` keeps account-scoped windows with tabs that move between them.
The app supplies persistence and tab payloads; the library stores no
application data, credentials or document format.

Create the service with `windows.NewService(windows.Options{...})`:

- `Owner` returns the current account scope as an opaque string.
- `Layouts` returns saved window names and geometry; `Drop` removes a closed
  secondary window's saved entry.
- `Keep` is called before close or quit and must save accepted edits durably.
- `Denied` is the error returned for a stale owner; `BlockedKey` is the
  message key sent with `window:blocked` when close or quit is blocked.
- `Width`, `Height`, `MinWidth` and `MinHeight` set default and minimum sizes.

After creating the Wails app:

- `windows.Configure` takes a factory that builds a window from a name and
  optional geometry. Attach input, title, background and geometry listeners
  there.
- `windows.Register` installs a window's close handshake, its cleanup function
  and its close-to-tray preference.
- `windows.RequestQuit` belongs in the application's `ShouldQuit`.
- `windows.SetOwner` switches the account.
- `windows.RegistryOf` returns the window set for tray focus, show and hide.
- `windows.PlaceWindow` clamps saved or torn-off geometry to the available
  monitors and the minimum size.

The main window is named `main`; the library names secondary windows.

On the frontend, `frontendgen -s99ab` emits `tabsBridge.ts` with
`bridgeDesktopTabs`. Create it in a Vue scope and pass the generated window
bindings, an S99AB `useTabs` workspace, the scope identity, layout readiness and
flush, accepted-edit settling, the app's capture and adopt functions, and
localization for blocked transfers and errors. Call `init` after shell setup and
`ready` when the layout or window label changes. Pass `dragHooks` to S99AB's
`UiTabStrip`, and show `failure`, `cancellable`, `retry` and `cancel` in the
app's own UI.

A transfer completes in order: the target adopts and flushes, then calls
`Claim`; the source removes and flushes, then calls `Released`. Go holds the
payload until completion and `Ready` replays unfinished transfers. `Cancel`
works before `Claim`. Transfer state lives in memory only, so the app must
journal accepted edits itself. A window with a tab in transit does not close,
and a failed layout or `Keep` write leaves the window open.

Cross-window hit testing uses GTK3's global pointer and the X11 window tree,
including child surfaces owned by another X client. An unrelated covering
window blocks the target. Dropping on the destination strip chooses a slot;
dropping elsewhere in its client area appends. A drop outside the source strip
still tears off when no other workspace is under the pointer.
The native regression runs with `go test -tags gtk3 ./shell/windows` and needs
`cc`, `pkg-config`, `xvfb-run` and `dbus-run-session`; it owns an isolated X server.
Native Wayland and other unsupported backends use tear-off on pointer release, with
compositor placement; they do not provide cross-window pointer targeting.
