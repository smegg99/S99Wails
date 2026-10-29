// cmd/s99webbench/main_test.go

//go:build linux

package main

import (
	"bytes"
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/smegg99/s99wails/v2/internal/cli"
	"github.com/smegg99/s99wails/v2/webbench"
)

func newTestApp() (*cli.App, *bytes.Buffer) {
	out := &bytes.Buffer{}
	return cli.New(out, &bytes.Buffer{}, []string{"LANG=C", "NO_COLOR=1", "S99WAILS_LANG=en"}), out
}

func TestATargetIsRequired(t *testing.T) {
	a, out := newTestApp()
	if code := a.Run(context.Background(), newRoot(a), []string{"--matrix"}); code != 2 ||
		!strings.Contains(out.String(), "one of --url or --dir is required") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

// A matrix child gets every flag the parent was given except --matrix.
func TestChildArgsDropOnlyTheMatrix(t *testing.T) {
	a, _ := newTestApp()
	root := newRoot(a)
	if err := root.ParseFlags([]string{"--dir", "dist", "--matrix", "--frames", "3s", "--lang", "pl"}); err != nil {
		t.Fatal(err)
	}
	got := childArgs(root)
	slices.Sort(got)
	want := []string{"--dir=dist", "--frames=3s", "--lang=pl"}
	if !slices.Equal(got, want) {
		t.Fatalf("child args %q, want %q", got, want)
	}
}

func TestComparisonRanksByFrameRateAgainstTheSlowest(t *testing.T) {
	a, _ := newTestApp()
	table := comparison(a, []webbench.Result{
		{Label: "legacy + x11", FPS: 10},
		{Label: "auto + x11", FPS: 50, CPU: map[string]float64{"WebKitGPUProcess": 12.5}},
		{Label: "shm + x11", Error: "child: exit status 1"},
	})
	auto, legacy, shm := strings.Index(table, "auto + x11"), strings.Index(table, "legacy + x11"), strings.Index(table, "shm + x11")
	if auto < 0 || legacy < auto || shm < legacy {
		t.Fatalf("rows out of order:\n%s", table)
	}
	for _, want := range []string{"configuration", "vs slowest", "5.0x", "1.0x", "12.5", "child: exit status 1"} {
		if !strings.Contains(table, want) {
			t.Errorf("table lacks %q:\n%s", want, table)
		}
	}
}

func TestResultRowsAreLocalized(t *testing.T) {
	a, _ := newTestApp()
	a.Run(context.Background(), newRoot(a), []string{"--lang", "pl", "--help"})
	block := result(a, webbench.Result{Label: "page", FPS: 59.9, CPU: map[string]float64{"WebKitWebProcess": 30}})
	for _, want := range []string{"page", "klatki na sekundę", "59.9", "CPU WebKitWebProcess", "30.0"} {
		if !strings.Contains(block, want) {
			t.Errorf("block lacks %q:\n%s", want, block)
		}
	}
}
