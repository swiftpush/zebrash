package svgdrawers

import (
	"github.com/swiftpush/zebrash/drawers"
	"github.com/swiftpush/zebrash/internal/elements"
	"github.com/swiftpush/zebrash/internal/svgwriter"
)

// NewGraphicDiagonalLineDrawer ports internal/pdfdrawers/graphic_diagonal_line.go
// to SVG. A diagonal stroke is drawn as a filled parallelogram so the border
// thickness defines the perpendicular width of the line.
func NewGraphicDiagonalLineDrawer() *ElementDrawer {
	return &ElementDrawer{
		Draw: func(doc *svgwriter.Doc, element any, _ drawers.DrawerOptions, state *DrawerState) error {
			line, ok := element.(*elements.GraphicDiagonalLine)
			if !ok {
				return nil
			}

			x := float64(line.Position.X)
			y := float64(line.Position.Y)
			w := float64(line.Width)
			h := float64(line.Height)
			b := float64(line.BorderThickness)

			fill := inkColor(state, line.LineColor)

			var pts [][2]float64
			if line.TopToBottom {
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

			scaled := make([][2]float64, len(pts))
			for i, p := range pts {
				scaled[i] = [2]float64{state.Dots(p[0]), state.Dots(p[1])}
			}
			doc.Polygon(scaled, fill, "", 0)
			return nil
		},
	}
}
