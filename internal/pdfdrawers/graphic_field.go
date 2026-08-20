package pdfdrawers

import (
	"bytes"
	"fmt"
	"image"
	"image/png"

	"github.com/go-pdf/fpdf"

	"github.com/ingridhq/zebrash/drawers"
	"github.com/ingridhq/zebrash/internal/elements"
	"github.com/ingridhq/zebrash/internal/images"
)

func NewGraphicFieldDrawer() *ElementDrawer {
	return &ElementDrawer{
		Draw: func(pdf *fpdf.Fpdf, element any, _ drawers.DrawerOptions, state *DrawerState) error {
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

			// Magnification is applied during PDF placement instead of pre-scaling pixels:
			// fpdf will resample on render, but for ^GF the source bitmap is the truth and
			// we let the PDF viewer scale it sharply.
			mx := max(field.MagnificationX, 1)
			my := max(field.MagnificationY, 1)

			var buf bytes.Buffer
			if err := png.Encode(&buf, img); err != nil {
				return fmt.Errorf("failed to encode graphic field as png: %w", err)
			}

			imgName := fmt.Sprintf("gf_%p", field)
			pdf.RegisterImageOptionsReader(imgName, fpdf.ImageOptions{ImageType: "PNG"}, &buf)

			wMm := state.Dots(float64(width * mx))
			hMm := state.Dots(float64(height * my))
			xMm := state.Dots(float64(field.Position.X))
			yMm := state.Dots(float64(field.Position.Y))

			pdf.ImageOptions(imgName, xMm, yMm, wMm, hMm, false, fpdf.ImageOptions{ImageType: "PNG"}, 0, "")

			return nil
		},
	}
}
