package pdfdrawers

import (
	"github.com/go-pdf/fpdf"
	"github.com/ingridhq/zebrash/drawers"
	"github.com/ingridhq/zebrash/internal/elements"
	"github.com/ingridhq/zebrash/internal/textlayout"
)

func NewTextFieldDrawer() *ElementDrawer {
	return &ElementDrawer{
		Draw: func(pdf *fpdf.Fpdf, element any, options drawers.DrawerOptions, state *DrawerState) error {
			text, ok := element.(*elements.TextField)
			if !ok {
				return nil
			}

			text = textlayout.AdjustTextField(text)

			fontSizeDots := text.Font.GetSize()
			scaleX := text.Font.GetScaleX()
			fontSizePt := FontSizePt(fontSizeDots, options.Dpmm)

			family := ResolveFontFamily(text.Font)
			pdf.SetFont(family, "", fontSizePt)
			setFillColor(pdf, state, elements.LineColorBlack)
			setDrawColor(pdf, state, elements.LineColorBlack)

			// Width measured in mm at the current font size; raster operates in
			// dot-space using the same TTF, so a back-conversion to dots gives
			// us the same w that the raster path computes.
			wMm := pdf.GetStringWidth(text.Text) * scaleX
			w := wMm / state.DotsToMm
			h := fontSizeDots

			x, y := state.TextPosition(text)
			x, y = textlayout.TopLeftPos(text, w, h, x, y)
			state.Advance(text, w)

			ax, ay := textlayout.AxAy(text)

			if text.Block != nil {
				drawWrappedText(pdf, state, text, x, y-h, ax, ay, scaleX)
				return nil
			}

			drawSingleLine(pdf, state, text, x, y, ax, ay, w, h, scaleX)
			return nil
		},
	}
}

func drawSingleLine(pdf *fpdf.Fpdf, state *DrawerState, text *elements.TextField, x, y, ax, ay, w, h, scaleX float64) {
	xMm := state.Dots(x)
	yMm := state.Dots(y)
	wMm := state.Dots(w)
	hMm := state.Dots(h)

	rotateDeg := orientationToDegreesCCW(text.Font.Orientation)

	if rotateDeg != 0 || scaleX != 1.0 {
		pdf.TransformBegin()
		defer pdf.TransformEnd()
		if rotateDeg != 0 {
			pdf.TransformRotate(rotateDeg, xMm, yMm)
		}
		if scaleX != 1.0 {
			pdf.TransformScale(scaleX*100, 100, xMm, yMm)
		}
	}

	// getTextTopLeftPos already returns a baseline-aligned y (it adds 3h/4
	// for the raster path, which gg.DrawStringAnchored(ay=0) interprets as
	// the baseline — gg does not anchor at the top of the bounding box). So
	// the same y goes straight to fpdf.Text, which is also baseline-anchored.
	//
	// For ax-based alignment under a horizontal scale, the offset has to be
	// computed in pre-scale (natural) coordinates: the TransformScale above
	// pivots at (xMm, yMm) and will rescale anything we offset from it, so
	// using the post-scale wMm here would double-apply scaleX. The raster
	// path avoids this because gg.DrawStringAnchored(ax) anchors in pre-scale
	// space and ScaleAbout then expands around the same pivot.
	naturalWidthMm := wMm
	if scaleX != 0 {
		naturalWidthMm = wMm / scaleX
	}
	textX := xMm - ax*naturalWidthMm
	textY := yMm - ay*hMm

	pdf.Text(textX, textY, text.Text)
}

func drawWrappedText(pdf *fpdf.Fpdf, state *DrawerState, text *elements.TextField, x, y, ax, ay, scaleX float64) {
	block := text.Block
	maxWidthDots := float64(block.MaxWidth) / scaleX
	lineSpacingExtra := float64(block.LineSpacing)

	lines := textlayout.WrapWords(text.Text, state.Dots(maxWidthDots), pdf.GetStringWidth)
	if len(lines) == 0 {
		return
	}
	if block.MaxLines > 0 && len(lines) > block.MaxLines {
		lines = lines[:block.MaxLines]
	}

	fontSizeDots := text.Font.GetSize()
	hMm := state.Dots(fontSizeDots)
	lineSpacingMm := state.Dots(lineSpacingExtra) + hMm

	xMm := state.Dots(x)
	yMm := state.Dots(y)

	// Baseline of the first line. The raster path pivots rotation/scale at
	// the baseline anchor (the y returned by getTextTopLeftPos before the
	// caller subtracts h). yMm here is that anchor minus h, so we add hMm
	// back for the pivot.
	baselineYMm := yMm - ay*float64(len(lines))*lineSpacingMm + hMm

	rotateDeg := orientationToDegreesCCW(text.Font.Orientation)
	if rotateDeg != 0 || scaleX != 1.0 {
		pdf.TransformBegin()
		defer pdf.TransformEnd()
		if rotateDeg != 0 {
			pdf.TransformRotate(rotateDeg, xMm, baselineYMm)
		}
		if scaleX != 1.0 {
			pdf.TransformScale(scaleX*100, 100, xMm, baselineYMm)
		}
	}

	maxWidthMm := state.Dots(maxWidthDots)

	// maxWidthMm is already in the pre-scale coordinate system (it was
	// derived from block.MaxWidth/scaleX), so the natural string width
	// from pdf.GetStringWidth — which is also pre-scale — is the right
	// thing to compare against. The horizontal scale transform applied
	// above will collapse both onto the page consistently.
	//
	cursorY := baselineYMm
	for _, line := range lines {
		naturalWidthMm := pdf.GetStringWidth(line)

		var lineX float64
		switch block.Alignment {
		case elements.TextAlignmentRight:
			lineX = xMm + maxWidthMm - naturalWidthMm
		case elements.TextAlignmentCenter:
			lineX = xMm + (maxWidthMm-naturalWidthMm)/2
		default:
			lineX = xMm
		}

		pdf.Text(lineX, cursorY, line)
		cursorY += lineSpacingMm
	}
}
