// svg_test.go
package main

import (
	"strings"
	"testing"
)

const fixtureSVG = `<?xml version="1.0"?>
<svg xmlns="http://www.w3.org/2000/svg" xmlns:inkscape="http://www.inkscape.org/namespaces/inkscape" width="64" height="64">
  <rect inkscape:label="base" x="4" y="4" width="56" height="56" style="fill:#12181d;stroke:none"/>
  <circle inkscape:label="badge" cx="48" cy="16" r="10" style="fill:#000000;display:none"/>
  <path inkscape:label="decoration" d="M0 0h4v4H0z" fill="#ffffff"/>
</svg>`

func TestApplySlotsRecolorsAndShows(t *testing.T) {
	out, err := ApplySlots([]byte(fixtureSVG), []string{"badge"}, map[string]string{"badge": "#ff0000"})
	if err != nil {
		t.Fatalf("ApplySlots: %v", err)
	}
	s := string(out)
	if !strings.Contains(s, "fill:#ff0000") {
		t.Errorf("badge not recolored:\n%s", s)
	}
	if strings.Contains(s, "display:none") {
		t.Errorf("badge should be shown (display:none stripped):\n%s", s)
	}
	// untouched elements keep their fill
	if !strings.Contains(s, "fill:#12181d") {
		t.Errorf("base fill must be untouched:\n%s", s)
	}
}

func TestApplySlotsHidesUnassigned(t *testing.T) {
	out, err := ApplySlots([]byte(fixtureSVG), []string{"badge"}, nil)
	if err != nil {
		t.Fatalf("ApplySlots: %v", err)
	}
	if !strings.Contains(string(out), `display="none"`) {
		t.Errorf("unassigned badge slot should be hidden:\n%s", out)
	}
}

func TestApplySlotsFillAttribute(t *testing.T) {
	// slot element styled via fill attribute instead of style
	svg := `<svg xmlns="http://www.w3.org/2000/svg" xmlns:inkscape="http://www.inkscape.org/namespaces/inkscape"><rect inkscape:label="badge" fill="#000000"/></svg>`
	out, err := ApplySlots([]byte(svg), []string{"badge"}, map[string]string{"badge": "#00ff00"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `fill="#00ff00"`) {
		t.Errorf("fill attribute not recolored:\n%s", out)
	}
}

func TestApplySlotsMissingSlotErrors(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg"><rect/></svg>`
	_, err := ApplySlots([]byte(svg), []string{"badge"}, nil)
	if err == nil || !strings.Contains(err.Error(), "badge") {
		t.Fatalf("want missing-slot error naming the slot, got %v", err)
	}
}

func TestSetFillRemovesBareFillAttr(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" xmlns:inkscape="http://www.inkscape.org/namespaces/inkscape"><rect inkscape:label="badge" fill="#000000" style="display:none"/></svg>`
	out, err := ApplySlots([]byte(svg), []string{"badge"}, map[string]string{"badge": "#00ff00"})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if strings.Contains(s, `fill="#000000"`) {
		t.Errorf("stale bare fill attribute must be removed:\n%s", s)
	}
	if !strings.Contains(s, "fill:#00ff00") {
		t.Errorf("style fill not applied:\n%s", s)
	}
}

func TestColoredSlotInsideHiddenSlotErrors(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" xmlns:inkscape="http://www.inkscape.org/namespaces/inkscape"><g inkscape:label="outer"><circle inkscape:label="inner" style="fill:#000000"/></g></svg>`
	_, err := ApplySlots([]byte(svg), []string{"outer", "inner"}, map[string]string{"inner": "#ff0000"})
	if err == nil || !strings.Contains(err.Error(), "inner") || !strings.Contains(err.Error(), "outer") {
		t.Fatalf("want nested-hidden error naming both slots, got %v", err)
	}
}
