prefix: "app-tray"
slots: ["badge"]
platforms: {
	linux: sizes: [32, 24]
	windows: sizes: [32, 24]
}
themes: {
	dark: source:  "logo-dark.svg"
	light: source: "logo-light.svg"
	light: colors: running: badge: "#1E88E5"
}
states: {
	idle: {}
	running: slots: badge: "#42A5F5"
	error: slots: badge: "#FF0000"
}
