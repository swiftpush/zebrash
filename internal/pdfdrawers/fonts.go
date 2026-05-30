package pdfdrawers

import (
	"github.com/go-pdf/fpdf"
	"github.com/ingridhq/zebrash/internal/assets"
	"github.com/ingridhq/zebrash/internal/elements"
)

const (
	FontHelvetica          = "ZebraFont0"
	FontDejavuSansMono     = "ZebraFont1"
	FontDejavuSansMonoBold = "ZebraFontB"
	FontZplGS              = "ZebraFontGS"
)

// RegisterBuiltInFonts embeds the same TTFs the raster pipeline uses for
// Zebra's built-in fonts (0, A-H, GS) into the PDF document. Call once per document.
func RegisterBuiltInFonts(pdf *fpdf.Fpdf) {
	pdf.AddUTF8FontFromBytes(FontHelvetica, "", assets.FontHelveticaBold)
	pdf.AddUTF8FontFromBytes(FontDejavuSansMono, "", assets.FontDejavuSansMono)
	pdf.AddUTF8FontFromBytes(FontDejavuSansMonoBold, "", assets.FontDejavuSansMonoBold)
	pdf.AddUTF8FontFromBytes(FontZplGS, "", assets.FontZplGS)
}

// ResolveFontFamily maps a ZPL font name to the registered fpdf family name.
// Custom (^DU-uploaded) fonts are not yet embedded in PDF and fall back to the default.
func ResolveFontFamily(font elements.FontInfo) string {
	if font.IsCustomFont() {
		return FontDejavuSansMono
	}

	switch font.Name {
	case "0":
		return FontHelvetica
	case "B":
		return FontDejavuSansMonoBold
	case "GS":
		return FontZplGS
	default:
		return FontDejavuSansMono
	}
}

// FontSizePt converts a ZPL font size measured in dots to PDF points.
// 1 inch = 25.4 mm = 72 pt; dpmm dots per mm => pt = dots * 72 / (dpmm * 25.4).
func FontSizePt(dots float64, dpmm int) float64 {
	if dpmm == 0 {
		return dots
	}
	return dots * 72.0 / (float64(dpmm) * 25.4)
}
