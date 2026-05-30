package svgdrawers

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"math"
	"regexp"
	"slices"
	"strings"

	"github.com/ingridhq/zebrash/drawers"
	"github.com/ingridhq/zebrash/internal/barcodes/aztec"
	"github.com/ingridhq/zebrash/internal/barcodes/code39"
	"github.com/ingridhq/zebrash/internal/barcodes/ean13"
	"github.com/ingridhq/zebrash/internal/barcodes/pdf417"
	"github.com/ingridhq/zebrash/internal/barcodes/twooffive"
	"github.com/ingridhq/zebrash/internal/elements"
	"github.com/ingridhq/zebrash/internal/svgwriter"
)

// embedBarcodeImage drops a rasterized barcode into the SVG as a PNG <image>.
// Mirrors pdfdrawers.embedBarcodeImage — same orientation-aware translate /
// rotate composition.
func embedBarcodeImage(doc *svgwriter.Doc, state *DrawerState, img image.Image, pos elements.LabelPosition, ori elements.FieldOrientation) (groups int, _ error) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width == 0 || height == 0 {
		return 0, nil
	}

	pos = adjustPositionFromBottom(pos, width, height, ori)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return 0, fmt.Errorf("failed to encode barcode bitmap as png: %w", err)
	}

	x := float64(pos.X)
	y := float64(pos.Y)
	xMm := state.Dots(x)
	yMm := state.Dots(y)
	wMm := state.Dots(float64(width))
	hMm := state.Dots(float64(height))

	rotateDeg := orientationToDegreesCCW(ori)
	if rotateDeg != 0 {
		doc.GroupTransform(transformRotate(rotateDeg, xMm, yMm))
		groups++
		var dx, dy float64
		switch ori {
		case elements.FieldOrientation90:
			dy = -hMm
		case elements.FieldOrientation180:
			dx = -wMm
			dy = -hMm
		case elements.FieldOrientation270:
			dx = -wMm
		}
		if dx != 0 || dy != 0 {
			doc.GroupTransform(transformTranslate(dx, dy))
			groups++
		}
	}

	doc.Image(xMm, yMm, wMm, hMm, buf.Bytes())

	for i := 0; i < groups; i++ {
		doc.EndGroup()
	}
	return 0, nil
}

func bitmapBarcode39Drawer() *ElementDrawer {
	return &ElementDrawer{
		Draw: func(doc *svgwriter.Doc, element any, options drawers.DrawerOptions, state *DrawerState) error {
			barcode, ok := element.(*elements.Barcode39WithData)
			if !ok {
				return nil
			}
			text := fmt.Sprintf("*%s*", barcode.Data)
			img, err := code39.Encode(barcode.Data, barcode.Width, barcode.Height, barcode.WidthRatio)
			if err != nil {
				return fmt.Errorf("failed to encode code39 barcode: %w", err)
			}
			if _, err := embedBarcodeImage(doc, state, img, barcode.Position, barcode.Orientation); err != nil {
				return err
			}
			if barcode.Line {
				bounds := img.Bounds()
				pos := adjustPositionFromBottom(barcode.Position, bounds.Dx(), bounds.Dy(), barcode.Orientation)
				drawBarcodeText(doc, state, options, text, float64(pos.X), float64(pos.Y), float64(barcode.Width), float64(bounds.Dx()), float64(bounds.Dy()), barcode.LineAbove)
			}
			return nil
		},
	}
}

var svgDigitsOnly = regexp.MustCompile(`[^0-9]+`)

func bitmapBarcode2of5Drawer() *ElementDrawer {
	return &ElementDrawer{
		Draw: func(doc *svgwriter.Doc, element any, options drawers.DrawerOptions, state *DrawerState) error {
			barcode, ok := element.(*elements.Barcode2of5WithData)
			if !ok {
				return nil
			}
			content := svgDigitsOnly.ReplaceAllString(barcode.Data, "")
			img, text, err := twooffive.EncodeInterleaved(content, barcode.Width, barcode.Height, barcode.WidthRatio, barcode.CheckDigit)
			if err != nil {
				return fmt.Errorf("failed to encode 2 of 5 barcode: %w", err)
			}
			if _, err := embedBarcodeImage(doc, state, img, barcode.Position, barcode.Orientation); err != nil {
				return err
			}
			if barcode.Line {
				bounds := img.Bounds()
				pos := adjustPositionFromBottom(barcode.Position, bounds.Dx(), bounds.Dy(), barcode.Orientation)
				drawBarcodeText(doc, state, options, text, float64(pos.X), float64(pos.Y), float64(barcode.Width), float64(bounds.Dx()), float64(bounds.Dy()), barcode.LineAbove)
			}
			return nil
		},
	}
}

func bitmapBarcodeEan13Drawer() *ElementDrawer {
	return &ElementDrawer{
		Draw: func(doc *svgwriter.Doc, element any, options drawers.DrawerOptions, state *DrawerState) error {
			barcode, ok := element.(*elements.BarcodeEan13WithData)
			if !ok {
				return nil
			}
			img, text, err := ean13.Encode(barcode.Data, barcode.Height, barcode.Width)
			if err != nil {
				return fmt.Errorf("failed to encode EAN-13 barcode: %w", err)
			}

			bounds := img.Bounds()
			width := bounds.Dx()
			height := bounds.Dy()
			if width == 0 || height == 0 {
				return nil
			}

			barWidth := max(barcode.Width, 1)
			guardExtension := ean13.CalculateGuardExtension(barWidth)

			pos := adjustPositionFromBottom(barcode.Position, width, height, barcode.Orientation)
			pos = adjustEan13Position(pos, barcode.Orientation, guardExtension)

			var buf bytes.Buffer
			if err := png.Encode(&buf, img); err != nil {
				return fmt.Errorf("failed to encode EAN-13 bitmap as png: %w", err)
			}

			x := float64(pos.X)
			y := float64(pos.Y)
			xMm := state.Dots(x)
			yMm := state.Dots(y)
			wMm := state.Dots(float64(width))
			hMm := state.Dots(float64(height))

			rotateDeg := orientationToDegreesCCW(barcode.Orientation)
			groups := 0
			if rotateDeg != 0 {
				doc.GroupTransform(transformRotate(rotateDeg, xMm, yMm))
				groups++
				var dx, dy float64
				switch barcode.Orientation {
				case elements.FieldOrientation90:
					dy = -hMm
				case elements.FieldOrientation180:
					dx = -wMm
					dy = -hMm
				case elements.FieldOrientation270:
					dx = -wMm
				}
				if dx != 0 || dy != 0 {
					doc.GroupTransform(transformTranslate(dx, dy))
					groups++
				}
			}

			doc.Image(xMm, yMm, wMm, hMm, buf.Bytes())

			if barcode.Line {
				drawEan13Text(doc, state, options, text, x, y, barcode.LineAbove, float64(width), float64(height), float64(barWidth), float64(guardExtension))
			}

			for i := 0; i < groups; i++ {
				doc.EndGroup()
			}
			return nil
		},
	}
}

func adjustEan13Position(pos elements.LabelPosition, ori elements.FieldOrientation, guardExtension int) elements.LabelPosition {
	if pos.CalculateFromBottom {
		return pos
	}
	x := pos.X
	y := pos.Y
	switch ori {
	case elements.FieldOrientation90:
		x -= guardExtension
	case elements.FieldOrientation180:
		y -= guardExtension
	}
	return elements.LabelPosition{X: x, Y: y}
}

// ean13FontInfo returns a synthetic FontInfo for EAN-13 human-readable text.
// EAN-13 always uses the Helvetica Bold face (font "0"), same as the PNG path.
func ean13FontInfo() elements.FontInfo {
	return elements.FontInfo{Name: "0"}
}

func drawEan13Text(doc *svgwriter.Doc, state *DrawerState, options drawers.DrawerOptions, text string, posX, posY float64, lineAbove bool, width, height, barWidth, guardExtension float64) {
	fontSizeDots := math.Round(width / 13)
	fontMm := fontSizeMm(fontSizeDots, state)
	fontPt := fontSizePt(fontSizeDots, options.Dpmm)
	fill := inkColor(state, elements.LineColorBlack)
	fi := ean13FontInfo()

	if len(text) == 13 && !lineAbove {
		formatted := wrapWithSpaces(fmt.Sprintf("%s||%s||%s", text[0:1], text[1:7], text[7:13]))
		x := posX - barWidth*10
		y := posY + height + fontSizeDots - guardExtension
		w := width + barWidth*5
		drawStringJustifiedSvg(doc, state, fi, fontSizeDots, fontMm, fontPt, fill, formatted, x, y, w, []string{"|"})
		return
	}

	formatted := wrapWithSpaces(text)
	x := posX + barWidth*8
	y := posY - guardExtension/2
	w := width - barWidth*16
	drawStringJustifiedSvg(doc, state, fi, fontSizeDots, fontMm, fontPt, fill, formatted, x, y, w, nil)
}

// drawStringJustifiedSvg mirrors pdfdrawers.drawStringJustifiedPdf, replacing
// the per-character estimate with real glyph advances from the embedded TTF.
func drawStringJustifiedSvg(doc *svgwriter.Doc, state *DrawerState, fi elements.FontInfo, fontSizeDots, fontMm, fontPt float64, fill, line string, xDots, yDots, maxWidthDots float64, hiddenWords []string) {
	words := strings.Fields(line)
	if len(words) == 0 {
		return
	}

	wordWidthsMm := make([]float64, len(words))
	totalWordWidthMm := 0.0
	for i, w := range words {
		wm := MeasureStringMm(fi, fontSizeDots, state.DotsToMm, w)
		wordWidthsMm[i] = wm
		totalWordWidthMm += wm
	}

	maxWidthMm := state.Dots(maxWidthDots)
	spaceCount := len(words) - 1
	spaceWidthMm := 0.0
	if spaceCount > 0 {
		spaceWidthMm = (maxWidthMm - totalWordWidthMm) / float64(spaceCount)
		if spaceWidthMm < 0 {
			// Fall back: use the advance of a space character.
			spaceWidthMm = MeasureStringMm(fi, fontSizeDots, state.DotsToMm, " ")
		}
	}

	yMm := state.Dots(yDots)
	cxMm := state.Dots(xDots)
	for i, word := range words {
		if !slices.Contains(hiddenWords, word) {
			doc.Text(cxMm, yMm, FontHelvetica, weightBold, fontPt, fontMm, fill, "start", word)
		}
		cxMm += wordWidthsMm[i] + spaceWidthMm
	}
}

func wrapWithSpaces(text string) string {
	var b strings.Builder
	for _, r := range text {
		b.WriteRune(' ')
		b.WriteRune(r)
		b.WriteRune(' ')
	}
	return b.String()
}

func bitmapBarcodePdf417Drawer() *ElementDrawer {
	return &ElementDrawer{
		Draw: func(doc *svgwriter.Doc, element any, _ drawers.DrawerOptions, state *DrawerState) error {
			barcode, ok := element.(*elements.BarcodePdf417WithData)
			if !ok {
				return nil
			}
			img, err := pdf417.Encode(barcode.Data, byte(barcode.Security), barcode.RowHeight, barcode.Columns)
			if err != nil {
				return fmt.Errorf("failed to encode pdf417 barcode: %w", err)
			}
			_, err = embedBarcodeImage(doc, state, img, barcode.Position, barcode.Orientation)
			return err
		},
	}
}

func bitmapBarcodeAztecDrawer() *ElementDrawer {
	return &ElementDrawer{
		Draw: func(doc *svgwriter.Doc, element any, _ drawers.DrawerOptions, state *DrawerState) error {
			barcode, ok := element.(*elements.BarcodeAztecWithData)
			if !ok {
				return nil
			}

			layers := aztec.DEFAULT_LAYERS
			minECCPercent := aztec.DEFAULT_EC_PERCENT
			const sizeModeFullRangeOffset = 200

			if barcode.Size > 0 {
				switch {
				case barcode.Size >= sizeModeFullRangeOffset && barcode.Size <= sizeModeFullRangeOffset+32:
					layers = barcode.Size - sizeModeFullRangeOffset
				case barcode.Size >= 1 && barcode.Size <= 99:
					minECCPercent = barcode.Size
				case barcode.Size >= 101 && barcode.Size <= 104:
					layers = -(barcode.Size - 100)
				default:
					return fmt.Errorf("aztec barcode size/mode %d is not supported", barcode.Size)
				}
			}

			img, err := aztec.Encode([]byte(barcode.Data), minECCPercent, layers, barcode.Magnification)
			if err != nil {
				return fmt.Errorf("failed to encode aztec barcode: %w", err)
			}
			_, err = embedBarcodeImage(doc, state, img, barcode.Position, barcode.Orientation)
			return err
		},
	}
}
