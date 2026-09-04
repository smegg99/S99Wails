module github.com/smegg99/s99wails/examples

go 1.27

require (
	github.com/joho/godotenv v1.5.1
	github.com/smegg99/s99config v1.0.1
	github.com/smegg99/s99logger v1.1.0
	github.com/smegg99/s99wails/v2 v2.0.0
	github.com/wailsapp/wails/v3 v3.0.0-beta.16
)

require (
	cuelabs.dev/go/oci/ociregistry v0.0.0-20260717083115-5eb5795f322a // indirect
	cuelang.org/go v0.17.1 // indirect
	github.com/adrg/xdg v0.5.3 // indirect
	github.com/beevik/etree v1.7.1 // indirect
	github.com/cockroachdb/apd/v3 v3.2.3 // indirect
	github.com/coder/websocket v1.8.15 // indirect
	github.com/danieljoos/wincred v1.2.3 // indirect
	github.com/dave/jennifer v1.7.1 // indirect
	github.com/emicklei/proto v1.14.3 // indirect
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/godbus/dbus/v5 v5.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jchv/go-winloader v0.0.0-20250406163304-c1995be93bd1 // indirect
	github.com/kanrichan/resvg-go v0.0.1 // indirect
	github.com/knadh/koanf/maps v0.1.3 // indirect
	github.com/knadh/koanf/parsers/toml/v2 v2.2.2 // indirect
	github.com/knadh/koanf/parsers/yaml v1.1.1 // indirect
	github.com/knadh/koanf/providers/file v1.2.1 // indirect
	github.com/knadh/koanf/v2 v2.3.6 // indirect
	github.com/mattn/go-colorable v0.1.15 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/nicksnyder/go-i18n/v2 v2.6.1 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/pelletier/go-toml/v2 v2.4.3 // indirect
	github.com/protocolbuffers/txtpbfmt v0.0.0-20260803135053-1fd8a60d1ffc // indirect
	github.com/rogpeppe/go-internal v1.16.0 // indirect
	github.com/smegg99/s99wails/traygen/v2 v2.0.0 // indirect
	github.com/tetratelabs/wazero v1.12.0 // indirect
	github.com/zalando/go-keyring v0.2.8 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/image v0.45.0 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/oauth2 v0.36.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

tool (
	github.com/smegg99/s99wails/traygen/v2
	github.com/smegg99/s99wails/v2/frontendgen
)

replace (
	github.com/smegg99/s99wails/traygen/v2 => ../traygen
	github.com/smegg99/s99wails/v2 => ../
)
