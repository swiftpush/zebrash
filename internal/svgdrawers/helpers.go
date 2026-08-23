package svgdrawers

import (
	"fmt"

	"github.com/swiftpush/zebrash/internal/elements"
)

// inkColor returns the CSS color string ("black" / "white") for a ZPL
// LineColor, accounting for reverse-print mode. Mirrors pdfdrawers.resolveInk.
func inkColor(state *DrawerState, color elements.LineColor) string {
	black := state == nil || !state.InverseInk
	if color == elements.LineColorWhite {
		black = !black
	}
	if black {
		return "black"
	}
	return "white"
}

// orientationToDegreesCCW converts a ZPL FieldOrientation (clockwise) to SVG
// rotation degrees (counter-clockwise around an explicit pivot via
// rotate(deg, cx, cy)). Mirrors pdfdrawers.orientationToDegreesCCW.
func orientationToDegreesCCW(ori elements.FieldOrientation) float64 {
	return -ori.GetDegrees()
}

// transformRotate returns an SVG transform string that rotates by deg about
// the explicit pivot (cx, cy). Equivalent to fpdf.TransformRotate.
func transformRotate(deg, cx, cy float64) string {
	return fmt.Sprintf("rotate(%g %g %g)", deg, cx, cy)
}

// transformTranslate returns an SVG transform string that translates by (dx, dy).
func transformTranslate(dx, dy float64) string {
	return fmt.Sprintf("translate(%g %g)", dx, dy)
}

// transformScaleAbout returns an SVG transform string that scales by (sx, sy)
// pivoted about (cx, cy). SVG's `scale(sx, sy)` is anchored at the origin, so
// to pivot we sandwich it between two translates.
func transformScaleAbout(sx, sy, cx, cy float64) string {
	return fmt.Sprintf("translate(%g %g) scale(%g %g) translate(%g %g)",
		cx, cy, sx, sy, -cx, -cy)
}

// fontSizeMm converts a ZPL font size measured in dots to millimeters using
// the state's conversion factor. SVG font-size with the "mm" unit makes the
// rendered glyph height match the dot-space height once the SVG is rasterized
// at the same DPI as the PNG backend.
func fontSizeMm(dots float64, state *DrawerState) float64 {
	return state.Dots(dots)
}

// fontSizePt is kept for parity with the PDF call signature; svgwriter.Text
// takes both and currently uses fontSizeMm.
func fontSizePt(dots float64, dpmm int) float64 {
	if dpmm == 0 {
		return dots
	}
	return dots * 72.0 / (float64(dpmm) * 25.4)
}
