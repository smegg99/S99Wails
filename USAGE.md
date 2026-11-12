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
| `v2/platform/runtimestore` | per-user values in the runtime directory, cleared at logout |
| `v2/locale` | native-side strings, plurals, language switching |
| `v2/shell/tray` | theme-aware tray, rebuildable menus, close-to-tray |
| `v2/shell/window` | resize limits |
| `v2/shell/windowstate` | geometry across restarts |
| `v2/shell/windows` | account-scoped windows whose tabs move between them |
| `v2/shell/editkeys` | undo and redo keys reach the page on Linux |
| `v2/shell/mousenav` | mouse back and forward buttons step through history |
| `v2/shell/crash` | error screen for startup failures and crashes, drawn by the app's frontend |
| `v2/services/browser` | open a URL in the real browser |
| `v2/services/logger` | frontend logs into s99logger |
| `v2/services/theme` | theme events to the frontend |
| `v2/services/title` | window title from the frontend |
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

## Render setup

Call `env.Setup(env.Options{})` before creating the Wails application. The zero
options preserve WebKit's native rendering path and the user's environment.
`S99_RENDER=nogbm`, `shm` or `legacy` selects an explicit fallback; `auto` leaves
the native path alone. Set `AppImageRenderPath` when a packaged app requires a
different default on unknown host GPU stacks. `PreferX11` is an explicit option,
since it changes the available refresh rates and monitor scaling under Wayland.
The result reports the selected path and its source for diagnostics.

## Generators

```go
//go:generate go tool traygen --config trayicons.cue
//go:generate go tool frontendgen --out ../frontend/app/composables/s99wails
```

`frontendgen --s99ab` also emits the appearance and tabs bridges, for an app that
extends the `@smegg99/s99ab` layer. Flags take two dashes; `--lang en|pl` (or
`S99WAILS_LANG`) picks the language of the help and the status lines. Exit 2
is a usage error, exit 1 failed work.

`names.ts` holds every event name, service path and query key the composables
share with Go. It is generated from the Go constants (`just names`); the
frontendgen tests fail when it is stale.

`frontendgen` also emits `installDesktopEditKeys`. Install it once in the
frontend alongside the native `shell/editkeys.Watch` registration:

```ts
import { installDesktopEditKeys } from './composables/s99wails/editKeys'

const stop = installDesktopEditKeys()
// Call stop when the frontend owner is disposed (including hot reload).
```

`editkeys.Watch(win, onError)` reports a window it cannot watch to `onError`; a nil
`onError` drops the failure.

The bridge dispatches the native `edit:key` undo/redo event to the focused
control. If no listener takes it, an editable control receives its native undo
or redo. It also prevents WebKitGTK from pasting the primary selection on a
middle release outside the focused text field; a release inside the field
retains its normal paste. The returned function removes both subscriptions.

## Error screen

`shell/crash` opens the app's own frontend at `/?s99crash` with only its
`Service` bound, in a window of `Options.Width` x `Height` that is also its
minimum (pass the main window's minimum). It gathers the report and runs Copy,
Save, Email, Show log file, Restart, Close, Wait, Kill and SetTitle; the
frontend draws the screen, for example with S99AB's `UiCrashScreen`, whose
`title` event goes to `crashActions.setTitle`.

```go
opts := crash.Options{App: application.Options{Name: "App", Icon: icon, Assets: assets}, Version: version, ReportEmail: "me@example.com"}
_ = crash.Watch(opts)                      // first in main
// application.Options{PanicHandler: crash.PanicHandler, ...} for the main app
log := s99logger.New(s99logger.MultiSink(s99logger.MinLevel(console, s99logger.LevelInfo), crash.Sink()), s99logger.Options{MinLevel: s99logger.LevelDebug})
crash.Describe(crash.Details{Lang: "pl", LogFile: logPath})
crash.Show(opts, crash.Failure{Kind: crash.KindConfig, File: path, Err: err.Error()}) // before the main app
// with the main app: register &crash.Heartbeat{}, call stop := crash.Beat() after application.New
// (past its single-instance check); the quit ends it, stop() ends it earlier
crash.Done()                               // last step of a clean shutdown, outside any shutdown deadline
```

`Watch` runs a watcher copy of the program and points `debug.SetCrashOutput`
at a pipe to it. The runtime makes its crash output blocking, so on Linux it gets
its own file description of the pipe and the app's end stays non-blocking: a line
of up to 512 bytes goes in at once when the pipe has room and nothing is queued,
so the lines just before a fatal crash are not lost. Otherwise, and on other
systems always, one writer goroutine sends the queue, and a full queue drops log
lines; no caller ever blocks. `Done` and `PanicHandler` drain the queue for up to
a second, then give the last line another second, so the process exits even when
the watcher stopped reading. The watcher keeps the last 1000 log lines from
`crash.Sink`, so a report has the debug history up to the end.

| The pipe ends with | The watcher shows |
| --- | --- |
| `Done` | nothing |
| no `Beat` yet, as when a second instance hands over to the first | nothing |
| a crash trace, or a panic Wails recovered through `PanicHandler` | `crash`, with the trace |
| a quit Wails started, as when a closer runs out of time or the user kills a slow quit | nothing |
| none of these | `killed`; on Linux `oom` when `oom_kill` in its cgroup's `memory.events` rose |

A native or page beat silent for 15 s opens `hang` with `alive`: Wait hides it
for another 15 s and beats that return hide it too; Kill sends SIGQUIT so the
runtime writes every goroutine's stack into the report, then SIGKILL after
5 s. Windows has no SIGQUIT, so its hang report has no stacks. When Wails
starts to quit, `Beat` tells the watcher, which stops expecting native and page
beats; a quit still running after 30 s opens `hang` the same way, so a quit that
never ends can be killed with its stacks. Close and Restart
are refused while a hung app still runs. A watcher that
gets SIGTERM, as at logout or from a dev runner signalling the process group,
dies quietly.

Email writes the full report to `<user cache>/<app>/reports` (kept 14 days) and
hands it and the log file to the mail client as attachments: `xdg-email` on
Linux when the default mailto handler is Thunderbird, Betterbird, KMail or
Evolution, Simple MAPI (`MAPISendMailW`) on Windows. Any other client, or a
failure within 10 s, gets a `mailto:` with the report in the body.

```ts
import { crashActions, installCrashHeartbeat, isCrashWindow, onCrashReport } from './composables/s99wails'
// isCrashWindow() in the root picks the screen; crashActions.report() loads it and
// onCrashReport follows a hang's update. installCrashHeartbeat() runs in the main window.
```

An `appIcon` block in `trayicons.cue` adds the window icon as `AppIcon`, a PNG
for `application.Options.Icon` (README, "Code generation").

## Update check

`shell/update` checks stable GitHub releases of a public repository and selects
one newer than the running version with an asset for the desktop OS and
architecture. Drafts, prereleases and releases for another platform are skipped.
It only opens a download page; it does not download or install the update.

```go
updates := update.New(update.Options{
    Owner: "smegg99", Repo: "App", Version: version,
    DownloadURL: "https://example.com/#downloads",
    Changelog: changelog, Store: store,
    Enabled: func() bool { return settings.CheckUpdates },
    Existing: hasHistory, OnError: logWarning,
})
// Add application.NewService(updates) to application.Options.Services.
stop := update.Schedule(updates) // after application.New
// Call stop() before closing the installation store on shutdown.
```

`Schedule` checks after 10 seconds, then every 6 hours. It derives its context
from the application and cancels the in-flight request when stopped; `stop`
returns immediately and can be called repeatedly. Calling `Check(ctx)` starts
a manual check, including when the setting is turned on. Overlapping checks do not start another request. Disabled checks, empty versions and non-desktop
platforms make no request; nil `Enabled` means enabled.

A check sends one plain GET to the GitHub API with User-Agent `Repo/Version`.
No install ID, usage data, token or other application data is sent. HTTP and
storage failures reach `OnError`; the check returns its error and emits no
failure event. `APIBase` and `HTTPClient` let tests use a local server.
`DownloadURL` overrides the GitHub release page; an empty value uses that page.

`Store` provides `Load() (Memory, bool, error)` and `Save(Memory) error`.
`Memory.Ignored` is the version the user ignored, without its leading `v`;
`Memory.Notes` is the last running version whose notes were handled. Storage
is read lazily, so construct the service before opening the application store.
A fresh installation records its version silently. When `Existing` reports
prior application data, or the stored notes version is older, `State(locale)`
returns the bundled changelog resolved from `<!-- lang:xx -->` sections, falling
back to English. Empty notes are omitted. Notes work offline and are independent
of `Enabled`. Nil `Existing` means a fresh installation.

`frontendgen` emits `updateActions` and its `UpdateRelease`, `UpdateNotes`,
`UpdateState` and `UpdateActions` types. The calls use generated service names:

| Action | Go method | Result |
| --- | --- | --- |
| `state(locale)` | `State` | current version, optional update and optional notes with their language |
| `ignore(version)` | `Ignore` | persists this ignored version; newer versions can still appear |
| `notesShown()` | `NotesShown` | records the running version and preserves the ignored version |
| `download()` | `Download` | opens the found release's page in the system browser |
| `check()` | `Check` | starts an enabled check now |
| `onAvailable(callback)` | event `s99update:available` | receives a Release; returns an unsubscribe function |

The frontend reads state on load and after events. It owns dialog visibility
and ordering; `shell/update` does not show or raise a window.

## Tray artwork

The generated `ForStatus` supplies SVG icon themes and native bitmap assets
to `shell/tray.Manager`. On Linux the manager is its own StatusNotifierItem
(Wails' Linux tray publishes only a pixmap): it sets `IconName` and
`IconThemePath`, keeps `IconPixmap` populated as a fallback, and publishes the
tooltip. Windows and macOS use the Wails tray. Flat SVG names support hosts without
theme indexing; `index.theme`, scalable and fixed-size status directories
support hosts that select artwork by size. The manager extracts embedded
themes to its own private temporary directory; `Close` returns after the tray is
gone and that directory removed, or after two seconds when a main-thread call is
stuck because `app.Run` already returned; an icon update requested before `Close`
is dropped. `OnError` runs outside the manager's locks, so it may rebuild the
menu; it must not call `Close`, which waits for the goroutine that reports icon
errors. Without an `OnError`, a panic in the app's icon code is raised again so
the crash watcher sees it.

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
)
```

A `replace` in a dependency's `go.mod` does not apply to your build, so a
released module must pin real tags. Wails itself comes from the module proxy
(`v3.0.0`); pin the same version in your app.

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

On the frontend, `frontendgen --s99ab` emits `tabsBridge.ts` with
`bridgeDesktopTabs`. Create it in a Vue scope and pass the generated window
bindings, an S99AB `useTabs` workspace, the scope identity, layout readiness and
flush, accepted-edit settling, the app's capture and adopt functions, and
localization for blocked transfers and errors. Call `init` after shell setup and
`ready` when the layout or window label changes. Pass `dragHooks` to S99AB's
`UiTabStrip`, and show `failure`, `cancellable`, `retry` and `cancel` in the
app's own UI. `closeEmptied` decides whether a secondary window left with only
pinned tabs by a move closes through the normal close handshake (`Close`); main
never does, because its close button hides every window or quits.

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

Native Wayland (GTK3) gives no global pointer and no window positions. Inside
the source window the tab is followed in that window's coordinates. Past it, a
GTK drag with the same-app target `application/x-s99wails-tab` takes over, and
every workspace webview is a drop site that reports its own coordinates, so the
destination strip shows the slot as on X11. WebKit does not see the drag as HTML
drag and drop. A drop outside the app's windows and Escape both tear off, and
the compositor places the new window. A drag GTK breaks (grab lost, timeout or
error) ends with the tab left in place. Drops and drag ends are never lost to a
burst of motion events.

Outside the source window the page hides its own dragged tab and the native drag
carries a PNG of it from `DragImage`: as the drag icon on Wayland, in an
input-less popup on X11. `tab:track` tells the source page whether the pointer
is inside its window and where. Other backends tear off on pointer release
without cross-window targeting.
