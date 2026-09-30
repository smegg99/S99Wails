// traygen/qt_test.go

package main

import (
	"bytes"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestFitSVGWithQtRenderer(t *testing.T) {
	compiler, err := exec.LookPath("c++")
	if err != nil {
		t.Skip("Qt regression needs a C++ compiler")
	}
	pkgConfig, err := exec.LookPath("pkg-config")
	if err != nil {
		t.Skip("Qt regression needs pkg-config")
	}
	flags, err := exec.Command(pkgConfig, "--cflags", "--libs", "Qt6Svg", "Qt6Gui").Output()
	if err != nil {
		t.Skip("Qt regression needs Qt6Svg and Qt6Gui development files")
	}
	dir := t.TempDir()
	binary := filepath.Join(dir, "qt-render")
	args := append([]string{"testdata/qt-render.cpp", "-o", binary}, strings.Fields(string(flags))...)
	if out, err := exec.Command(compiler, args...).CombinedOutput(); err != nil {
		t.Fatalf("compile Qt probe: %v\n%s", err, out)
	}
	for name, source := range map[string]string{
		"offset": `<svg xmlns="http://www.w3.org/2000/svg" viewBox="6 -26 301 301"><rect x="90" y="80" width="120" height="120" fill="#ff0000"/></svg>`,
		"wide":   `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 50"><rect width="100" height="50" fill="#ff0000"/></svg>`,
	} {
		t.Run(name, func(t *testing.T) {
			svg, err := FitSVG([]byte(source), 32, .125)
			if err != nil {
				t.Fatal(err)
			}
			input, output := filepath.Join(dir, name+".svg"), filepath.Join(dir, name+".png")
			if err := os.WriteFile(input, svg, 0600); err != nil {
				t.Fatal(err)
			}
			if out, err := exec.Command(binary, input, output).CombinedOutput(); err != nil {
				t.Fatalf("Qt render: %v\n%s", err, out)
			}
			data, err := os.ReadFile(output)
			if err != nil {
				t.Fatal(err)
			}
			img, err := png.Decode(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			_, _, _, center := img.At(16, 16).RGBA()
			if center == 0 {
				t.Fatal("Qt renders the fitted artwork as transparent")
			}
			_, _, _, corner := img.At(0, 0).RGBA()
			if corner != 0 {
				t.Fatal("Qt lost the transparent margin")
			}
		})
	}
}
