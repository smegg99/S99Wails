// examples/traygen/trayicons.cue

prefix: "example-tray"
goPackage: "main"
slots: ["badge"]
platforms: windows: padding: 0
themes: {
	dark: source:  "logo-dark.svg"
	light: source: "logo-light.svg"
	dark: platforms: windows: sizeSources: "16": "logo-dark-small.svg"
	light: platforms: windows: sizeSources: "16": "logo-light-small.svg"
}
states: {
	idle: {}
	running: slots: badge: "#42A5F5"
	error: slots: badge: "#FF0000"
}
