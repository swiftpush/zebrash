package pdfdrawers

import (
	"fmt"

	"github.com/go-pdf/fpdf"

	"github.com/ingridhq/zebrash/drawers"
	"github.com/ingridhq/zebrash/internal/barcodes/qrcode"
	"github.com/ingridhq/zebrash/internal/barcodes/qrcode/encoder"
	"github.com/ingridhq/zebrash/internal/elements"
)

func NewBarcodeQrDrawer() *ElementDrawer {
	return &ElementDrawer{
		Draw: func(pdf *fpdf.Fpdf, element any, _ drawers.DrawerOptions, state *DrawerState) error {
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
			modulesW := matrix.GetWidth()
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
			_ = modulesW

			drawBitMatrix(pdf, state, matrix, x, y, mag, mag)
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
