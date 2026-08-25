// traygen/svg.go

package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/beevik/etree"
)

var (
	fillRe    = regexp.MustCompile(`fill:\s*[^;]+`)
	displayRe = regexp.MustCompile(`display:\s*[^;]+;?`)
)

// ApplySlots recolors and shows each declared inkscape:label slot with a color,
// hides the rest, and errors on declared slots missing from the document.
func ApplySlots(svgData []byte, declared []string, colors map[string]string) ([]byte, error) {
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(svgData); err != nil {
		return nil, fmt.Errorf("parse svg: %w", err)
	}

	seen := map[string]bool{}
	for _, el := range allElements(&doc.Element) {
		label := el.SelectAttrValue("inkscape:label", "")
		if label == "" {
			continue
		}
		for _, slot := range declared {
			if label != slot {
				continue
			}
			seen[slot] = true
			if color, ok := colors[slot]; ok {
				setFill(el, color)
				show(el)
			} else {
				hide(el)
			}
		}
	}

	for _, slot := range declared {
		if !seen[slot] {
			return nil, fmt.Errorf("declared slot %q not found in svg (no element with inkscape:label=%q)", slot, slot)
		}
	}

	// A colored slot nested inside a hidden slot would never render: error.
	hidden := map[string]bool{}
	for _, slot := range declared {
		if _, ok := colors[slot]; !ok {
			hidden[slot] = true
		}
	}
	for _, el := range allElements(&doc.Element) {
		label := el.SelectAttrValue("inkscape:label", "")
		if _, ok := colors[label]; !ok {
			continue
		}
		for parent := el.Parent(); parent != nil; parent = parent.Parent() {
			if plabel := parent.SelectAttrValue("inkscape:label", ""); hidden[plabel] {
				return nil, fmt.Errorf("colored slot %q is nested inside hidden slot %q and would never render", label, plabel)
			}
		}
	}

	return doc.WriteToBytes()
}

// allElements flattens the element tree depth-first.
func allElements(root *etree.Element) []*etree.Element {
	out := []*etree.Element{root}
	for _, child := range root.ChildElements() {
		out = append(out, allElements(child)...)
	}
	return out
}

func setFill(el *etree.Element, color string) {
	if style := el.SelectAttrValue("style", ""); style != "" {
		if fillRe.MatchString(style) {
			style = fillRe.ReplaceAllString(style, "fill:"+color)
		} else {
			style = strings.TrimSuffix(style, ";") + ";fill:" + color
		}
		el.CreateAttr("style", style)
		el.RemoveAttr("fill")
		return
	}
	el.CreateAttr("fill", color)
}

func show(el *etree.Element) {
	el.RemoveAttr("display")
	if style := el.SelectAttrValue("style", ""); style != "" {
		style = strings.Trim(displayRe.ReplaceAllString(style, ""), ";")
		el.CreateAttr("style", style)
	}
}

func hide(el *etree.Element) {
	el.CreateAttr("display", "none")
}
