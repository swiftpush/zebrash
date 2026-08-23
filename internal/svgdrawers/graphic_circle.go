package svgdrawers

import (
	"github.com/swiftpush/zebrash/drawers"
	"github.com/swiftpush/zebrash/internal/elements"
	"github.com/swiftpush/zebrash/internal/svgwriter"
)

// NewGraphicCircleDrawer ports internal/pdfdrawers/graphic_circle.go to SVG.
// ZPL ^GC draws a stroked circle; the diameter on the wire is the outer
// diameter so the stroke is centered on the geometric edge.
func NewGraphicCircleDrawer() *ElementDrawer {
	return &ElementDrawer{
		Draw: func(doc *svgwriter.Doc, element any, _ drawers.DrawerOptions, state *DrawerState) error {
			circle, ok := element.(*elements.GraphicCircle)
			if !ok {
				return nil
			}

			radius := float64(circle.CircleDiameter) / 2.0
			cx := float64(circle.Position.X) + radius
			cy := float64(circle.Position.Y) + radius

			stroke := inkColor(state, circle.LineColor)
			strokeWidthMm := state.Dots(float64(circle.BorderThickness))

			doc.Circle(state.Dots(cx), state.Dots(cy), state.Dots(radius), "none", stroke, strokeWidthMm)
			return nil
		},
	}
}
