package svgdrawers

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ingridhq/zebrash/drawers"
	"github.com/ingridhq/zebrash/internal/barcodes/code128"
	"github.com/ingridhq/zebrash/internal/elements"
	"github.com/ingridhq/zebrash/internal/svgwriter"
)

var (
	barcode128FNC1 = string(code128.ESCAPE_FNC_1)

	parenthesisAndSpacesRegex = regexp.MustCompile(`[\(\)\s]`)
)

// NewBarcode128Drawer ports internal/pdfdrawers/barcode_128.go to SVG.
// Vectorized: one filled rect per run of dark bars, plus a human-readable
// text line if Line is set.
func NewBarcode128Drawer() *ElementDrawer {
	return &ElementDrawer{
		Draw: func(doc *svgwriter.Doc, element any, options drawers.DrawerOptions, state *DrawerState) error {
			barcode, ok := element.(*elements.Barcode128WithData)
			if !ok {
				return nil
			}

			content := barcode.Data
			text := barcode.Data

			switch barcode.Mode {
			case elements.BarcodeModeEan:
				content, text = modifyBarcodeContentEanMode(content)
			case elements.BarcodeModeUcc:
				content = modifyBarcodeContentUccMode(content)
			case elements.BarcodeModeAutomatic:
			}

			var (
				pattern []bool
				err     error
			)

			switch barcode.Mode {
			case elements.BarcodeModeNo:
				pattern, text, err = code128.EncodePatternNoMode(content)
			default:
				pattern, err = code128.EncodePatternAuto(content)
			}
			if err != nil {
				return fmt.Errorf("failed to encode barcode: %w", err)
			}

			barWidth := max(barcode.Width, 1)
			barHeight := max(barcode.Height, 1)

			widthDots := len(pattern) * barWidth
			heightDots := barHeight

			pos := adjustPositionFromBottom(barcode.Position, widthDots, heightDots, barcode.Orientation)

			x := float64(pos.X)
			y := float64(pos.Y)

			rotateDeg := orientationToDegreesCCW(barcode.Orientation)
			groups := 0
			if rotateDeg != 0 {
				// SVG composes transforms left-to-right (the leftmost transform
				// is applied last to user points), so to match the PDF path's
				// "translate then rotate around (x,y)" we put rotate before
				// translate in the SVG transform string.
				doc.GroupTransform(transformRotate(rotateDeg, state.Dots(x), state.Dots(y)))
				groups++
				var dx, dy float64
				switch barcode.Orientation {
				case elements.FieldOrientation90:
					dy = -state.Dots(float64(heightDots))
				case elements.FieldOrientation180:
					dx = -state.Dots(float64(widthDots))
					dy = -state.Dots(float64(heightDots))
				case elements.FieldOrientation270:
					dx = -state.Dots(float64(widthDots))
				}
				if dx != 0 || dy != 0 {
					doc.GroupTransform(transformTranslate(dx, dy))
					groups++
				}
			}

			drawBarPattern(doc, state, pattern, x, y, barWidth, barHeight)

			if barcode.Line {
				drawBarcodeText(doc, state, options, text, x, y, float64(barWidth), float64(widthDots), float64(heightDots), barcode.LineAbove)
			}

			for i := 0; i < groups; i++ {
				doc.EndGroup()
			}
			return nil
		},
	}
}

// drawBarPattern mirrors pdfdrawers.drawBarPattern: one filled rect per run
// of true (ink) modules.
func drawBarPattern(doc *svgwriter.Doc, state *DrawerState, pattern []bool, xDots, yDots float64, barWidthDots, heightDots int) {
	if len(pattern) == 0 {
		return
	}

	fill := inkColor(state, elements.LineColorBlack)

	bw := float64(barWidthDots)
	h := float64(heightDots)

	i := 0
	for i < len(pattern) {
		if !pattern[i] {
			i++
			continue
		}
		runStart := i
		for i < len(pattern) && pattern[i] {
			i++
		}
		runLen := i - runStart

		rectX := xDots + float64(runStart)*bw
		rectW := float64(runLen) * bw
		doc.Rect(state.Dots(rectX), state.Dots(yDots), state.Dots(rectW), state.Dots(h), fill, "", 0)
	}
}

// drawBarcodeText mirrors pdfdrawers.drawBarcodeText: human-readable line
// under (or above) a 1D barcode, centered horizontally on the barcode.
func drawBarcodeText(doc *svgwriter.Doc, state *DrawerState, options drawers.DrawerOptions, text string, xDots, yDots, barWidthDots, widthDots, heightDots float64, lineAbove bool) {
	fontSizeDots := max(barWidthDots, 1) * 10
	fontMm := fontSizeMm(fontSizeDots, state)
	fontPt := fontSizePt(fontSizeDots, options.Dpmm)

	cxMm := state.Dots(xDots + widthDots/2)

	var baselineYDots float64
	if lineAbove {
		baselineYDots = yDots - fontSizeDots/2
	} else {
		baselineYDots = yDots + heightDots + fontSizeDots
	}

	fill := inkColor(state, elements.LineColorBlack)
	doc.Text(cxMm, state.Dots(baselineYDots), FontDejavuSansMono, weightNormal, fontPt, fontMm, fill, "middle", text)
}

// modifyBarcodeContentEanMode / modifyBarcodeContentUccMode mirror pdfdrawers.

func modifyBarcodeContentEanMode(content string) (string, string) {
	text := strings.ReplaceAll(content, ">8", "")

	content = parenthesisAndSpacesRegex.ReplaceAllString(content, "")
	content = strings.ReplaceAll(content, ">8", barcode128FNC1)
	if !strings.HasPrefix(content, barcode128FNC1) {
		content = barcode128FNC1 + content
	}
	return content, text
}

func modifyBarcodeContentUccMode(content string) string {
	content = addZerosPrefix(content)
	content = content[:19]
	checksumDigit := calculateUccBarcodeChecksumDigit(content)
	return barcode128FNC1 + content + strconv.Itoa(checksumDigit)
}

func addZerosPrefix(in string) string {
	prefixLen := 19 - len(in)
	var b strings.Builder
	for range prefixLen {
		b.WriteRune('0')
	}
	b.WriteString(in)
	return b.String()
}

func calculateUccBarcodeChecksumDigit(content string) int {
	checksum := 0
	for i := range 19 {
		checksum += int(content[i]-48) * (i%2*2 + 7)
	}
	return checksum % 10
}
