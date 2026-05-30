package svgdrawers

import (
	"github.com/ingridhq/zebrash/drawers"
	"github.com/ingridhq/zebrash/internal/elements"
	"github.com/ingridhq/zebrash/internal/svgwriter"
)

// NewGraphicBoxDrawer ports internal/pdfdrawers/graphic_box.go to SVG.
// Solid rect when the border meets/exceeds either dimension; hollow stroked
// rect otherwise, with the stroke centered on the geometric edge to match
// the raster path.
func NewGraphicBoxDrawer() *ElementDrawer {
	return &ElementDrawer{
		Draw: func(doc *svgwriter.Doc, element any, _ drawers.DrawerOptions, state *DrawerState) error {
			box, ok := element.(*elements.GraphicBox)
			if !ok {
				return nil
			}

			width := float64(box.Width)
			height := float64(box.Height)
			border := float64(box.BorderThickness)

			if border > width {
				width = border
			}
			if border > height {
				height = border
			}

			x := float64(box.Position.X)
			y := float64(box.Position.Y)

			fill := inkColor(state, box.LineColor)

			xMm := state.Dots(x)
			yMm := state.Dots(y)
			wMm := state.Dots(width)
			hMm := state.Dots(height)
			borderMm := state.Dots(border)

			if border*2 >= width || border*2 >= height || box.CornerRounding == 0 && border == 0 {
				if box.CornerRounding > 0 {
					rMm := state.Dots(roundingRadius(box.CornerRounding, width, height, 0))
					doc.RoundedRect(xMm, yMm, wMm, hMm, rMm, fill, "", 0)
				} else {
					doc.Rect(xMm, yMm, wMm, hMm, fill, "", 0)
				}
				return nil
			}

			// Hollow rectangle: stroke centered on the geometric edge so the
			// rendered border width is exactly border on each side.
			if box.CornerRounding > 0 {
				rMm := state.Dots(roundingRadius(box.CornerRounding, width, height, border))
				doc.RoundedRect(xMm+borderMm/2, yMm+borderMm/2, wMm-borderMm, hMm-borderMm, rMm, "none", fill, borderMm)
			} else {
				doc.Rect(xMm+borderMm/2, yMm+borderMm/2, wMm-borderMm, hMm-borderMm, "none", fill, borderMm)
			}

			return nil
		},
	}
}

// roundingRadius mirrors pdfdrawers.roundingRadius.
func roundingRadius(rounding int, w, h, border float64) float64 {
	side := min(w, h)
	if border > 0 {
		side = min(w-2*border, h-2*border)
	}
	if side <= 0 {
		return 0
	}
	return float64(rounding) * side / 16.0
}
