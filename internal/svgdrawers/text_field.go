package svgdrawers

import (
	"strings"
	"unicode"

	"github.com/ingridhq/zebrash/drawers"
	"github.com/ingridhq/zebrash/internal/elements"
	"github.com/ingridhq/zebrash/internal/svgwriter"
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

			text = adjustTextField(text)

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

			x, y := getTextTopLeftPos(text, w, h, state)
			state.UpdateAutomaticTextPosition(text, w)

			ax, _ := getTextAxAy(text)

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

	lines := wrapWords(text.Text, state.Dots(maxWidthDots), measure)
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

// wrapWords is a port of pdfdrawers.wrapWords that uses a caller-supplied
// measure function instead of pdf.GetStringWidth. The measure function must
// return the rendered width of a string in millimetres.
//
// Algorithm: split each paragraph into alternating (word, gap) pairs via
// splitOnSpace, greedily pack words into lines, preserve internal whitespace.
func wrapWords(s string, maxWidthMm float64, measure func(string) float64) []string {
	var lines []string
	for _, paragraph := range strings.Split(s, "\n") {
		fields := splitOnSpace(paragraph)
		if len(fields) == 0 {
			lines = append(lines, "")
			continue
		}
		// Pad to even length so each iteration consumes a (word, gap) pair.
		if len(fields)%2 == 1 {
			fields = append(fields, "")
		}

		current := ""
		for i := 0; i < len(fields); i += 2 {
			candidate := current + fields[i]
			if current != "" && measure(candidate) > maxWidthMm {
				lines = append(lines, strings.TrimSpace(current))
				current = ""
				candidate = fields[i]
			}
			current = candidate + fields[i+1]
		}
		if current != "" {
			lines = append(lines, strings.TrimSpace(current))
		}
	}
	return lines
}

// splitOnSpace mirrors pdfdrawers.splitOnSpace / gg's wrap.go: alternating
// runs of non-space / space runes, preserving original whitespace.
func splitOnSpace(x string) []string {
	if x == "" {
		return nil
	}
	var result []string
	pi := 0
	ps := false
	for i, c := range x {
		s := unicode.IsSpace(c)
		if s != ps && i > 0 {
			result = append(result, x[pi:i])
			pi = i
		}
		ps = s
	}
	result = append(result, x[pi:])
	return result
}

func adjustTextField(text *elements.TextField) *elements.TextField {
	res := *text
	if text.Font.Name == "B" {
		res.Text = strings.ToUpper(res.Text)
	}
	return &res
}

// getTextTopLeftPos is a verbatim port of pdfdrawers.getTextTopLeftPos —
// same offsets, same orientation handling. Returns a baseline-aligned y.
func getTextTopLeftPos(text *elements.TextField, w, h float64, state *DrawerState) (float64, float64) {
	x, y := state.GetTextPosition(text)

	lines := 1.0
	spacing := 0.0
	if text.Block != nil {
		lines = float64(max(text.Block.MaxLines, 1))
		spacing = float64(text.Block.LineSpacing)
		w = float64(text.Block.MaxWidth)
	}

	if !text.Position.CalculateFromBottom {
		switch text.Font.Orientation {
		case elements.FieldOrientation90:
			return x + h/4, y
		case elements.FieldOrientation180:
			return x + w, y + h/4
		case elements.FieldOrientation270:
			return x + 3*h/4, y + w
		default:
			return x, y + 3*h/4
		}
	}

	offset := (lines - 1) * (h + spacing)
	switch text.Font.Orientation {
	case elements.FieldOrientation90:
		return x + offset, y
	case elements.FieldOrientation180:
		return x, y + offset
	case elements.FieldOrientation270:
		return x - offset, y
	default:
		return x, y - offset
	}
}

func getTextAxAy(text *elements.TextField) (float64, float64) {
	if text.Alignment == elements.FieldAlignmentRight {
		return 1, 0
	}
	return 0, 0
}
