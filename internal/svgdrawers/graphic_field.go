package svgdrawers

import (
	"bytes"
	"fmt"
	"image"
	"image/png"

	"github.com/ingridhq/zebrash/drawers"
	"github.com/ingridhq/zebrash/internal/elements"
	"github.com/ingridhq/zebrash/internal/images"
	"github.com/ingridhq/zebrash/internal/svgwriter"
)

// NewGraphicFieldDrawer ports internal/pdfdrawers/graphic_field.go to SVG.
// The bitmap is encoded as PNG and embedded as a data: URL inside <image>.
func NewGraphicFieldDrawer() *ElementDrawer {
	return &ElementDrawer{
		Draw: func(doc *svgwriter.Doc, element any, _ drawers.DrawerOptions, state *DrawerState) error {
			field, ok := element.(*elements.GraphicField)
			if !ok {
				return nil
			}

			dataLen := len(field.Data)
			if field.TotalBytes > 0 {
				dataLen = min(field.TotalBytes, dataLen)
			}

			width := field.RowBytes * 8
			if field.RowBytes == 0 {
				return nil
			}
			height := dataLen / field.RowBytes

			img := image.NewRGBA(image.Rect(0, 0, width, height))
			for y := range height {
				for x := range width {
					idx := y*field.RowBytes + x/8
					if idx >= len(field.Data) {
						continue
					}
					val := (field.Data[idx] >> (7 - x%8)) & 1
					if val != 0 {
						img.SetRGBA(x, y, images.ColorBlack)
					}
				}
			}

			mx := max(field.MagnificationX, 1)
			my := max(field.MagnificationY, 1)

			var buf bytes.Buffer
			if err := png.Encode(&buf, img); err != nil {
				return fmt.Errorf("failed to encode graphic field as png: %w", err)
			}

			wMm := state.Dots(float64(width * mx))
			hMm := state.Dots(float64(height * my))
			xMm := state.Dots(float64(field.Position.X))
			yMm := state.Dots(float64(field.Position.Y))

			doc.Image(xMm, yMm, wMm, hMm, buf.Bytes())
			return nil
		},
	}
}
