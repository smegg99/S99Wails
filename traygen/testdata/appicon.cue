prefix: "app-tray"
slots: ["badge"]
themes: {
	dark: source:  "logo-dark.svg"
	light: source: "logo-light.svg"
}
states: {
	idle: {}
	error: slots: badge: "#FF0000"
}
appIcon: {
	source: "logo-dark.svg"
	sizes: [64, 16]
}
