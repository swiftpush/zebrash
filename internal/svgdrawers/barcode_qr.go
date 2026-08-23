package svgdrawers

import (
	"fmt"

	"github.com/swiftpush/zebrash/drawers"
	"github.com/swiftpush/zebrash/internal/barcodes/qrcode"
	"github.com/swiftpush/zebrash/internal/barcodes/qrcode/encoder"
	"github.com/swiftpush/zebrash/internal/elements"
	"github.com/swiftpush/zebrash/internal/svgwriter"
)

// NewBarcodeQrDrawer ports internal/pdfdrawers/barcode_qr.go to SVG —
// vectorized via drawBitMatrix.
func NewBarcodeQrDrawer() *ElementDrawer {
	return &ElementDrawer{
		Draw: func(doc *svgwriter.Doc, element any, _ drawers.DrawerOptions, state *DrawerState) error {
			barcode, ok := element.(*elements.BarcodeQrWithData)
			if !ok {
				return nil
			}

			inputData, ec, _, err := barcode.GetInputData()
			if err != nil {
				return err
			}

			matrix, err := qrcode.Encode(inputData, 1, 1, mapQrErrorCorrectionLevel(ec), encoder.Options{
				QuietZone: 0,
			})
			if err != nil {
				return fmt.Errorf("failed to encode qr barcode: %w", err)
			}

			mag := max(barcode.Magnification, 1)
			modulesH := matrix.GetHeight()

			pos := barcode.Position
			x := float64(pos.X)
			y := float64(pos.Y)
			if !pos.CalculateFromBottom {
				y += float64(barcode.Height)
			} else {
				ftOffset := mag * 7
				y = max(y-float64(modulesH*mag), 0) - float64(ftOffset)
			}

			drawBitMatrix(doc, state, matrix, x, y, mag, mag)
			return nil
		},
	}
}

func mapQrErrorCorrectionLevel(ec elements.QrErrorCorrectionLevel) encoder.ErrorCorrectionLevel {
	switch ec {
	case elements.QrErrorCorrectionL:
		return encoder.ErrorCorrectionLevel_L
	case elements.QrErrorCorrectionQ:
		return encoder.ErrorCorrectionLevel_Q
	case elements.QrErrorCorrectionH:
		return encoder.ErrorCorrectionLevel_H
	default:
		return encoder.ErrorCorrectionLevel_M
	}
}
