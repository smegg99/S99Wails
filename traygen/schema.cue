// traygen/schema.cue

package main

#HexColor: string & =~"^#[0-9a-fA-F]{6}$"

#Config: {
	// prefix is the base file name, e.g. "example-tray".
	prefix: string & !="" @go(Prefix)
	// goPackage is the package name of the generated icons_gen.go.
	goPackage: string & !="" | *"trayicons" @go(GoPackage)
	// trayAdapter emits the s99wails/tray ForStatus adapter.
	trayAdapter: bool | *true @go(TrayAdapter)
	platforms: #Platforms @go(Platforms)
	// slots declares the inkscape:label'd elements states may recolor.
	// A state that assigns no color to a declared slot hides it.
	slots: [...string] @go(Slots)
	themes: [string]: #Theme
	themes: dark:  _
	themes: light: _
	themes: _ @go(Themes)
	states: [string]: #State
	states: _ @go(States)
	// appIcon renders one more SVG, with no themes and no states, as the
	// application's own icon (window, taskbar).
	appIcon?: #AppIcon @go(AppIcon,optional=nillable)
}

#AppIcon: {
	// source is relative to the config file, like the themes' sources.
	source: string & !="" @go(Source)
	// sizes in px; the first is embedded as AppIcon and gets the bare file name.
	sizes: [...int & >0] | *[256] @go(Sizes,type=[]int)
}

#Theme: {
	source: string & !="" @go(Source)
	// Per-state source overrides, before platform and size overrides.
	stateSources?: [string]: string
	stateSources?: _ @go(StateSources,type=map[string]string)
	// per-theme color overrides: state -> slot -> color
	colors?: [string]: [string]: #HexColor
	colors?: _ @go(Colors,type=map[string]map[string]string)
	// Platform and pixel-size artwork overrides, relative to the config.
	platforms?: [string]: #Sources
	platforms?: _ @go(Platforms)
}

#Platforms: {
	linux: #Platform & {sizes: *[32, 22, 24, 48, 64] | [...int & >0], padding: *0 | number & >=0 & <0.5}
	windows: #Platform & {sizes: *[16, 20, 24, 32, 40, 48] | [...int & >0], padding: *0.0625 | number & >=0 & <0.5}
	darwin: #Platform & {sizes: *[44, 22] | [...int & >0], padding: *0.09 | number & >=0 & <0.5}
}

#Platform: {
	// Physical pixels; the first is the bitmap fallback's primary size.
	sizes: [...int & >0 & <=256] @go(Sizes,type=[]int)
	// Transparent inset as a fraction of the square canvas.
	padding: number & >=0 & <0.5 @go(Padding,type=float64)
	// macOS only: black pixels with the original alpha, tinted by AppKit.
	template: *false | bool @go(Template)
}

#Sources: {
	source?: string & !="" @go(Source)
	stateSources?: [string]: string & !=""
	stateSources?: _ @go(StateSources,type=map[string]string)
	sizeSources?: [string]: string & !=""
	sizeSources?: _ @go(SizeSources,type=map[string]string)
}

#State: {
	// slot -> fill color; omitted declared slots are hidden in this state
	slots?: [string]: #HexColor
	slots?: _ @go(Slots,type=map[string]string)
	// source swaps the base SVG for this state (all themes, unless a
	// theme's stateSources overrides it)
	source?: string @go(Source)
}
