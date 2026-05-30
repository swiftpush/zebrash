package svgdrawers

// Mirrors internal/pdfdrawers/barcodes.go — wrap the bitmap drawers in
// constructors that match the PNG/PDF naming scheme so drawer.go can
// register them positionally.

func NewBarcode39Drawer() *ElementDrawer     { return bitmapBarcode39Drawer() }
func NewBarcodeEan13Drawer() *ElementDrawer  { return bitmapBarcodeEan13Drawer() }
func NewBarcode2of5Drawer() *ElementDrawer   { return bitmapBarcode2of5Drawer() }
func NewBarcodePdf417Drawer() *ElementDrawer { return bitmapBarcodePdf417Drawer() }
func NewBarcodeAztecDrawer() *ElementDrawer  { return bitmapBarcodeAztecDrawer() }
