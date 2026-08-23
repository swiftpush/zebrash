package pdfdrawers

import "github.com/swiftpush/zebrash/internal/textlayout"

type DrawerState struct {
	// AutoPosition tracks the ^FT running pen position (Advance / TextPosition).
	textlayout.AutoPosition

	// DotsToMm converts a ZPL dot coordinate to millimeters (1.0 / dpmm).
	DotsToMm float64

	// InverseInk is set while rendering a reverse-print element. Color helpers
	// (setFillColor / setDrawColor) flip black to white when this is true so
	// that, combined with a "Difference" blend mode, drawing yields the XOR
	// effect ZPL's ^LR / ^FR commands describe.
	InverseInk bool
}

// Dots converts a ZPL dot coordinate to millimeters using the state's conversion factor.
func (state *DrawerState) Dots(dots float64) float64 {
	return dots * state.DotsToMm
}
