package pdfdrawers

import (
	"github.com/go-pdf/fpdf"

	"github.com/swiftpush/zebrash/internal/barcodes/utils"
	"github.com/swiftpush/zebrash/internal/elements"
)

// drawBitMatrix renders the dark modules of a BitMatrix as filled rects in PDF
// vector space. The matrix is treated as a grid of dotsPerModule-by-dotsPerModule
// dot squares positioned with its top-left at (xDots, yDots).
//
// Adjacent dark modules in a row are coalesced into a single rect so the output
// is dense: ~1 path operation per run of bars instead of one per module.
func drawBitMatrix(pdf *fpdf.Fpdf, state *DrawerState, m *utils.BitMatrix, xDots, yDots float64, dotsPerModuleX, dotsPerModuleY int) {
	if m == nil {
		return
	}
	width := m.GetWidth()
	height := m.GetHeight()
	if width == 0 || height == 0 {
		return
	}

	setFillColor(pdf, state, elements.LineColorBlack)

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

			pdf.Rect(state.Dots(rectX), state.Dots(rectY), state.Dots(rectW), state.Dots(rectH), "F")
		}
	}
}
