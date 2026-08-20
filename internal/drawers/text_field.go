package drawers

import (
	"slices"
	"strings"

	"github.com/golang/freetype/truetype"
	"github.com/ingridhq/gg"
	"github.com/ingridhq/zebrash/drawers"
	"github.com/ingridhq/zebrash/internal/assets"
	"github.com/ingridhq/zebrash/internal/elements"
	"github.com/ingridhq/zebrash/internal/textlayout"
)

var (
	font0  = mustLoadFont(assets.FontHelveticaBold)
	font1  = mustLoadFont(assets.FontDejavuSansMono)
	fontB  = mustLoadFont(assets.FontDejavuSansMonoBold)
	fontGS = mustLoadFont(assets.FontZplGS)
)

func NewTextFieldDrawer() *ElementDrawer {
	return &ElementDrawer{
		Draw: func(gCtx *gg.Context, element any, _ drawers.DrawerOptions, state *DrawerState) error {
			text, ok := element.(*elements.TextField)
			if !ok {
				return nil
			}

			text = textlayout.AdjustTextField(text)

			fontSize := text.Font.GetSize()
			scaleX := text.Font.GetScaleX()
			face := truetype.NewFace(getTffFont(text.Font), &truetype.Options{Size: fontSize})
			gCtx.SetFontFace(face)

			setLineColor(gCtx, elements.LineColorBlack)

			w, h := gCtx.MeasureString(text.Text)
			w *= scaleX

			x, y := state.TextPosition(text)
			x, y = textlayout.TopLeftPos(text, w, h, x, y)
			state.Advance(text, w)

			ax, ay := textlayout.AxAy(text)

			if rotate := text.Font.Orientation.GetDegrees(); rotate != 0 {
				gCtx.RotateAbout(gg.Radians(rotate), x, y)
			}

			if scaleX != 1.0 {
				gCtx.ScaleAbout(scaleX, 1, x, y)
			}

			defer gCtx.Identity()

			if text.Block != nil {
				maxWidth := float64(text.Block.MaxWidth) / scaleX
				drawStringWrapped(gCtx, text.Text, x, y-h, ax, ay, maxWidth, 1+float64(text.Block.LineSpacing)/h, text.Block.Alignment)
			} else {
				gCtx.DrawStringAnchored(text.Text, x, y, ax, ay)
			}

			return nil
		},
	}
}

func getTffFont(font elements.FontInfo) *truetype.Font {
	if font.CustomFont != nil {
		return font.CustomFont
	}

	switch font.Name {
	case "0":
		return font0
	case "B":
		return fontB
	case "GS":
		return fontGS
	default:
		return font1
	}
}

func mustLoadFont(fontData []byte) *truetype.Font {
	font, err := truetype.Parse(fontData)
	if err != nil {
		panic(err.Error())
	}

	return font
}

// Similar to gCtx.DrawStringWrapped but supports justified alignment
func drawStringWrapped(gCtx *gg.Context, s string, x, y, ax, ay, width, lineSpacing float64, align elements.TextAlignment) {
	fontHeight := gCtx.FontHeight()
	lines := gCtx.WordWrap(s, width)

	h := float64(len(lines)) * fontHeight * lineSpacing
	h -= (lineSpacing - 1) * fontHeight

	x -= ax * width
	y -= ay * h
	switch align {
	case elements.TextAlignmentLeft, elements.TextAlignmentJustified:
		ax = 0
	case elements.TextAlignmentCenter:
		ax = 0.5
		x += width / 2
	case elements.TextAlignmentRight:
		ax = 1
		x += width
	}
	ay = 1

	lastLine := len(lines) - 1

	for i, line := range lines {
		switch {
		case align == elements.TextAlignmentJustified && i < lastLine:
			drawStringJustified(gCtx, line, x, y, ax, ay, width, nil)
		default:
			gCtx.DrawStringAnchored(line, x, y, ax, ay)
		}

		y += fontHeight * lineSpacing
	}
}

func drawStringJustified(gCtx *gg.Context, line string, x, y, ax, ay, maxWidth float64, hiddenWords []string) {
	words := strings.Fields(line)
	fontHeight := gCtx.FontHeight()

	totalWordWidth := 0.0
	wordsWidth := make([]float64, len(words))
	for i, word := range words {
		w, _ := gCtx.MeasureString(word)
		wordsWidth[i] = w
		totalWordWidth += w
	}

	spaceCount := len(words) - 1
	spaceWidth := 0.0
	if spaceCount > 0 {
		spaceWidth = (maxWidth - totalWordWidth) / float64(spaceCount)
		if spaceWidth < 0 {
			spaceWidth = fontHeight * 0.3
		}
	}

	cx := x
	for i, word := range words {
		if !slices.Contains(hiddenWords, word) {
			gCtx.DrawStringAnchored(word, cx, y, ax, ay)
		}

		cx += wordsWidth[i] + spaceWidth
	}
}
