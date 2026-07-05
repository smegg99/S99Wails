prefix: "example-tray"
goPackage: "main"
slots: ["badge"]
themes: {
	dark: source:  "logo-dark.svg"
	light: source: "logo-light.svg"
}
states: {
	idle: {}
	running: slots: badge: "#42A5F5"
	error: slots: badge: "#FF0000"
}
