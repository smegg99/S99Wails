// examples/accent/main.go

package main

import (
	"fmt"

	"github.com/smegg99/s99wails/accent"
)

func main() {
	info := accent.Snapshot()
	fmt.Println("scheme:", info.Scheme)
	fmt.Println("dark:  ", info.DarkMode)
	fmt.Println("accent:", info.AccentColor)
}
