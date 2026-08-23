package drawers

import "github.com/swiftpush/zebrash/internal/textlayout"

// DrawerState carries the mutable state threaded through the PNG element
// drawers. The embedded AutoPosition provides Advance / TextPosition.
type DrawerState struct {
	textlayout.AutoPosition
}
