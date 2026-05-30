package zebrash

import (
	"fmt"
	"io"
	"math"

	"github.com/go-pdf/fpdf"
	"github.com/ingridhq/gg"
	"github.com/ingridhq/zebrash/drawers"
	"github.com/ingridhq/zebrash/elements"
	drawers_internal "github.com/ingridhq/zebrash/internal/drawers"
	elements_internal "github.com/ingridhq/zebrash/internal/elements"
	"github.com/ingridhq/zebrash/internal/images"
	pdfdrawers_internal "github.com/ingridhq/zebrash/internal/pdfdrawers"
)

type reversePrintable interface {
	IsReversePrint() bool
}

type Drawer struct {
	elementDrawers    []*drawers_internal.ElementDrawer
	pdfElementDrawers []*pdfdrawers_internal.ElementDrawer
}

func NewDrawer() *Drawer {
	return &Drawer{
		elementDrawers: []*drawers_internal.ElementDrawer{
			drawers_internal.NewGraphicBoxDrawer(),
			drawers_internal.NewGraphicCircleDrawer(),
			drawers_internal.NewGraphicFieldDrawer(),
			drawers_internal.NewGraphicDiagonalLineDrawer(),
			drawers_internal.NewTextFieldDrawer(),
			drawers_internal.NewMaxicodeDrawer(),
			drawers_internal.NewBarcode128Drawer(),
			drawers_internal.NewBarcodeEan13Drawer(),
			drawers_internal.NewBarcode2of5Drawer(),
			drawers_internal.NewBarcode39Drawer(),
			drawers_internal.NewBarcodePdf417Drawer(),
			drawers_internal.NewBarcodeAztecDrawer(),
			drawers_internal.NewBarcodeDatamatrixDrawer(),
			drawers_internal.NewBarcodeQrDrawer(),
		},
		pdfElementDrawers: []*pdfdrawers_internal.ElementDrawer{
			pdfdrawers_internal.NewGraphicBoxDrawer(),
			pdfdrawers_internal.NewGraphicCircleDrawer(),
			pdfdrawers_internal.NewGraphicFieldDrawer(),
			pdfdrawers_internal.NewGraphicDiagonalLineDrawer(),
			pdfdrawers_internal.NewTextFieldDrawer(),
			pdfdrawers_internal.NewMaxicodeDrawer(),
			pdfdrawers_internal.NewBarcode128Drawer(),
			pdfdrawers_internal.NewBarcodeEan13Drawer(),
			pdfdrawers_internal.NewBarcode2of5Drawer(),
			pdfdrawers_internal.NewBarcode39Drawer(),
			pdfdrawers_internal.NewBarcodePdf417Drawer(),
			pdfdrawers_internal.NewBarcodeAztecDrawer(),
			pdfdrawers_internal.NewBarcodeDatamatrixDrawer(),
			pdfdrawers_internal.NewBarcodeQrDrawer(),
		},
	}
}

func (d *Drawer) DrawLabelAsPng(label elements.LabelInfo, output io.Writer, options drawers.DrawerOptions) error {
	options = options.WithDefaults()
	state := &drawers_internal.DrawerState{}

	widthMm := options.LabelWidthMm
	heightMm := options.LabelHeightMm
	dpmm := options.Dpmm

	labelWidth := int(math.Ceil(widthMm * float64(dpmm)))
	imageWidth := labelWidth
	if label.PrintWidth > 0 {
		imageWidth = min(labelWidth, label.PrintWidth)
	}

	imageHeight := int(math.Ceil(heightMm * float64(dpmm)))

	gCtx := gg.NewContext(imageWidth, imageHeight)
	gCtx.SetColor(images.ColorWhite)
	gCtx.Clear()

	var gReversePrintBuff *gg.Context

	for _, element := range label.Elements {
		reversePrint := false

		if el, ok := element.(reversePrintable); ok {
			reversePrint = el.IsReversePrint()
		}

		gCtx2 := gCtx
		if reversePrint {
			if gReversePrintBuff == nil {
				gReversePrintBuff = gg.NewContext(imageWidth, imageHeight)
			} else if err := images.Zerofill(gReversePrintBuff.Image()); err != nil {
				return fmt.Errorf("failed to clear reverse print buffer: %w", err)
			}

			gCtx2 = gReversePrintBuff
		}

		for _, drawer := range d.elementDrawers {
			err := drawer.Draw(gCtx2, element, options, state)
			if err != nil {
				return fmt.Errorf("failed to draw zpl element: %w", err)
			}
		}

		if reversePrint {
			if err := images.ReversePrint(gCtx2.Image(), gCtx.Image()); err != nil {
				return err
			}
		}
	}

	// If print width was less than label width
	// or label was inverted
	// Draw everything onto the new, wider image and center / rotate the content
	invertLabel := (options.EnableInvertedLabels && label.Inverted)
	if (imageWidth != labelWidth) || invertLabel {
		imgCtx := gCtx
		gCtx = gg.NewContext(labelWidth, imageHeight)
		gCtx.SetColor(images.ColorWhite)
		gCtx.Clear()

		if invertLabel {
			gCtx.Translate(float64(labelWidth), float64(imageHeight))
			gCtx.Scale(-1, -1)
		}

		gCtx.DrawImage(imgCtx.Image(), (labelWidth-imageWidth)/2, 0)
	}

	if options.GrayscaleOutput {
		return images.EncodeGrayscale(output, gCtx.Image())
	}
	return images.EncodeMonochrome(output, gCtx.Image())
}

// DrawLabelAsPdf renders a parsed ZPL label as a single-page vector PDF.
//
// Coordinate system: ZPL elements live in dot-space (Position.X/Y in dots).
// The PDF document uses millimeter units; each PDF drawer scales dot
// coordinates via state.DotsToMm.
//
// Page-level transforms (^POI inversion, print-width centering) are applied
// once around the element-drawing loop via fpdf's transform stack — the same
// CTM treatment the raster pipeline applies to its final canvas.
//
// DrawerOptions.GrayscaleOutput is ignored on the PDF path (vector PDFs have
// no grayscale/monochrome distinction).
func (d *Drawer) DrawLabelAsPdf(label elements.LabelInfo, output io.Writer, options drawers.DrawerOptions) error {
	options = options.WithDefaults()

	widthMm := options.LabelWidthMm
	heightMm := options.LabelHeightMm
	dpmm := options.Dpmm

	labelWidth := int(math.Ceil(widthMm * float64(dpmm)))
	imageWidth := labelWidth
	if label.PrintWidth > 0 {
		imageWidth = min(labelWidth, label.PrintWidth)
	}

	dotsToMm := 1.0 / float64(dpmm)

	pdf := fpdf.NewCustom(&fpdf.InitType{
		UnitStr: "mm",
		Size:    fpdf.SizeType{Wd: widthMm, Ht: heightMm},
	})
	pdf.SetMargins(0, 0, 0)
	pdf.SetAutoPageBreak(false, 0)
	pdfdrawers_internal.RegisterBuiltInFonts(pdf)
	pdf.AddPage()

	state := &pdfdrawers_internal.DrawerState{
		DotsToMm: dotsToMm,
	}

	invertLabel := options.EnableInvertedLabels && label.Inverted
	xOffsetMm := float64(labelWidth-imageWidth) * dotsToMm / 2.0

	transformActive := invertLabel || xOffsetMm != 0
	if transformActive {
		pdf.TransformBegin()
		if invertLabel {
			pdf.TransformTranslate(widthMm, heightMm)
			pdf.TransformScale(-100, -100, 0, 0)
		}
		if xOffsetMm != 0 {
			pdf.TransformTranslate(xOffsetMm, 0)
		}
	}

	for _, element := range label.Elements {
		reverse := false
		if el, ok := element.(reversePrintable); ok {
			reverse = el.IsReversePrint()
		}

		// Difference blend + white ink ("XOR over the backdrop") survives MuPDF
		// rasterization only for TextField — and even then only by accident, see
		// docs/pdf-reverse-print.md. For vector primitives (boxes, lines,
		// barcodes) MuPDF silently drops the blend mode for path fills, so a
		// white-on-white reverse-print rectangle ends up invisible. The
		// overwhelmingly common ^FR^GB pattern on carrier labels just sits over
		// the white label background, so falling back to Normal black ink there
		// lands the right output for that case (and matches what the raster
		// path emits via XOR with white). Reverse-print over a *filled* black
		// region is the rare counter-case where this regresses; that's already
		// only working incidentally for text and never worked here.
		applyBlend := reverse
		if _, isText := element.(*elements_internal.TextField); !isText {
			applyBlend = false
		}

		if applyBlend {
			pdf.SetAlpha(1.0, "Difference")
			state.InverseInk = true
		}

		for _, drawer := range d.pdfElementDrawers {
			if err := drawer.Draw(pdf, element, options, state); err != nil {
				return fmt.Errorf("failed to draw zpl element to pdf: %w", err)
			}
		}

		if applyBlend {
			state.InverseInk = false
			pdf.SetAlpha(1.0, "Normal")
		}
	}

	if transformActive {
		pdf.TransformEnd()
	}

	if err := pdf.Output(output); err != nil {
		return fmt.Errorf("failed to write pdf: %w", err)
	}
	return nil
}
