// traygen/render.go

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"

	resvg "github.com/kanrichan/resvg-go"
)

// Renderer rasterizes SVGs with resvg in wasm, byte-identical across machines,
// and memoizes per (svg, size). It is not safe for concurrent use.
type Renderer struct {
	ctx   *resvg.Context
	r     *resvg.Renderer
	cache map[string][]byte
}

func NewRenderer(ctx context.Context) (*Renderer, error) {
	rctx, err := resvg.NewContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("resvg context: %w", err)
	}
	renderer, err := rctx.NewRenderer()
	if err != nil {
		_ = rctx.Close()
		return nil, fmt.Errorf("resvg renderer: %w", err)
	}
	return &Renderer{ctx: rctx, r: renderer, cache: map[string][]byte{}}, nil
}

// PNG renders the SVG to a size x size PNG.
func (r *Renderer) PNG(svg []byte, size int) ([]byte, error) {
	if size <= 0 {
		return nil, fmt.Errorf("render: size must be positive, got %d", size)
	}
	key := fmt.Sprintf("%d:%x", size, sha256.Sum256(svg))
	if data, ok := r.cache[key]; ok {
		return bytes.Clone(data), nil
	}
	data, err := r.r.RenderWithSize(svg, uint32(size), uint32(size))
	if err != nil {
		return nil, fmt.Errorf("render %dpx: %w", size, err)
	}
	r.cache[key] = data
	return bytes.Clone(data), nil
}

func (r *Renderer) Close() error {
	return errors.Join(r.r.Close(), r.ctx.Close())
}
