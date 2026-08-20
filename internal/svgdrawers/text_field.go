package svgdrawers

import (
	"github.com/ingridhq/zebrash/drawers"
	"github.com/ingridhq/zebrash/internal/elements"
	"github.com/ingridhq/zebrash/internal/svgwriter"
	"github.com/ingridhq/zebrash/internal/textlayout"
)

// NewTextFieldDrawer ports internal/pdfdrawers/text_field.go to SVG.
//
// Width measurement is delegated to the SVG rasterizer/viewer via text-anchor
// ("start" / "middle" / "end") and we only have to position the anchor point.
// This avoids needing a glyph shaper inside the writer.
//
// Coordinate-system notes from pdfdrawers/text_field.go apply identically —
// see CLAUDE.md "Coordinate-system gotcha". Status: single-line ports cleanly
// (limited by rasterizer font support). ^FB wrapped text is approximate —
// see "Text width / wrapping" in docs/svg-backend.md.
func NewTextFieldDrawer() *ElementDrawer {
	return &ElementDrawer{
		Draw: func(doc *svgwriter.Doc, element any, options drawers.DrawerOptions, state *DrawerState) error {
			text, ok := element.(*elements.TextField)
			if !ok {
				return nil
			}

			text = textlayout.AdjustTextField(text)

			fontSizeDots := text.Font.GetSize()
			scaleX := text.Font.GetScaleX()

			family, weight := ResolveFontFamilyWeight(text.Font)
			fontMm := fontSizeMm(fontSizeDots, state)
			fontPt := fontSizePt(fontSizeDots, options.Dpmm)
			fill := inkColor(state, elements.LineColorBlack)

			// Measure the real advance width using the embedded TTF.
			wDots := MeasureStringDots(text.Font, fontSizeDots, text.Text) * scaleX
			w := wDots
			h := fontSizeDots

			x, y := state.TextPosition(text)
			x, y = textlayout.TopLeftPos(text, w, h, x, y)
			state.Advance(text, w)

			ax, _ := textlayout.AxAy(text)

			if text.Block != nil {
				drawWrappedText(doc, state, text, family, weight, fontMm, fontPt, fill, x, y-h, ax, scaleX)
				return nil
			}

			drawSingleLine(doc, state, text, family, weight, fontMm, fontPt, fill, x, y, ax, scaleX)
			return nil
		},
	}
}

func drawSingleLine(doc *svgwriter.Doc, state *DrawerState, text *elements.TextField, family, weight string, fontMm, fontPt float64, fill string, x, y, ax, scaleX float64) {
	xMm := state.Dots(x)
	yMm := state.Dots(y)

	rotateDeg := orientationToDegreesCCW(text.Font.Orientation)

	groups := 0
	if rotateDeg != 0 {
		doc.GroupTransform(transformRotate(rotateDeg, xMm, yMm))
		groups++
	}
	if scaleX != 1.0 && scaleX != 0 {
		doc.GroupTransform(transformScaleAbout(scaleX, 1, xMm, yMm))
		groups++
	}

	anchor := "start"
	if ax == 1 {
		anchor = "end"
	}

	doc.Text(xMm, yMm, family, weight, fontPt, fontMm, fill, anchor, text.Text)

	for i := 0; i < groups; i++ {
		doc.EndGroup()
	}
}

func drawWrappedText(doc *svgwriter.Doc, state *DrawerState, text *elements.TextField, family, weight string, fontMm, fontPt float64, fill string, x, y, ax, scaleX float64) {
	_ = ax
	block := text.Block
	maxWidthDots := float64(block.MaxWidth) / scaleX
	lineSpacingExtra := float64(block.LineSpacing)

	fontSizeDots := text.Font.GetSize()
	measure := measureFn(text.Font, fontSizeDots, state.DotsToMm)

	lines := textlayout.WrapWords(text.Text, state.Dots(maxWidthDots), measure)
	if len(lines) == 0 {
		return
	}
	if block.MaxLines > 0 && len(lines) > block.MaxLines {
		lines = lines[:block.MaxLines]
	}

	hMm := state.Dots(fontSizeDots)
	lineSpacingMm := state.Dots(lineSpacingExtra) + hMm

	xMm := state.Dots(x)
	yMm := state.Dots(y)

	// Single-line baseline of the first line. ax in ^FB is the AlignmentTop
	// flag, not the same as the single-line anchor — block alignment is
	// implemented via per-line text-anchor below.
	baselineYMm := yMm + hMm

	rotateDeg := orientationToDegreesCCW(text.Font.Orientation)
	groups := 0
	if rotateDeg != 0 {
		doc.GroupTransform(transformRotate(rotateDeg, xMm, baselineYMm))
		groups++
	}
	if scaleX != 1.0 && scaleX != 0 {
		doc.GroupTransform(transformScaleAbout(scaleX, 1, xMm, baselineYMm))
		groups++
	}

	maxWidthMm := state.Dots(maxWidthDots)

	cursorY := baselineYMm
	for _, line := range lines {
		naturalWidthMm := measure(line)

		var lineX float64
		switch block.Alignment {
		case elements.TextAlignmentRight:
			lineX = xMm + maxWidthMm - naturalWidthMm
		case elements.TextAlignmentCenter:
			lineX = xMm + (maxWidthMm-naturalWidthMm)/2
		default:
			lineX = xMm
		}

		doc.Text(lineX, cursorY, family, weight, fontPt, fontMm, fill, "start", line)
		cursorY += lineSpacingMm
	}

	for i := 0; i < groups; i++ {
		doc.EndGroup()
	}
}
