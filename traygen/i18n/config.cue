// traygen/i18n/config.cue

defaultLocale: "en"
locales: ["en", "pl"]

// traygen is a CLI with no web target.
targets: ["backend"]

roots: ["locales"]

// Written into the package that embeds it, since go:embed cannot leave its own directory.
outputs: {backend: "../internal/catalog"}

scan: ["..", "../internal/cli"]

go: {package: "catalog"}
