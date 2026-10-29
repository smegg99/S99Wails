// i18n/config.cue

defaultLocale: "en"
locales: ["en", "pl"]

// The module's commands are CLIs with no web target.
targets: ["backend"]

roots: ["locales"]

// Written into the package that embeds it, since go:embed cannot leave its own directory.
outputs: {backend: "../internal/cli/catalog"}

scan: ["../internal/cli", "../frontendgen", "../cmd"]

go: {package: "catalog"}
