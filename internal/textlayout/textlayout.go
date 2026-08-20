// Package textlayout holds the backend-agnostic text positioning logic shared
// by the PNG (internal/drawers), PDF (internal/pdfdrawers) and SVG
// (internal/svgdrawers) rendering backends.
//
// These functions used to be copy-pasted (byte-for-byte, in most cases) into
// each backend's text_field.go and state.go. They depend only on the parsed
// element IR (internal/elements), not on any rendering API, so they live here
// once and each backend calls in.
package textlayout

import (
	"strings"
	"unicode"

	"github.com/ingridhq/zebrash/internal/elements"
)

// AutoPosition tracks the running pen position for automatic-position text
// fields (^FT without explicit coordinates). Backends embed it in their
// DrawerState so the Advance/TextPosition methods are promoted onto the state.
type AutoPosition struct {
	X float64
	Y float64
}

// Advance moves the running position forward after a field of advance width w
// has been drawn. It only tracks fields anchored from the bottom (^FT).
func (p *AutoPosition) Advance(text *elements.TextField, w float64) {
	if !text.Position.CalculateFromBottom {
		return
	}

	if !text.Position.AutomaticPosition {
		p.X = float64(text.Position.X)
		p.Y = float64(text.Position.Y)
	}

	switch text.Font.Orientation {
	case elements.FieldOrientation90:
		p.Y += w
	case elements.FieldOrientation180:
		p.X -= w
	case elements.FieldOrientation270:
		p.Y -= w
	default:
		p.X += w
	}
}

// TextPosition returns the origin of a text field, resolving automatic
// position to the running pen position.
func (p *AutoPosition) TextPosition(text *elements.TextField) (float64, float64) {
	if text.Position.AutomaticPosition {
		return p.X, p.Y
	}

	return float64(text.Position.X), float64(text.Position.Y)
}

// TopLeftPos converts a field origin (x, y) into the anchor point each backend
// draws from, accounting for orientation, block layout and ^FT vs ^FO. The
// returned y is baseline-aligned (see CLAUDE.md "Coordinate-system gotcha").
// w and h are the measured text width and height in the same units as x, y.
func TopLeftPos(text *elements.TextField, w, h, x, y float64) (float64, float64) {
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

// AxAy returns the (ax, ay) draw-anchor factors for a field's alignment. ay is
// always 0; only right alignment shifts the anchor (ax = 1).
func AxAy(text *elements.TextField) (float64, float64) {
	if text.Alignment == elements.FieldAlignmentRight {
		return 1, 0
	}

	return 0, 0
}

// AdjustTextField applies font-driven text transforms, returning a copy. Font
// "B" (bold) renders in all uppercase.
func AdjustTextField(text *elements.TextField) *elements.TextField {
	res := *text
	if text.Font.Name == "B" {
		res.Text = strings.ToUpper(res.Text)
	}

	return &res
}

// WrapWords greedily wraps s to lines no wider than maxWidth, measuring each
// candidate with measure. It mirrors gg.Context.WordWrap: paragraphs split on
// "\n", words packed via splitOnSpace so runs of internal whitespace are
// preserved (a naive Fields+join would collapse them and diverge from the
// raster MeasureString). maxWidth and measure's result must share units.
func WrapWords(s string, maxWidth float64, measure func(string) float64) []string {
	var lines []string
	for paragraph := range strings.SplitSeq(s, "\n") {
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
			if current != "" && measure(candidate) > maxWidth {
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
// at its natural width.
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
