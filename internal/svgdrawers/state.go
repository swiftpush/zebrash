package svgdrawers

import "github.com/ingridhq/zebrash/internal/textlayout"

// DrawerState mirrors pdfdrawers.DrawerState — same auto-position bookkeeping
// (embedded from textlayout), same dot→mm conversion factor, same
// reverse-print flag. Keep these in sync.
type DrawerState struct {
	// AutoPosition tracks the ^FT running pen position (Advance / TextPosition).
	textlayout.AutoPosition

	// DotsToMm converts a ZPL dot coordinate to millimeters (1.0 / dpmm).
	DotsToMm float64

	// InverseInk is set while rendering a reverse-print element. Color helpers
	// flip black to white when this is true so a downstream feComposite/feBlend
	// filter on the group can XOR the field over the backdrop.
	InverseInk bool
}

// Dots converts a ZPL dot coordinate to millimeters using the state's
// conversion factor.
func (state *DrawerState) Dots(dots float64) float64 {
	return dots * state.DotsToMm
}
