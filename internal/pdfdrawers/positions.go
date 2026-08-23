package pdfdrawers

import "github.com/swiftpush/zebrash/internal/elements"

// adjustPositionFromBottom mirrors adjustImageTypeSetPosition in
// internal/drawers/element_drawer.go, but operates on dot dimensions instead of
// an image bounding box.
func adjustPositionFromBottom(pos elements.LabelPosition, widthDots, heightDots int, ori elements.FieldOrientation) elements.LabelPosition {
	if !pos.CalculateFromBottom {
		return pos
	}

	x := pos.X
	y := pos.Y

	switch ori {
	case elements.FieldOrientationNormal:
		y = max(y-heightDots, 0)
	case elements.FieldOrientation180:
		x -= widthDots
	case elements.FieldOrientation270:
		x = max(x-heightDots, 0)
		y -= widthDots
	}

	return elements.LabelPosition{X: x, Y: y}
}
