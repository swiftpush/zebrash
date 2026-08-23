package pdfdrawers

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-pdf/fpdf"

	"github.com/swiftpush/zebrash/drawers"
	"github.com/swiftpush/zebrash/internal/barcodes/code128"
	"github.com/swiftpush/zebrash/internal/elements"
)

var (
	barcode128FNC1 = string(code128.ESCAPE_FNC_1)

	parenthesisAndSpacesRegex = regexp.MustCompile(`[\(\)\s]`)
)

func NewBarcode128Drawer() *ElementDrawer {
	return &ElementDrawer{
		Draw: func(pdf *fpdf.Fpdf, element any, options drawers.DrawerOptions, state *DrawerState) error {
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
			if rotateDeg != 0 {
				pdf.TransformBegin()
				defer pdf.TransformEnd()
				// Mirror gg's rotateImage: translate is applied first to user
				// points, then rotation about (x, y). PDF's cm operator
				// post-multiplies the CTM (CTM' = CTM × M_new), so the
				// last-emitted cm runs first on user-space points — emit the
				// rotation first and the translation second.
				pdf.TransformRotate(rotateDeg, state.Dots(x), state.Dots(y))
				switch barcode.Orientation {
				case elements.FieldOrientation90:
					pdf.TransformTranslate(0, -state.Dots(float64(heightDots)))
				case elements.FieldOrientation180:
					pdf.TransformTranslate(-state.Dots(float64(widthDots)), -state.Dots(float64(heightDots)))
				case elements.FieldOrientation270:
					pdf.TransformTranslate(-state.Dots(float64(widthDots)), 0)
				}
			}

			drawBarPattern(pdf, state, pattern, x, y, barWidth, barHeight)

			if barcode.Line {
				drawBarcodeText(pdf, state, options, text, x, y, float64(barWidth), float64(widthDots), float64(heightDots), barcode.LineAbove)
			}

			return nil
		},
	}
}

// drawBarPattern emits one filled rect per run of true (ink) modules.
// Coalescing adjacent dark modules keeps the PDF small for dense barcodes.
func drawBarPattern(pdf *fpdf.Fpdf, state *DrawerState, pattern []bool, xDots, yDots float64, barWidthDots, heightDots int) {
	if len(pattern) == 0 {
		return
	}

	setFillColor(pdf, state, elements.LineColorBlack)

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
		pdf.Rect(state.Dots(rectX), state.Dots(yDots), state.Dots(rectW), state.Dots(h), "F")
	}
}

// drawBarcodeText renders the human-readable line under or above a 1D barcode.
// Mirrors applyLineTextToCtx in internal/drawers/barcode_128.go.
func drawBarcodeText(pdf *fpdf.Fpdf, state *DrawerState, options drawers.DrawerOptions, text string, xDots, yDots, barWidthDots, widthDots, heightDots float64, lineAbove bool) {
	fontSizeDots := max(barWidthDots, 1) * 10
	fontSizePt := FontSizePt(fontSizeDots, options.Dpmm)

	pdf.SetFont(FontDejavuSansMono, "", fontSizePt)
	setFillColor(pdf, state, elements.LineColorBlack)

	textWidthMm := pdf.GetStringWidth(text)

	cxMm := state.Dots(xDots + widthDots/2)
	textXMm := cxMm - textWidthMm/2

	var baselineYDots float64
	if lineAbove {
		baselineYDots = yDots - fontSizeDots/2
	} else {
		baselineYDots = yDots + heightDots + fontSizeDots
	}

	pdf.Text(textXMm, state.Dots(baselineYDots), text)
}

// modifyBarcodeContentEanMode mirrors the raster drawer.
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
