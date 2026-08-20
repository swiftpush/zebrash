package pdfdrawers

import (
	"github.com/go-pdf/fpdf"

	"github.com/ingridhq/zebrash/drawers"
	"github.com/ingridhq/zebrash/internal/elements"
)

func NewGraphicDiagonalLineDrawer() *ElementDrawer {
	return &ElementDrawer{
		Draw: func(pdf *fpdf.Fpdf, element any, _ drawers.DrawerOptions, state *DrawerState) error {
			line, ok := element.(*elements.GraphicDiagonalLine)
			if !ok {
				return nil
			}

			x := float64(line.Position.X)
			y := float64(line.Position.Y)
			w := float64(line.Width)
			h := float64(line.Height)
			b := float64(line.BorderThickness)

			setFillColor(pdf, state, line.LineColor)
			setDrawColor(pdf, state, line.LineColor)

			// Mirror drawDiagonalLine in internal/drawers/graphic_diagonal_line.go.
			// A diagonal stroke is drawn as a filled parallelogram so the border
			// thickness defines the perpendicular width of the line.
			var pts [][2]float64
			if line.TopToBottom {
				// raster's TopToBottom is the bottomToTop flag in drawDiagonalLine — kept as-is
				pts = [][2]float64{
					{x, y},
					{x + b, y},
					{x + b + w, y + h},
					{x + w, y + h},
				}
			} else {
				pts = [][2]float64{
					{x, y + h},
					{x + b, y + h},
					{x + b + w, y},
					{x + w, y},
				}
			}

			pdf.MoveTo(state.Dots(pts[0][0]), state.Dots(pts[0][1]))
			for _, p := range pts[1:] {
				pdf.LineTo(state.Dots(p[0]), state.Dots(p[1]))
			}
			pdf.ClosePath()
			pdf.DrawPath("F")

			return nil
		},
	}
}
