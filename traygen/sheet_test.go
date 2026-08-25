// traygen/sheet_test.go

package main

import (
	"bytes"
	"context"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteSheet(t *testing.T) {
	cfg := loadTestConfig(t)
	variants, err := BuildVariants(cfg, "testdata")
	if err != nil {
		t.Fatal(err)
	}
	r, err := NewRenderer(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	out := t.TempDir()
	if err := WriteSheet(cfg, variants, r, out); err != nil {
		t.Fatalf("WriteSheet: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(out, "preview.png"))
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	// 3 state columns + label gutter, 2 theme rows: just sanity-check the
	// sheet is a real grid, not a 1x1 artifact.
	const cell = 32 + 2*sheetPadding
	wantW := sheetGutter + 3*cell
	wantH := 2 * (cell + sheetLabelH)
	if b := img.Bounds(); b.Dx() != wantW || b.Dy() != wantH {
		t.Errorf("sheet bounds = %v, want %dx%d", b, wantW, wantH)
	}
}
