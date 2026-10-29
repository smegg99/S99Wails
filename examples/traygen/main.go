// examples/traygen/main.go

// Reads the icons generated from trayicons.cue:
//
//	go generate ./traygen
//	go run ./traygen
//
//go:generate go tool traygen --config trayicons.cue
package main

import "fmt"

func main() {
	for _, status := range []Status{StatusIdle, StatusRunning, StatusError} {
		for _, dark := range []bool{false, true} {
			data, err := Icon(dark, status)
			if err != nil {
				panic(err)
			}
			fmt.Printf("%-7s dark=%-5v %4d bytes\n", status, dark, len(data))
		}
	}
}
