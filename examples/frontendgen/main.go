// examples/frontendgen/main.go
//
// Emit the Vue composables into gen/ and list them:
//
//	go generate ./frontendgen
//	go run ./frontendgen
//
//go:generate go tool frontendgen -out gen
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func main() {
	// Anchor on the source file so this works from any working directory.
	_, self, _, _ := runtime.Caller(0)
	gen := filepath.Join(filepath.Dir(self), "gen")

	entries, err := os.ReadDir(gen)
	if err != nil {
		panic("run `go generate ./frontendgen` first: " + err.Error())
	}
	for _, entry := range entries {
		fmt.Println("gen/" + entry.Name())
	}
}
