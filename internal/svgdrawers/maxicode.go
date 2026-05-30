package svgdrawers

import (
	"bytes"
	"fmt"
	"image/png"
	"math"

	"github.com/ingridhq/maxicode"
	"github.com/ingridhq/zebrash/drawers"
	"github.com/ingridhq/zebrash/internal/elements"
	"github.com/ingridhq/zebrash/internal/svgwriter"
)

// NewMaxicodeDrawer ports internal/pdfdrawers/maxicode.go to SVG. Maxicode
// is hexagonal — vectorising every hex would bloat the SVG without scan
// benefit, so the bitmap is embedded as a PNG <image>.
func NewMaxicodeDrawer() *ElementDrawer {
	return &ElementDrawer{
		Draw: func(doc *svgwriter.Doc, element any, options drawers.DrawerOptions, state *DrawerState) error {
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

			var buf bytes.Buffer
			if err := png.Encode(&buf, img); err != nil {
				return fmt.Errorf("failed to encode maxicode png: %w", err)
			}

			x := float64(barcode.Position.X - hexRectW)
			y := float64(barcode.Position.Y - hexRectH)
			w := float64(b.Dx())
			h := float64(b.Dy())

			doc.Image(state.Dots(x), state.Dots(y), state.Dots(w), state.Dots(h), buf.Bytes())
			return nil
		},
	}
}
