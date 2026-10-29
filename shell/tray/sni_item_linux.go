//go:build linux

// shell/tray/sni_item_linux.go

package tray

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"

	"github.com/godbus/dbus/v5/prop"
)

// pixmap is one StatusNotifierItem image: ARGB32 pixels in network byte order.
type pixmap struct {
	Width, Height int32
	Pixels        []byte
}

// tooltip is the StatusNotifierItem ToolTip: icon name, icons, title, text.
type tooltip struct {
	IconName string
	Icons    []pixmap
	Title    string
	Text     string
}

func (s *sni) itemSpec() map[string]map[string]*prop.Prop {
	read := func(v any) *prop.Prop { return &prop.Prop{Value: v, Emit: prop.EmitTrue} }
	return map[string]map[string]*prop.Prop{itemIface: {
		"Category":            read("ApplicationStatus"),
		"Id":                  read(s.name),
		"Title":               read(s.name),
		"Status":              read("Active"),
		"WindowId":            read(int32(0)),
		"IconName":            read(s.iconName),
		"IconThemePath":       read(s.themePath),
		"IconPixmap":          read(s.icons),
		"OverlayIconName":     read(""),
		"OverlayIconPixmap":   read([]pixmap{}),
		"AttentionIconName":   read(""),
		"AttentionIconPixmap": read([]pixmap{}),
		"AttentionMovieName":  read(""),
		"ToolTip":             read(s.tooltip()),
		"ItemIsMenu":          read(false),
		"Menu":                read(menuPath),
	}}
}

// tooltip titles the tooltip with the application and puts the text under it.
func (s *sni) tooltip() tooltip {
	return tooltip{Icons: []pixmap{}, Title: s.name, Text: s.text}
}

func (s *sni) setIcon(icon []byte) error {
	px, err := toPixmap(icon)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.icons = []pixmap{px}
	err = s.publish(itemIface, "IconPixmap", s.icons, "NewIcon")
	s.mu.Unlock()
	return err
}

func (s *sni) setThemedIcon(themePath, name string) {
	s.mu.Lock()
	if s.themePath == themePath && s.iconName == name {
		s.mu.Unlock()
		return
	}
	s.themePath, s.iconName = themePath, name
	err := errors.Join(
		s.publish(itemIface, "IconThemePath", themePath, ""),
		s.publish(itemIface, "IconName", name, "NewIcon"),
	)
	s.mu.Unlock()
	s.report(err)
}

func (s *sni) setTooltip(text string) {
	s.mu.Lock()
	if s.text == text {
		s.mu.Unlock()
		return
	}
	s.text = text
	err := s.publish(itemIface, "ToolTip", s.tooltip(), "NewToolTip")
	s.mu.Unlock()
	s.report(err)
}

// publish sets an exported property and sends the item signal hosts listen for; the caller holds mu
// and reports the error after unlocking it, so an OnError may call back into the item.
func (s *sni) publish(iface, name string, value any, signal string) (err error) {
	if s.props == nil {
		return nil
	}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("tray: publish %s: %w", name, recoveredError(r))
		}
	}()
	s.props.SetMust(iface, name, value)
	if signal != "" {
		return s.conn.Emit(itemPath, itemIface+"."+signal)
	}
	return nil
}

// toPixmap converts a PNG to the item's ARGB32 network-order pixels.
func toPixmap(icon []byte) (pixmap, error) {
	img, err := png.Decode(bytes.NewReader(icon))
	if err != nil {
		return pixmap{}, fmt.Errorf("tray: icon is not a PNG: %w", err)
	}
	bounds := img.Bounds()
	rgba := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(rgba, rgba.Bounds(), img, bounds.Min, draw.Src)
	pixels := make([]byte, len(rgba.Pix))
	for i := 0; i < len(rgba.Pix); i += 4 {
		pixels[i] = rgba.Pix[i+3]
		pixels[i+1] = rgba.Pix[i]
		pixels[i+2] = rgba.Pix[i+1]
		pixels[i+3] = rgba.Pix[i+2]
	}
	return pixmap{Width: int32(bounds.Dx()), Height: int32(bounds.Dy()), Pixels: pixels}, nil
}
