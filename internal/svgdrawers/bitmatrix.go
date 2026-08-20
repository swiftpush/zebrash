package svgdrawers

import (
	"github.com/ingridhq/zebrash/internal/barcodes/utils"
	"github.com/ingridhq/zebrash/internal/elements"
	"github.com/ingridhq/zebrash/internal/svgwriter"
)

// drawBitMatrix renders the dark modules of a BitMatrix as filled rects.
// Mirrors pdfdrawers.drawBitMatrix — coalesce adjacent dark modules in a row
// into one rect to keep the SVG dense.
func drawBitMatrix(doc *svgwriter.Doc, state *DrawerState, m *utils.BitMatrix, xDots, yDots float64, dotsPerModuleX, dotsPerModuleY int) {
	if m == nil {
		return
	}
	width := m.GetWidth()
	height := m.GetHeight()
	if width == 0 || height == 0 {
		return
	}

	fill := inkColor(state, elements.LineColorBlack)

	dpmX := float64(dotsPerModuleX)
	dpmY := float64(dotsPerModuleY)

	for y := range height {
		x := 0
		for x < width {
			if !m.Get(x, y) {
				x++
				continue
			}
			runStart := x
			for x < width && m.Get(x, y) {
				x++
			}
			runLen := x - runStart

			rectX := xDots + float64(runStart)*dpmX
			rectY := yDots + float64(y)*dpmY
			rectW := float64(runLen) * dpmX
			rectH := dpmY

			doc.Rect(state.Dots(rectX), state.Dots(rectY), state.Dots(rectW), state.Dots(rectH), fill, "", 0)
		}
	}
}
