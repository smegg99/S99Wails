// render_test.go
package main

import (
	"bytes"
	"context"
	"image/png"
	"testing"
)

const renderFixture = `<svg xmlns="http://www.w3.org/2000/svg" width="64" height="64"><rect x="8" y="8" width="48" height="48" style="fill:#f0b429"/></svg>`

func TestRendererPNG(t *testing.T) {
	r, err := NewRenderer(context.Background())
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}
	defer r.Close()

	data, err := r.PNG([]byte(renderFixture), 32)
	if err != nil {
		t.Fatalf("PNG: %v", err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if b := img.Bounds(); b.Dx() != 32 || b.Dy() != 32 {
		t.Errorf("bounds = %v, want 32x32", b)
	}

	// Determinism: a fresh Renderer instance must produce identical bytes.
	r2, err := NewRenderer(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer r2.Close()
	again, err := r2.PNG([]byte(renderFixture), 32)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, again) {
		t.Error("render is not deterministic across renderer instances")
	}
}

func TestRendererRejectsNonPositiveSize(t *testing.T) {
	r, err := NewRenderer(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if _, err := r.PNG([]byte(renderFixture), 0); err == nil {
		t.Error("size 0 must error")
	}
	if _, err := r.PNG([]byte(renderFixture), -3); err == nil {
		t.Error("negative size must error")
	}
}

func TestRendererMemoizes(t *testing.T) {
	r, err := NewRenderer(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	a, err := r.PNG([]byte(renderFixture), 32)
	if err != nil {
		t.Fatal(err)
	}
	b, err := r.PNG([]byte(renderFixture), 32)
	if err != nil {
		t.Fatal(err)
	}
	// Cached result is returned as a defensive copy: equal bytes, distinct backing array.
	if !bytes.Equal(a, b) {
		t.Error("cached render differs from original")
	}
	if len(a) > 0 && len(b) > 0 && &a[0] == &b[0] {
		t.Error("cache must return a defensive copy, not the cached slice itself")
	}
	// Mutating a caller's copy must not poison the cache.
	a[0] ^= 0xFF
	c, err := r.PNG([]byte(renderFixture), 32)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, c) {
		t.Error("cache was poisoned by caller mutation")
	}
}
