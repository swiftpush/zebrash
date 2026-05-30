package svgdrawers

import "github.com/ingridhq/zebrash/internal/elements"

// DrawerState mirrors pdfdrawers.DrawerState — same auto-position bookkeeping,
// same dot→mm conversion factor, same reverse-print flag. Keep these in sync.
type DrawerState struct {
	AutoPosX float64
	AutoPosY float64

	// DotsToMm converts a ZPL dot coordinate to millimeters (1.0 / dpmm).
	DotsToMm float64

	// InverseInk is set while rendering a reverse-print element. Color helpers
	// flip black to white when this is true so a downstream feComposite/feBlend
	// filter on the group can XOR the field over the backdrop.
	InverseInk bool
}

func (state *DrawerState) UpdateAutomaticTextPosition(text *elements.TextField, w float64) {
	if !text.Position.CalculateFromBottom {
		return
	}

	x := float64(text.Position.X)
	y := float64(text.Position.Y)

	if !text.Position.AutomaticPosition {
		state.AutoPosX = x
		state.AutoPosY = y
	}

	switch text.Font.Orientation {
	case elements.FieldOrientation90:
		state.AutoPosY += w
	case elements.FieldOrientation180:
		state.AutoPosX -= w
	case elements.FieldOrientation270:
		state.AutoPosY -= w
	default:
		state.AutoPosX += w
	}
}

func (state *DrawerState) GetTextPosition(text *elements.TextField) (float64, float64) {
	if text.Position.AutomaticPosition {
		return state.AutoPosX, state.AutoPosY
	}

	return float64(text.Position.X), float64(text.Position.Y)
}

// Dots converts a ZPL dot coordinate to millimeters using the state's
// conversion factor.
func (state *DrawerState) Dots(dots float64) float64 {
	return dots * state.DotsToMm
}
