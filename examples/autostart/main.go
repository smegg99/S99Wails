// examples/autostart/main.go

// Round-trips an autostart entry in a temp XDG_CONFIG_HOME, never the real autostart directory.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/smegg99/s99wails/v2/platform/autostart"
)

func main() {
	tmp, err := os.MkdirTemp("", "autostart-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmp)
	os.Setenv("XDG_CONFIG_HOME", tmp)

	opts := autostart.Options{
		Name:        "s99wails-example",
		DisplayName: "S99Wails Example",
		Exec:        []string{"/opt/my app/bin", "--home=$HOME"},
	}

	if err := autostart.Enable(opts); err != nil {
		panic(err)
	}
	on, _ := autostart.Enabled(opts)
	fmt.Println("enabled:", on)

	data, err := os.ReadFile(filepath.Join(tmp, "autostart", "s99wails-example.desktop"))
	if err != nil {
		panic(err)
	}
	fmt.Printf("\n%s", data)

	if err := autostart.Disable(opts); err != nil {
		panic(err)
	}
	on, _ = autostart.Enabled(opts)
	fmt.Println("enabled after disable:", on)
}
