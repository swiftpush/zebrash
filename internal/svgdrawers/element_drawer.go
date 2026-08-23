// Package svgdrawers implements zebrash's third rendering backend, alongside
// the raster PNG drawers in internal/drawers and the vector PDF drawers in
// internal/pdfdrawers. Each file mirrors the structure of its pdfdrawers
// sibling — port the PDF drawer line for line and substitute svgwriter calls
// for fpdf calls.
//
// The PDF backend is the spec for SVG. Coordinate system, units, and
// anchoring all match PDF (mm with baseline-anchored text). See
// docs/svg-backend.md for status of each drawer.
package svgdrawers

import (
	"github.com/swiftpush/zebrash/drawers"
	"github.com/swiftpush/zebrash/internal/svgwriter"
)

// ElementDrawer mirrors the PNG / PDF ElementDrawer shape: a single typed
// Draw function which is offered every element by the main loop and no-ops
// (returns nil) when the element isn't of its type.
type ElementDrawer struct {
	Draw func(doc *svgwriter.Doc, element any, options drawers.DrawerOptions, state *DrawerState) error
}
