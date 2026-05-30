package pdfdrawers

import (
	"github.com/go-pdf/fpdf"
	"github.com/ingridhq/zebrash/drawers"
	"github.com/ingridhq/zebrash/internal/elements"
)

func NewGraphicCircleDrawer() *ElementDrawer {
	return &ElementDrawer{
		Draw: func(pdf *fpdf.Fpdf, element any, _ drawers.DrawerOptions, state *DrawerState) error {
			circle, ok := element.(*elements.GraphicCircle)
			if !ok {
				return nil
			}

			radius := float64(circle.CircleDiameter) / 2.0
			cx := float64(circle.Position.X) + radius
			cy := float64(circle.Position.Y) + radius

			setDrawColor(pdf, state, circle.LineColor)
			pdf.SetLineWidth(state.Dots(float64(circle.BorderThickness)))
			pdf.Circle(state.Dots(cx), state.Dots(cy), state.Dots(radius), "D")

			return nil
		},
	}
}
