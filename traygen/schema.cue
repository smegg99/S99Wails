package main

#HexColor: string & =~"^#[0-9a-fA-F]{6}$"

#Config: {
	// prefix is the base file name, e.g. "example-tray".
	prefix: string & !="" @go(Prefix)
	// goPackage is the package name of the generated icons_gen.go.
	goPackage: string & !="" | *"trayicons" @go(GoPackage)
	// trayAdapter emits the s99wails/tray ForStatus adapter.
	trayAdapter: bool | *true @go(TrayAdapter)
	// sizes in px; the first is primary and gets the bare file name.
	sizes: [...int & >0] | *[32] @go(Sizes,type=[]int)
	// slots declares the inkscape:label'd elements states may recolor.
	// A state that assigns no color to a declared slot hides it.
	slots: [...string] @go(Slots)
	themes: [string]: #Theme
	themes: dark:  _
	themes: light: _
	themes: _ @go(Themes)
	states: [string]: #State
	states: _ @go(States)
}

#Theme: {
	source: string & !="" @go(Source)
	// per-state source overrides for this theme (highest precedence)
	stateSources?: [string]: string
	stateSources?: _ @go(StateSources,type=map[string]string)
	// per-theme color overrides: state -> slot -> color
	colors?: [string]: [string]: #HexColor
	colors?: _ @go(Colors,type=map[string]map[string]string)
}

#State: {
	// slot -> fill color; omitted declared slots are hidden in this state
	slots?: [string]: #HexColor
	slots?: _ @go(Slots,type=map[string]string)
	// source swaps the base SVG for this state (all themes, unless a
	// theme's stateSources overrides it)
	source?: string @go(Source)
}
