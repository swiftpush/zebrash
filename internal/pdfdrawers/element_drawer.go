package pdfdrawers

import (
	"github.com/go-pdf/fpdf"

	"github.com/swiftpush/zebrash/drawers"
)

type ElementDrawer struct {
	Draw func(pdf *fpdf.Fpdf, element any, options drawers.DrawerOptions, state *DrawerState) error
}
