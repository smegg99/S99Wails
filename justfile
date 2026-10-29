# List available commands.
default:
    @just --list

# Format, vet and test every module, then the TypeScript half.
check: fmt vet test test-frontendgen lint typecheck check-version check-frontendgen check-traygen

# Exercise emitted helper behavior without a browser or native runtime.
test-frontendgen:
    pnpm test

# Format the Go files in every module.
fmt:
    # gofmt ignores module boundaries, so one call covers all three modules.
    gofmt -l -w .

# Run go vet in every module.
vet:
    #!/usr/bin/env bash
    set -euo pipefail
    for mod in . traygen examples; do
        (cd "{{justfile_directory()}}/$mod" && go vet ./...)
    done

# Run the tests in every module.
test:
    #!/usr/bin/env bash
    set -euo pipefail
    for mod in . traygen examples; do
        (cd "{{justfile_directory()}}/$mod" && go test ./...)
    done

# Lint and check formatting on the TypeScript assets. gofmt covers the Go.
lint:
    pnpm lint

# Fix everything eslint can fix by itself.
fix:
    pnpm lint --fix

# Typecheck the composables as an app receives them.
typecheck:
    #!/usr/bin/env bash
    # appearanceBridge.ts imports ./systemTheme, which resolves only after
    # frontendgen flattens assets/ and integrations/ into one directory.
    set -euo pipefail
    out="{{justfile_directory()}}/.frontendgen-check"
    rm -rf "$out"
    (cd "{{justfile_directory()}}/examples" && go tool frontendgen -out "$out" -s99ab >/dev/null)
    pnpm typecheck

# Set the version and sync generated metadata.
set-version ver:
    node scripts/sync-version.mjs "{{ver}}"

# Fail when package.json has drifted from VERSION.
check-version:
    node scripts/sync-version.mjs --check

# Regenerate frontendgen/assets/names.ts from the Go constants it shares.
names:
    go test ./frontendgen -run TestNamesFollowTheGoConstants -update

# Regenerate the worked example's emitted composables.
frontendgen:
    cd examples && go generate ./frontendgen

# Regenerate the worked example's tray icons, accessors and contact sheet.
traygen:
    cd examples && go generate ./traygen

# Fail when the committed example output has drifted from the assets.
check-frontendgen:
    #!/usr/bin/env bash
    # The assets are embedded, so committed output drifts until regenerated.
    set -euo pipefail
    tmp="$(mktemp -d)"
    trap 'rm -rf "$tmp"' EXIT
    (cd "{{justfile_directory()}}/examples" && go tool frontendgen -out "$tmp" >/dev/null)
    diff -ru "{{justfile_directory()}}/examples/frontendgen/gen" "$tmp"

# Fail when the committed example icons have drifted from the SVGs and the CUE.
check-traygen:
    #!/usr/bin/env bash
    # A renderer update changes the PNG output without touching the sources.
    set -euo pipefail
    tmp="$(mktemp -d)"
    trap 'rm -rf "$tmp"' EXIT
    (cd "{{justfile_directory()}}/examples" \
        && go tool traygen -config traygen/trayicons.cue -out "$tmp" >/dev/null)
    for item in dark light icons_gen.go preview.png preview-linux.png preview-windows.png preview-darwin.png; do
        diff -ru "{{justfile_directory()}}/examples/traygen/$item" "$tmp/$item"
    done
