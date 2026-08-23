package pdfdrawers

import (
	"fmt"
	"strings"

	"github.com/go-pdf/fpdf"

	"github.com/swiftpush/zebrash/drawers"
	"github.com/swiftpush/zebrash/internal/barcodes/datamatrix"
	"github.com/swiftpush/zebrash/internal/barcodes/datamatrix/encoder"
	"github.com/swiftpush/zebrash/internal/elements"
)

func NewBarcodeDatamatrixDrawer() *ElementDrawer {
	return &ElementDrawer{
		Draw: func(pdf *fpdf.Fpdf, element any, _ drawers.DrawerOptions, state *DrawerState) error {
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
			if rotateDeg != 0 {
				pdf.TransformBegin()
				defer pdf.TransformEnd()
				// Mirror gg's rotateImage: rotate about (x, y), then translate so the
				// rotated symbol lands with its top-left at (x, y). PDF's cm
				// post-multiplies the CTM, so the last-emitted cm runs first on
				// user-space points — emit rotation first, translation second.
				pdf.TransformRotate(rotateDeg, state.Dots(x), state.Dots(y))
				switch barcode.Orientation {
				case elements.FieldOrientation90:
					pdf.TransformTranslate(0, -state.Dots(float64(height)))
				case elements.FieldOrientation180:
					pdf.TransformTranslate(-state.Dots(float64(width)), -state.Dots(float64(height)))
				case elements.FieldOrientation270:
					pdf.TransformTranslate(-state.Dots(float64(width)), 0)
				}
			}

			drawBitMatrix(pdf, state, matrix, x, y, scale, scale)
			return nil
		},
	}
}
