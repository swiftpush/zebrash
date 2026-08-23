package svgdrawers

import "github.com/swiftpush/zebrash/internal/elements"

// adjustPositionFromBottom mirrors pdfdrawers.adjustPositionFromBottom and
// the raster adjustImageTypeSetPosition: when the field anchors at the
// baseline (CalculateFromBottom), shift the top-left corner so the symbol
// renders above/left of the anchor depending on orientation.
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
