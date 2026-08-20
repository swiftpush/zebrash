package pdfdrawers

import (
	"github.com/go-pdf/fpdf"

	"github.com/ingridhq/zebrash/internal/elements"
)

// setDrawColor sets fpdf's draw (stroke) color to match a ZPL line color.
// Drawers should pass the state so reverse-print mode can flip the ink color.
func setDrawColor(pdf *fpdf.Fpdf, state *DrawerState, color elements.LineColor) {
	r, g, b := resolveInk(state, color)
	pdf.SetDrawColor(r, g, b)
}

// setFillColor sets fpdf's fill color to match a ZPL line color.
func setFillColor(pdf *fpdf.Fpdf, state *DrawerState, color elements.LineColor) {
	r, g, b := resolveInk(state, color)
	pdf.SetFillColor(r, g, b)
}

func resolveInk(state *DrawerState, color elements.LineColor) (int, int, int) {
	black := state == nil || !state.InverseInk
	if color == elements.LineColorWhite {
		black = !black
	}
	if black {
		return 0, 0, 0
	}
	return 255, 255, 255
}

// orientationToDegreesCCW converts a ZPL FieldOrientation (clockwise) to fpdf
// rotation degrees (counter-clockwise).
func orientationToDegreesCCW(ori elements.FieldOrientation) float64 {
	return -ori.GetDegrees()
}
