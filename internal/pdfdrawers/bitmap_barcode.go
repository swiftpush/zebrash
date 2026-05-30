package pdfdrawers

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"math"
	"regexp"
	"slices"
	"strings"

	"github.com/go-pdf/fpdf"
	"github.com/ingridhq/zebrash/drawers"
	"github.com/ingridhq/zebrash/internal/barcodes/aztec"
	"github.com/ingridhq/zebrash/internal/barcodes/code39"
	"github.com/ingridhq/zebrash/internal/barcodes/ean13"
	"github.com/ingridhq/zebrash/internal/barcodes/pdf417"
	"github.com/ingridhq/zebrash/internal/barcodes/twooffive"
	"github.com/ingridhq/zebrash/internal/elements"
)

// embedBarcodeImage drops a rasterized barcode into the PDF as an embedded PNG.
// This is the v1 fallback for 1D and 2D barcodes whose encoders haven't yet
// been refactored to expose bar/module patterns. It produces correct output but
// loses the scan-quality benefit of vector emission at zoom — vectorizing each
// of these is a follow-up driven by EncodePattern* / module-grid extraction.
func embedBarcodeImage(pdf *fpdf.Fpdf, state *DrawerState, img image.Image, pos elements.LabelPosition, ori elements.FieldOrientation, name string) error {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width == 0 || height == 0 {
		return nil
	}

	pos = adjustPositionFromBottom(pos, width, height, ori)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return fmt.Errorf("failed to encode barcode bitmap as png: %w", err)
	}
	pdf.RegisterImageOptionsReader(name, fpdf.ImageOptions{ImageType: "PNG"}, &buf)

	x := float64(pos.X)
	y := float64(pos.Y)

	rotateDeg := orientationToDegreesCCW(ori)
	if rotateDeg != 0 {
		pdf.TransformBegin()
		defer pdf.TransformEnd()
		// Mirror gg's rotateImage: rotate about (x, y), then translate so the
		// rotated image lands with its top-left at (x, y). PDF's cm
		// post-multiplies the CTM, so the last-emitted cm runs first on
		// user-space points — emit rotation first, translation second.
		pdf.TransformRotate(rotateDeg, state.Dots(x), state.Dots(y))
		switch ori {
		case elements.FieldOrientation90:
			pdf.TransformTranslate(0, -state.Dots(float64(height)))
		case elements.FieldOrientation180:
			pdf.TransformTranslate(-state.Dots(float64(width)), -state.Dots(float64(height)))
		case elements.FieldOrientation270:
			pdf.TransformTranslate(-state.Dots(float64(width)), 0)
		}
	}

	pdf.ImageOptions(name,
		state.Dots(x), state.Dots(y),
		state.Dots(float64(width)), state.Dots(float64(height)),
		false, fpdf.ImageOptions{ImageType: "PNG"}, 0, "")
	return nil
}

func bitmapBarcode39Drawer() *ElementDrawer {
	return &ElementDrawer{
		Draw: func(pdf *fpdf.Fpdf, element any, options drawers.DrawerOptions, state *DrawerState) error {
			barcode, ok := element.(*elements.Barcode39WithData)
			if !ok {
				return nil
			}
			text := fmt.Sprintf("*%s*", barcode.Data)
			img, err := code39.Encode(barcode.Data, barcode.Width, barcode.Height, barcode.WidthRatio)
			if err != nil {
				return fmt.Errorf("failed to encode code39 barcode: %w", err)
			}
			if err := embedBarcodeImage(pdf, state, img, barcode.Position, barcode.Orientation, fmt.Sprintf("c39_%p", barcode)); err != nil {
				return err
			}
			if barcode.Line {
				bounds := img.Bounds()
				pos := adjustPositionFromBottom(barcode.Position, bounds.Dx(), bounds.Dy(), barcode.Orientation)
				drawBarcodeText(pdf, state, options, text, float64(pos.X), float64(pos.Y), float64(barcode.Width), float64(bounds.Dx()), float64(bounds.Dy()), barcode.LineAbove)
			}
			return nil
		},
	}
}

var pdfDigitsOnly = regexp.MustCompile(`[^0-9]+`)

func bitmapBarcode2of5Drawer() *ElementDrawer {
	return &ElementDrawer{
		Draw: func(pdf *fpdf.Fpdf, element any, options drawers.DrawerOptions, state *DrawerState) error {
			barcode, ok := element.(*elements.Barcode2of5WithData)
			if !ok {
				return nil
			}
			content := pdfDigitsOnly.ReplaceAllString(barcode.Data, "")
			img, text, err := twooffive.EncodeInterleaved(content, barcode.Width, barcode.Height, barcode.WidthRatio, barcode.CheckDigit)
			if err != nil {
				return fmt.Errorf("failed to encode 2 of 5 barcode: %w", err)
			}
			if err := embedBarcodeImage(pdf, state, img, barcode.Position, barcode.Orientation, fmt.Sprintf("c25_%p", barcode)); err != nil {
				return err
			}
			if barcode.Line {
				bounds := img.Bounds()
				pos := adjustPositionFromBottom(barcode.Position, bounds.Dx(), bounds.Dy(), barcode.Orientation)
				drawBarcodeText(pdf, state, options, text, float64(pos.X), float64(pos.Y), float64(barcode.Width), float64(bounds.Dx()), float64(bounds.Dy()), barcode.LineAbove)
			}
			return nil
		},
	}
}

func bitmapBarcodeEan13Drawer() *ElementDrawer {
	return &ElementDrawer{
		Draw: func(pdf *fpdf.Fpdf, element any, options drawers.DrawerOptions, state *DrawerState) error {
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
			imgName := fmt.Sprintf("ean_%p", barcode)
			pdf.RegisterImageOptionsReader(imgName, fpdf.ImageOptions{ImageType: "PNG"}, &buf)

			x := float64(pos.X)
			y := float64(pos.Y)
			xMm := state.Dots(x)
			yMm := state.Dots(y)
			wMm := state.Dots(float64(width))
			hMm := state.Dots(float64(height))

			rotateDeg := orientationToDegreesCCW(barcode.Orientation)
			if rotateDeg != 0 {
				pdf.TransformBegin()
				defer pdf.TransformEnd()
				pdf.TransformRotate(rotateDeg, xMm, yMm)
				switch barcode.Orientation {
				case elements.FieldOrientation90:
					pdf.TransformTranslate(0, -hMm)
				case elements.FieldOrientation180:
					pdf.TransformTranslate(-wMm, -hMm)
				case elements.FieldOrientation270:
					pdf.TransformTranslate(-wMm, 0)
				}
			}

			pdf.ImageOptions(imgName, xMm, yMm, wMm, hMm, false, fpdf.ImageOptions{ImageType: "PNG"}, 0, "")

			if barcode.Line {
				drawEan13Text(pdf, state, options, text, x, y, barcode.LineAbove, float64(width), float64(height), float64(barWidth), float64(guardExtension))
			}

			return nil
		},
	}
}

// adjustEan13Position mirrors internal/drawers/barcode_ean13.go: when the
// barcode is rotated 90° or 180° via ^FO, the guard-bar extension makes the
// rendered image stick out past pos by guardExtension dots in the rotation
// axis, so we shift pos to keep the symbol's main body anchored at ^FO.
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

// drawEan13Text mirrors applyEan13TextToCtx in internal/drawers/barcode_ean13.go.
// All positional math is in dots so it stays line-for-line with the raster path;
// state.Dots converts to mm only at the call into pdf.Text.
func drawEan13Text(pdf *fpdf.Fpdf, state *DrawerState, options drawers.DrawerOptions, text string, posX, posY float64, lineAbove bool, width, height, barWidth, guardExtension float64) {
	setFillColor(pdf, state, elements.LineColorBlack)

	fontSizeDots := math.Round(width / 13)
	fontSizePt := FontSizePt(fontSizeDots, options.Dpmm)
	pdf.SetFont(FontHelvetica, "", fontSizePt)

	if len(text) == 13 && !lineAbove {
		formatted := wrapWithSpaces(fmt.Sprintf("%s||%s||%s", text[0:1], text[1:7], text[7:13]))
		x := posX - barWidth*10
		y := posY + height + fontSizeDots - guardExtension
		w := width + barWidth*5
		drawStringJustifiedPdf(pdf, state, formatted, x, y, w, []string{"|"})
		return
	}

	formatted := wrapWithSpaces(text)
	x := posX + barWidth*8
	y := posY - guardExtension/2
	w := width - barWidth*16
	drawStringJustifiedPdf(pdf, state, formatted, x, y, w, nil)
}

// drawStringJustifiedPdf mirrors drawStringJustified in internal/drawers/text_field.go:
// each whitespace-separated word is drawn at evenly distributed positions across
// maxWidth. hiddenWords are still allotted layout space but not drawn (used as
// invisible separators by the EAN-13 grouping).
func drawStringJustifiedPdf(pdf *fpdf.Fpdf, state *DrawerState, line string, xDots, yDots, maxWidthDots float64, hiddenWords []string) {
	words := strings.Fields(line)
	if len(words) == 0 {
		return
	}

	wordWidthsMm := make([]float64, len(words))
	totalWordWidthMm := 0.0
	for i, w := range words {
		wm := pdf.GetStringWidth(w)
		wordWidthsMm[i] = wm
		totalWordWidthMm += wm
	}

	maxWidthMm := state.Dots(maxWidthDots)
	spaceCount := len(words) - 1
	spaceWidthMm := 0.0
	if spaceCount > 0 {
		spaceWidthMm = (maxWidthMm - totalWordWidthMm) / float64(spaceCount)
		if spaceWidthMm < 0 {
			// Mirror raster fallback when content overflows the line.
			spaceWidthMm = pdf.GetStringWidth(" ") * 0.3
		}
	}

	yMm := state.Dots(yDots)
	cxMm := state.Dots(xDots)
	for i, word := range words {
		if !slices.Contains(hiddenWords, word) {
			pdf.Text(cxMm, yMm, word)
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
		Draw: func(pdf *fpdf.Fpdf, element any, _ drawers.DrawerOptions, state *DrawerState) error {
			barcode, ok := element.(*elements.BarcodePdf417WithData)
			if !ok {
				return nil
			}
			img, err := pdf417.Encode(barcode.Data, byte(barcode.Security), barcode.RowHeight, barcode.Columns)
			if err != nil {
				return fmt.Errorf("failed to encode pdf417 barcode: %w", err)
			}
			return embedBarcodeImage(pdf, state, img, barcode.Position, barcode.Orientation, fmt.Sprintf("p417_%p", barcode))
		},
	}
}

func bitmapBarcodeAztecDrawer() *ElementDrawer {
	return &ElementDrawer{
		Draw: func(pdf *fpdf.Fpdf, element any, _ drawers.DrawerOptions, state *DrawerState) error {
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
			return embedBarcodeImage(pdf, state, img, barcode.Position, barcode.Orientation, fmt.Sprintf("az_%p", barcode))
		},
	}
}
