package pdfdrawers

import (
	"bytes"
	"fmt"
	"image/png"
	"math"

	"github.com/go-pdf/fpdf"
	"github.com/ingridhq/maxicode"

	"github.com/swiftpush/zebrash/drawers"
	"github.com/swiftpush/zebrash/internal/elements"
)

func NewMaxicodeDrawer() *ElementDrawer {
	return &ElementDrawer{
		Draw: func(pdf *fpdf.Fpdf, element any, options drawers.DrawerOptions, state *DrawerState) error {
			barcode, ok := element.(*elements.MaxicodeWithData)
			if !ok {
				return nil
			}

			inputData, err := barcode.GetInputData()
			if err != nil {
				return err
			}

			grid, err := maxicode.Encode(barcode.Code.Mode, 0, inputData)
			if err != nil {
				return fmt.Errorf("failed to encode maxicode grid: %w", err)
			}

			dpmm := float64(options.Dpmm)
			hexRectW := int(math.Round(0.76 * dpmm))
			hexRectH := int(math.Round(0.88 * dpmm))

			img := grid.Draw(dpmm).Image()
			b := img.Bounds()

			// Maxicode is hexagonal; vector emission of every hex would bloat the
			// PDF without scan benefit. Embed as a 1-bit PNG and let the rasterizer
			// scale it. This is the "maxicode stays bitmap-embedded" plan note.
			var buf bytes.Buffer
			if err := png.Encode(&buf, img); err != nil {
				return fmt.Errorf("failed to encode maxicode png: %w", err)
			}

			imgName := fmt.Sprintf("mx_%p", barcode)
			pdf.RegisterImageOptionsReader(imgName, fpdf.ImageOptions{ImageType: "PNG"}, &buf)

			x := float64(barcode.Position.X - hexRectW)
			y := float64(barcode.Position.Y - hexRectH)
			w := float64(b.Dx())
			h := float64(b.Dy())

			pdf.ImageOptions(imgName,
				state.Dots(x), state.Dots(y),
				state.Dots(w), state.Dots(h),
				false, fpdf.ImageOptions{ImageType: "PNG"}, 0, "")

			return nil
		},
	}
}
