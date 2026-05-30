package pdfdrawers

import (
	"strings"
	"unicode"

	"github.com/go-pdf/fpdf"
	"github.com/ingridhq/zebrash/drawers"
	"github.com/ingridhq/zebrash/internal/elements"
)

func NewTextFieldDrawer() *ElementDrawer {
	return &ElementDrawer{
		Draw: func(pdf *fpdf.Fpdf, element any, options drawers.DrawerOptions, state *DrawerState) error {
			text, ok := element.(*elements.TextField)
			if !ok {
				return nil
			}

			text = adjustTextField(text)

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

			x, y := getTextTopLeftPos(text, w, h, state)
			state.UpdateAutomaticTextPosition(text, w)

			ax, ay := getTextAxAy(text)

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

	lines := wrapWords(pdf, text.Text, state.Dots(maxWidthDots))
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

// wrapWords mirrors gg.Context.WordWrap: split each paragraph into alternating
// non-space / space runs, greedily pack words into lines under maxWidthMm,
// and preserve the original internal whitespace inside each line. The naive
// strings.Fields + " " join collapses runs of whitespace to a single space,
// which makes the PDF GetStringWidth diverge from the raster MeasureString
// for any input that contains "double space" or tab runs (e.g. the centered
// "USPS TRACKING #  eVS" header on the USPS fixture).
func wrapWords(pdf *fpdf.Fpdf, s string, maxWidthMm float64) []string {
	var lines []string
	for _, paragraph := range strings.Split(s, "\n") {
		fields := splitOnSpace(paragraph)
		if len(fields) == 0 {
			lines = append(lines, "")
			continue
		}
		// Pad to even length so each iteration consumes a (word, gap) pair —
		// the last word may legitimately have no trailing gap.
		if len(fields)%2 == 1 {
			fields = append(fields, "")
		}

		current := ""
		for i := 0; i < len(fields); i += 2 {
			candidate := current + fields[i]
			if current != "" && pdf.GetStringWidth(candidate) > maxWidthMm {
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

// splitOnSpace mirrors gg's wrap.go: alternating runs of non-space / space
// runes, preserving the original whitespace so a wrapped line can be measured
// at its natural width (multiple spaces and all).
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
