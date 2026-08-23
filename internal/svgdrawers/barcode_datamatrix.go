package svgdrawers

import (
	"fmt"
	"strings"

	"github.com/swiftpush/zebrash/drawers"
	"github.com/swiftpush/zebrash/internal/barcodes/datamatrix"
	"github.com/swiftpush/zebrash/internal/barcodes/datamatrix/encoder"
	"github.com/swiftpush/zebrash/internal/elements"
	"github.com/swiftpush/zebrash/internal/svgwriter"
)

// NewBarcodeDatamatrixDrawer ports internal/pdfdrawers/barcode_datamatrix.go
// to SVG — vectorized via drawBitMatrix.
func NewBarcodeDatamatrixDrawer() *ElementDrawer {
	return &ElementDrawer{
		Draw: func(doc *svgwriter.Doc, element any, _ drawers.DrawerOptions, state *DrawerState) error {
			barcode, ok := element.(*elements.BarcodeDatamatrixWithData)
			if !ok {
				return nil
			}

			columns := max(barcode.Columns, 1)
			rows := max(barcode.Rows, 1)

			opts := encoder.Options{
				MinSize: encoder.NewDimension(columns, rows),
			}

			switch barcode.Ratio {
			case elements.DatamatrixRatioRectangular:
				opts.Shape = encoder.SymbolShapeHint_FORCE_RECTANGLE
			default:
				opts.Shape = encoder.SymbolShapeHint_FORCE_SQUARE
			}

			data := barcode.Data
			fnc1 := fmt.Sprintf("%c1", barcode.Escape)

			if strings.HasPrefix(data, fnc1) {
				opts.Gs1 = true
				data = strings.TrimPrefix(data, fnc1)
			}
			const GS = byte(29)
			data = strings.ReplaceAll(data, fnc1, string(GS))

			matrix, err := datamatrix.Encode(data, columns, rows, opts)
			if err != nil {
				return fmt.Errorf("failed to encode datamatrix barcode: %w", err)
			}

			scale := max(barcode.Height, 1)
			width := matrix.GetWidth() * scale
			height := matrix.GetHeight() * scale

			pos := adjustPositionFromBottom(barcode.Position, width, height, barcode.Orientation)

			x := float64(pos.X)
			y := float64(pos.Y)

			rotateDeg := orientationToDegreesCCW(barcode.Orientation)
			groups := 0
			if rotateDeg != 0 {
				doc.GroupTransform(transformRotate(rotateDeg, state.Dots(x), state.Dots(y)))
				groups++
				var dx, dy float64
				switch barcode.Orientation {
				case elements.FieldOrientation90:
					dy = -state.Dots(float64(height))
				case elements.FieldOrientation180:
					dx = -state.Dots(float64(width))
					dy = -state.Dots(float64(height))
				case elements.FieldOrientation270:
					dx = -state.Dots(float64(width))
				}
				if dx != 0 || dy != 0 {
					doc.GroupTransform(transformTranslate(dx, dy))
					groups++
				}
			}

			drawBitMatrix(doc, state, matrix, x, y, scale, scale)

			for i := 0; i < groups; i++ {
				doc.EndGroup()
			}
			return nil
		},
	}
}
