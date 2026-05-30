package pdfdrawers

import (
	"github.com/go-pdf/fpdf"
	"github.com/ingridhq/zebrash/drawers"
)

// Remaining stubs for barcode drawers that still embed via the bitmap fallback
// (drawn through bitmap_barcode.go) until their encoders are refactored to
// expose patterns.

func NewBarcode39Drawer() *ElementDrawer     { return bitmapBarcode39Drawer() }
func NewBarcodeEan13Drawer() *ElementDrawer  { return bitmapBarcodeEan13Drawer() }
func NewBarcode2of5Drawer() *ElementDrawer   { return bitmapBarcode2of5Drawer() }
func NewBarcodePdf417Drawer() *ElementDrawer { return bitmapBarcodePdf417Drawer() }
func NewBarcodeAztecDrawer() *ElementDrawer  { return bitmapBarcodeAztecDrawer() }

func noopDrawer() *ElementDrawer {
	return &ElementDrawer{
		Draw: func(_ *fpdf.Fpdf, _ any, _ drawers.DrawerOptions, _ *DrawerState) error {
			return nil
		},
	}
}
