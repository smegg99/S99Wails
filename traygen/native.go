// traygen/native.go

package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"strconv"
	"strings"

	"github.com/beevik/etree"
)

// FitSVG preserves the artwork's aspect ratio inside a padded square viewport.
func FitSVG(data []byte, size int, padding float64) ([]byte, error) {
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(data); err != nil {
		return nil, err
	}
	root := doc.Root()
	if root == nil || root.Tag != "svg" {
		return nil, fmt.Errorf("missing SVG root")
	}
	viewBox := root.SelectAttrValue("viewBox", "")
	if viewBox == "" {
		values := []string{"0", "0"}
		for _, name := range []string{"width", "height"} {
			value, err := strconv.ParseFloat(strings.TrimSuffix(root.SelectAttrValue(name, ""), "px"), 64)
			if err != nil || value <= 0 {
				return nil, fmt.Errorf("SVG needs a viewBox or positive pixel dimensions")
			}
			values = append(values, strconv.FormatFloat(value, 'g', -1, 64))
		}
		viewBox = strings.Join(values, " ")
	}
	fields := strings.FieldsFunc(viewBox, func(r rune) bool { return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r' })
	if len(fields) != 4 {
		return nil, fmt.Errorf("SVG viewBox needs four numbers")
	}
	var bounds [4]float64
	for i, field := range fields {
		value, err := strconv.ParseFloat(field, 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			return nil, fmt.Errorf("invalid SVG viewBox number %q", field)
		}
		bounds[i] = value
	}
	if bounds[2] <= 0 || bounds[3] <= 0 {
		return nil, fmt.Errorf("SVG viewBox needs positive dimensions")
	}
	// Expand the existing viewport: nested SVG viewports render blank in Qt.
	side := math.Max(bounds[2], bounds[3]) / (1 - 2*padding)
	x := bounds[0] - (side-bounds[2])/2
	y := bounds[1] - (side-bounds[3])/2
	root.CreateAttr("viewBox", fmt.Sprintf("%g %g %g %g", x, y, side, side))
	root.CreateAttr("width", strconv.Itoa(size))
	root.CreateAttr("height", strconv.Itoa(size))
	root.CreateAttr("preserveAspectRatio", "xMidYMid meet")
	return doc.WriteToBytes()
}

type iconFrame struct {
	Size int
	PNG  []byte
}

func encodeICO(frames []iconFrame) ([]byte, error) {
	if len(frames) == 0 || len(frames) > 65535 {
		return nil, fmt.Errorf("ICO needs between 1 and 65535 frames")
	}
	header := make([]byte, 6+16*len(frames))
	binary.LittleEndian.PutUint16(header[2:], 1)
	binary.LittleEndian.PutUint16(header[4:], uint16(len(frames)))
	offset := len(header)
	for i, frame := range frames {
		if frame.Size < 1 || frame.Size > 256 {
			return nil, fmt.Errorf("invalid ICO frame size %d", frame.Size)
		}
		entry := header[6+i*16:]
		entry[0], entry[1] = byte(frame.Size), byte(frame.Size)
		binary.LittleEndian.PutUint16(entry[4:], 1)
		binary.LittleEndian.PutUint16(entry[6:], 32)
		binary.LittleEndian.PutUint32(entry[8:], uint32(len(frame.PNG)))
		binary.LittleEndian.PutUint32(entry[12:], uint32(offset))
		offset += len(frame.PNG)
	}
	data := append([]byte(nil), header...)
	for _, frame := range frames {
		data = append(data, frame.PNG...)
	}
	return data, nil
}

func templatePNG(data []byte) ([]byte, error) {
	source, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	out := image.NewNRGBA(source.Bounds())
	for y := source.Bounds().Min.Y; y < source.Bounds().Max.Y; y++ {
		for x := source.Bounds().Min.X; x < source.Bounds().Max.X; x++ {
			_, _, _, a := source.At(x, y).RGBA()
			out.SetNRGBA(x, y, color.NRGBA{A: uint8(a >> 8)})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, out); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
