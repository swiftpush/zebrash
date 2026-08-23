//go:build svgraster

// Package svgraster rasterizes SVG bytes to a grayscale image for use by the
// SVG test harness. It is gated behind the `svgraster` build tag so callers
// that only need the SVG drawer don't transitively depend on the rasterizer.
//
// Implementation note: this wraps github.com/kanrichan/resvg-go, which is a
// pure-Go binding to resvg (Rust) compiled to WebAssembly and run under
// wazero — no cgo, no system libs. If a future binding switch is needed
// (e.g. native resvg cgo for performance), the only place to change is this
// file; the test harness only depends on the RasterizeSVG signature below.
package svgraster

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"

	resvg "github.com/kanrichan/resvg-go"

	"github.com/swiftpush/zebrash/internal/assets"
)

// RasterizeSVG renders svg at dpi and returns a grayscale image. The PNG
// harness writes grayscale (image.Gray) goldens, so we convert here to the
// same shape and let the diff helper assume *image.Gray.
//
// Font data is loaded from internal/assets so that @font-face rules in the
// SVG can be resolved to matching local data. Without this, resvg falls back
// to system fonts and the rendered glyphs drift wildly from the PNG goldens.
func RasterizeSVG(svg []byte, dpi float64) (*image.Gray, error) {
	ctx, err := resvg.NewContext(context.Background())
	if err != nil {
		return nil, fmt.Errorf("resvg: new context: %w", err)
	}
	defer func() { _ = ctx.Close() }()

	r, err := ctx.NewRenderer()
	if err != nil {
		return nil, fmt.Errorf("resvg: new renderer: %w", err)
	}
	defer func() { _ = r.Close() }()

	if err := r.SetDpi(float32(dpi)); err != nil {
		return nil, fmt.Errorf("resvg: set dpi: %w", err)
	}

	for _, fontData := range [][]byte{
		assets.FontHelveticaBold,
		assets.FontDejavuSansMono,
		assets.FontDejavuSansMonoBold,
		assets.FontZplGS,
	} {
		if err := r.LoadFontData(fontData); err != nil {
			return nil, fmt.Errorf("resvg: load font: %w", err)
		}
	}

	pngBytes, err := r.Render(svg)
	if err != nil {
		return nil, fmt.Errorf("resvg: render: %w", err)
	}

	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		return nil, fmt.Errorf("resvg: decode png: %w", err)
	}

	bounds := img.Bounds()
	gray := image.NewGray(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			lum := uint8((299*int(r>>8) + 587*int(g>>8) + 114*int(b>>8)) / 1000)
			gray.SetGray(x, y, color.Gray{Y: lum})
		}
	}
	return gray, nil
}
