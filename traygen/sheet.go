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

// WriteSheet renders preview.png, a themes x states grid of primary-size icons on gray.
func WriteSheet(cfg *Config, variants []Variant, r *Renderer, outDir string) error {
	themes := cfg.ThemeNames()
	states := cfg.StateNames()
	size := cfg.Platforms.Linux.Sizes[0]
	cell := size + 2*sheetPadding

	width := sheetGutter + len(states)*cell
	height := len(themes) * (cell + sheetLabelH)
	sheet := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(sheet, sheet.Bounds(), image.NewUniform(color.RGBA{0x80, 0x80, 0x80, 0xFF}), image.Point{}, draw.Src)

	byKey := map[string]Variant{}
	for _, v := range variants {
		if v.Platform == "linux" && v.Size == size {
			byKey[v.Theme+"/"+v.State] = v
		}
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
	return writePlatformSheets(cfg, variants, r, outDir)
}

func writePlatformSheets(cfg *Config, variants []Variant, r *Renderer, outDir string) error {
	for _, platform := range platformNames {
		profile, _ := cfg.Platform(platform)
		maxSize, height := 0, 0
		for _, size := range profile.Sizes {
			maxSize = max(maxSize, size)
			height += len(cfg.Themes) * (size + 2*sheetPadding + sheetLabelH)
		}
		const gutter = 140
		cellWidth := max(maxSize+2*sheetPadding, 80)
		canvas := image.NewRGBA(image.Rect(0, 0, gutter+len(cfg.States)*cellWidth, height))
		byKey := map[string]Variant{}
		for _, v := range variants {
			if v.Platform == platform {
				byKey[fmt.Sprintf("%s/%s/%d", v.Theme, v.State, v.Size)] = v
			}
		}
		y := 0
		for _, theme := range cfg.ThemeNames() {
			for _, size := range profile.Sizes {
				rowH := size + 2*sheetPadding + sheetLabelH
				bg, ink := color.RGBA{0xf2, 0xf2, 0xf2, 255}, color.RGBA{0x20, 0x20, 0x20, 255}
				if theme == "dark" {
					bg, ink = color.RGBA{0x24, 0x24, 0x24, 255}, color.RGBA{0xe7, 0xe5, 0xdf, 255}
				}
				draw.Draw(canvas, image.Rect(0, y, canvas.Bounds().Dx(), y+rowH), image.NewUniform(bg), image.Point{}, draw.Src)
				drawText(canvas, 4, y+sheetPadding+size/2+4, fmt.Sprintf("%s %s %dpx", platform, theme, size), ink)
				for col, state := range cfg.StateNames() {
					v, ok := byKey[fmt.Sprintf("%s/%s/%d", theme, state, size)]
					if !ok {
						return fmt.Errorf("missing preview %s/%s/%s/%d", platform, theme, state, size)
					}
					data, err := r.PNG(v.SVG, size)
					if err != nil {
						return err
					}
					if profile.Template {
						data, err = templatePNG(data)
						if err != nil {
							return err
						}
					}
					icon, err := png.Decode(bytes.NewReader(data))
					if err != nil {
						return err
					}
					if profile.Template {
						// AppKit tints templates to match the menu bar's appearance.
						tinted := image.NewRGBA(icon.Bounds())
						draw.DrawMask(tinted, tinted.Bounds(), image.NewUniform(ink), image.Point{}, icon, image.Point{}, draw.Src)
						icon = tinted
					}
					x := gutter + col*cellWidth + sheetPadding
					draw.Draw(canvas, image.Rect(x, y+sheetPadding, x+size, y+sheetPadding+size), icon, image.Point{}, draw.Over)
					drawText(canvas, gutter+col*cellWidth+4, y+rowH-4, state, ink)
				}
				y += rowH
			}
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, canvas); err != nil {
			return err
		}
		if err := writeFile(filepath.Join(outDir, "preview-"+platform+".png"), buf.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

func drawLabel(dst *image.RGBA, x, y int, text string) {
	drawText(dst, x, y, text, color.RGBA{0x10, 0x10, 0x10, 0xFF})
}

func drawText(dst *image.RGBA, x, y int, text string, ink color.Color) {
	d := &font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(ink),
		Face: basicfont.Face7x13,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(text)
}
