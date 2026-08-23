package svgdrawers

import (
	"github.com/swiftpush/zebrash/internal/assets"
	"github.com/swiftpush/zebrash/internal/elements"
	"github.com/swiftpush/zebrash/internal/svgwriter"
)

// The SVG <text> font-family values match the real TTF family names rather
// than the abstract "ZebraFontX" aliases the PDF backend uses, because resvg
// (and most other Go SVG rasterizers we've evaluated) resolve fonts via the
// system font DB / loaded font data — keyed on the TTF's own family — rather
// than via inline @font-face aliases. Using the real names lets the same
// SVG render in:
//   - resvg with our four embedded TTFs registered via LoadFontData;
//   - a browser, where the inline @font-face <defs> entry kicks in.
//
// HelveticaBoldCondensedCustom.ttf has a non-standard family value ("My
// Font" / "Bold" subfamily) — we use it as-is to avoid republishing the
// TTF. Down side: any consumer of these SVGs has to resolve "My Font" too.
const (
	FontHelvetica          = "My Font"
	FontDejavuSansMono     = "DejaVu Sans Mono"
	FontDejavuSansMonoBold = "DejaVu Sans Mono"
	FontZplGS              = "ZPL GS"

	// Font weights mirror the TTF subfamily so an SVG renderer can pick the
	// right face when multiple weights share a family name (DejaVu Sans
	// Mono has both normal and bold weights registered by RegisterBuiltInFonts).
	weightNormal = "normal"
	weightBold   = "bold"
)

// RegisterBuiltInFonts embeds the four built-in Zebra TTFs as @font-face
// rules in the SVG's <defs> block. The rules use the same family+weight
// pair the <text> elements reference, so a browser viewer resolves them
// from the inline data URL. resvg-go's font DB doesn't read @font-face;
// the SVG rasterizer's RasterizeSVG explicitly LoadFontDatas the same
// bytes so the names match either way.
func RegisterBuiltInFonts(doc *svgwriter.Doc) {
	doc.FontFace(FontHelvetica, weightBold, "font/ttf", assets.FontHelveticaBold)
	doc.FontFace(FontDejavuSansMono, weightNormal, "font/ttf", assets.FontDejavuSansMono)
	doc.FontFace(FontDejavuSansMonoBold, weightBold, "font/ttf", assets.FontDejavuSansMonoBold)
	doc.FontFace(FontZplGS, weightNormal, "font/ttf", assets.FontZplGS)
}

// ResolveFontFamily maps a ZPL font name to the registered SVG family name.
// Custom (^DU) fonts are not yet embedded — same v1 caveat as PDF.
func ResolveFontFamily(font elements.FontInfo) string {
	family, _ := ResolveFontFamilyWeight(font)
	return family
}

// ResolveFontFamilyWeight returns the SVG font-family and font-weight for a
// ZPL font. The PDF backend works around this by registering the bold TTF
// under a separate family name ("ZebraFontB"); SVG uses the standard CSS
// family+weight pair so the @font-face rule and the <text> element line up.
func ResolveFontFamilyWeight(font elements.FontInfo) (family, weight string) {
	if font.IsCustomFont() {
		return FontDejavuSansMono, weightNormal
	}

	switch font.Name {
	case "0":
		// HelveticaBoldCondensedCustom is only shipped in Bold.
		return FontHelvetica, weightBold
	case "B":
		return FontDejavuSansMonoBold, weightBold
	case "GS":
		return FontZplGS, weightNormal
	default:
		return FontDejavuSansMono, weightNormal
	}
}
