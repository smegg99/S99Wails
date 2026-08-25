// traygen/sheet.go

package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

const (
	sheetPadding = 8  // px around each icon inside its cell
	sheetLabelH  = 14 // px reserved under each cell for the state name
	sheetGutter  = 64 // px on the left for theme row labels
)

// WriteSheet renders preview.png, a themes x states sheet of primary-size icons
// on a neutral gray background.
func WriteSheet(cfg *Config, variants []Variant, r *Renderer, outDir string) error {
	themes := cfg.ThemeNames()
	states := cfg.StateNames()
	size := cfg.Sizes[0]
	cell := size + 2*sheetPadding

	width := sheetGutter + len(states)*cell
	height := len(themes) * (cell + sheetLabelH)
	sheet := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(sheet, sheet.Bounds(), image.NewUniform(color.RGBA{0x80, 0x80, 0x80, 0xFF}), image.Point{}, draw.Src)

	byKey := map[string]Variant{}
	for _, v := range variants {
		byKey[v.Theme+"/"+v.State] = v
	}

	for row, theme := range themes {
		rowY := row * (cell + sheetLabelH)
		drawLabel(sheet, 4, rowY+cell/2, theme)

		for col, state := range states {
			v, ok := byKey[theme+"/"+state]
			if !ok {
				return fmt.Errorf("missing variant %s/%s", theme, state)
			}
			data, err := r.PNG(v.SVG, size)
			if err != nil {
				return fmt.Errorf("sheet %s: %w", v.Name, err)
			}
			icon, err := png.Decode(bytes.NewReader(data))
			if err != nil {
				return fmt.Errorf("sheet %s: decode: %w", v.Name, err)
			}

			x := sheetGutter + col*cell + sheetPadding
			y := rowY + sheetPadding
			draw.Draw(sheet, image.Rect(x, y, x+size, y+size), icon, icon.Bounds().Min, draw.Over)
			drawLabel(sheet, sheetGutter+col*cell+4, rowY+cell+sheetLabelH-4, state)
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, sheet); err != nil {
		return fmt.Errorf("encode preview: %w", err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "preview.png"), buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write preview.png: %w", err)
	}
	return nil
}

func drawLabel(dst *image.RGBA, x, y int, text string) {
	d := &font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(color.RGBA{0x10, 0x10, 0x10, 0xFF}),
		Face: basicfont.Face7x13,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(text)
}
