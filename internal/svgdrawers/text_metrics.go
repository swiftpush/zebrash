package svgdrawers

import (
	"sync"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"github.com/ingridhq/zebrash/internal/assets"
	"github.com/ingridhq/zebrash/internal/elements"
)

// faceKey identifies a font face by family, weight, and size in dots.
type faceKey struct {
	family string
	weight string
	size   float64
}

var (
	ttFont0  = mustParseTTF(assets.FontHelveticaBold)
	ttFont1  = mustParseTTF(assets.FontDejavuSansMono)
	ttFontB  = mustParseTTF(assets.FontDejavuSansMonoBold)
	ttFontGS = mustParseTTF(assets.FontZplGS)

	faceCacheMu sync.Mutex
	faceCache   = make(map[faceKey]font.Face)
)

func mustParseTTF(data []byte) *truetype.Font {
	f, err := truetype.Parse(data)
	if err != nil {
		panic("svgdrawers: failed to parse TTF: " + err.Error())
	}
	return f
}

// ttFontForSVGFamily returns the *truetype.Font corresponding to the SVG
// family+weight constants resolved by ResolveFontFamilyWeight.
func ttFontForSVGFamily(family, weight string) *truetype.Font {
	switch family {
	case FontHelvetica:
		return ttFont0
	case FontDejavuSansMono:
		if weight == weightBold {
			return ttFontB
		}
		return ttFont1
	case FontZplGS:
		return ttFontGS
	default:
		return ttFont1
	}
}

// getFace returns (possibly cached) font.Face for a given ZPL FontInfo at
// fontSizeDots. The face metrics are in fixed.Int26_6 font units; advances
// must be divided by 64 to get pixel values at the effective DPI.
func getFace(fi elements.FontInfo, fontSizeDots float64) font.Face {
	family, weight := ResolveFontFamilyWeight(fi)
	key := faceKey{family, weight, fontSizeDots}

	faceCacheMu.Lock()
	defer faceCacheMu.Unlock()

	if f, ok := faceCache[key]; ok {
		return f
	}
	ttf := ttFontForSVGFamily(family, weight)
	f := truetype.NewFace(ttf, &truetype.Options{Size: fontSizeDots})
	faceCache[key] = f
	return f
}

// MeasureStringDots measures the advance width of s in ZPL dots, using real
// glyph advances from the embedded TTF corresponding to fi at fontSizeDots.
// The result is in the same dot coordinate space as fontSizeDots itself.
func MeasureStringDots(fi elements.FontInfo, fontSizeDots float64, s string) float64 {
	face := getFace(fi, fontSizeDots)
	var total fixed.Int26_6
	for _, r := range s {
		adv, ok := face.GlyphAdvance(r)
		if !ok {
			// Fall back to a space advance for unmapped glyphs.
			adv, _ = face.GlyphAdvance(' ')
		}
		total += adv
	}
	// fixed.Int26_6 stores values as 1/64 pixel; convert to float pixels (= dots).
	return float64(total) / 64.0
}

// MeasureStringMm measures string width in millimeters, given the face size
// in dots and the dot-to-mm conversion factor.
func MeasureStringMm(fi elements.FontInfo, fontSizeDots, dotsToMm float64, s string) float64 {
	return MeasureStringDots(fi, fontSizeDots, s) * dotsToMm
}

// measureFn returns a closure that measures a string in millimeters using the
// font face resolved from fi at fontSizeDots. Convenient for passing to
// wrapWords / drawStringJustifiedSvg.
func measureFn(fi elements.FontInfo, fontSizeDots, dotsToMm float64) func(string) float64 {
	return func(s string) float64 {
		return MeasureStringMm(fi, fontSizeDots, dotsToMm, s)
	}
}
