package pdfdrawers

import (
	"github.com/go-pdf/fpdf"

	"github.com/ingridhq/zebrash/drawers"
	"github.com/ingridhq/zebrash/internal/elements"
)

func NewGraphicBoxDrawer() *ElementDrawer {
	return &ElementDrawer{
		Draw: func(pdf *fpdf.Fpdf, element any, _ drawers.DrawerOptions, state *DrawerState) error {
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

			setFillColor(pdf, state, box.LineColor)
			setDrawColor(pdf, state, box.LineColor)

			xMm := state.Dots(x)
			yMm := state.Dots(y)
			wMm := state.Dots(width)
			hMm := state.Dots(height)
			borderMm := state.Dots(border)

			// A solid rectangle: border thickness >= the smaller dimension means a filled fill.
			if border*2 >= width || border*2 >= height || box.CornerRounding == 0 && border == 0 {
				if box.CornerRounding > 0 {
					rMm := state.Dots(roundingRadius(box.CornerRounding, width, height, 0))
					pdf.RoundedRect(xMm, yMm, wMm, hMm, rMm, "1234", "F")
				} else {
					pdf.Rect(xMm, yMm, wMm, hMm, "F")
				}
				return nil
			}

			// Hollow rectangle with stroked border. Match the raster path's offset
			// (border centered on the geometric edge) so a border of thickness B
			// renders centered on the rectangle's perimeter.
			pdf.SetLineWidth(borderMm)

			if box.CornerRounding > 0 {
				rMm := state.Dots(roundingRadius(box.CornerRounding, width, height, border))
				pdf.RoundedRect(xMm+borderMm/2, yMm+borderMm/2, wMm-borderMm, hMm-borderMm, rMm, "1234", "D")
			} else {
				pdf.Rect(xMm+borderMm/2, yMm+borderMm/2, wMm-borderMm, hMm-borderMm, "D")
			}

			return nil
		},
	}
}

// roundingRadius mirrors the raster path's CornerRounding scaling
// (drawRoundedRectangle in internal/drawers/graphic_box.go).
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
