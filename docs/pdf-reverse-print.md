# PDF reverse-print (`^FR`) — known limitation

ZPL's reverse-print (`^FR` per field, `^LR` per label) is a **binary XOR** of the
field's mask against whatever has already been drawn. The PNG backend implements
this directly: `^FR` elements are rendered into `gReversePrintBuff` and
XOR-merged with the main canvas via `images.ReversePrint`
(`internal/images/reverse_print.go`). Where the mask has ink, the destination
pixel is inverted; everywhere else, the destination is untouched. Net effect:

- text-glyph mask over a **white** region → black glyph (XOR(0,1) = 1)
- text-glyph mask over a **black** region → white glyph (XOR(1,1) = 0)

The PDF backend can't do pixel XOR — vector PDFs only have blend modes. The
closest blend mode to XOR is `Difference` (`|backdrop − source|`), which is
mathematically equivalent to XOR for binary 0/255 sources:

- white source over white backdrop → 0 (black)
- white source over black backdrop → 255 (white)
- black source over white → 255 (white) ❌ inverted from XOR
- black source over black → 0 (black) ❌ inverted from XOR

So `Difference` produces XOR semantics **only when the source is white**. The
PDF drawer (`drawer.go`'s reverse-print branch) sets `SetAlpha(1.0,
"Difference")` and the helpers' `InverseInk = true` so that `setFillColor`
flips black to white during reverse-print elements.

## The MuPDF / TTF bug

The test harness rasterizes generated PDFs through MuPDF (`github.com/gen2brain/go-fitz`)
to compare against the PNG goldens. Empirically (see `/tmp/test_diff_ttf.png`
in the repro from 2026-04-30): **MuPDF applies `Difference` blend mode to
built-in PostScript fonts but silently ignores it for UTF-8 TTF text.**

Our pipeline embeds every Zebra font as UTF-8 TTF via
`AddUTF8FontFromBytes` (see `internal/pdfdrawers/fonts.go`'s
`RegisterBuiltInFonts`), so `^FR` text rendering hits the broken MuPDF path.

A second MuPDF gap (confirmed 2026-04-30 against `glsdk_return`): **MuPDF
also drops `Difference` for path fills**. A `^FR^GB` rectangle drawn as a
white-filled `re f` under `/GS … gs` (Difference) rasterizes through MuPDF
as plain white-on-white — invisible — even though Acrobat renders it
correctly. The same goes for the bitmatrix path fills used by `^BX`
(DataMatrix) and other barcodes that emit individual filled cells. So the
"invisible reverse-print" bug isn't a TTF-only issue; any vector primitive
hits it.

Because of this, the drawer loop in `drawer.go` only enters the
`SetAlpha("Difference") + InverseInk` branch when the element is a
`*elements.TextField`. For graphic primitives and barcodes we fall back to
Normal blend mode with the original (black) ink: that produces a correct
rendering for `^FR` over a white backdrop (the overwhelmingly common case
on carrier labels) and an invisible one for `^FR` over a black backdrop —
which is the same outcome the broken Difference path produces, so it is
strictly an improvement.

## Why `^FR` text on white backgrounds happens to render correctly anyway

It looks like `^FR` text on white backgrounds works in the current PDF output,
even though `Difference` is being silently dropped. That is **incidental, not
intentional**. The mechanism:

1. `text_field.go` calls `setFillColor(... InverseInk)` → emits `1 g` (white).
2. fpdf's `Text()` always wraps each `Tj` in `q <text-color> Q` whenever
   `f.colorFlag` is set. The text color defaults to `0 g` (black) and is
   never updated by `setFillColor`, so the inner `q 0 g ... Q` overrides the
   white fill back to black for the actual glyph rendering.
3. With `Difference` blended-but-ignored by MuPDF, the rendered ink is just
   black-source-on-white-background → black text. Visible. Looks correct.

The same code path on a **black** background renders black ink on black
background → invisible. That is the bug visible on `dhlparceluk.zpl`'s
GL55 6HU postcode (white-on-black inside a `^GB` filled box).

## Why the obvious fix regresses

Adding `setTextColor` so the text color also flips to white during reverse-print
makes the GL55 6HU case render correctly in the PDF stream:

- ink color is now genuinely white
- `Difference` *would* turn white-on-white into black and white-on-black into
  white, matching XOR semantics

…but MuPDF ignores `Difference` for TTF text, so what actually rasterizes is
plain white ink everywhere — invisible on white backgrounds. That regresses
labels that are mostly `^FR` text on white backgrounds (e.g. `glsdk_return`
goes 7.9% → 10.9%) more than it gains on the black-on-box cases.

We tried this on 2026-04-30 and reverted; the diff lived in
`internal/pdfdrawers/helpers.go` (added `setTextColor`) and
`internal/pdfdrawers/text_field.go` (called it alongside `setFillColor`).

## Possible real fixes (none implemented)

All of these are larger pieces of work than the rotation/position fixes that
make up the rest of the PDF parity effort:

1. **Emit glyphs as filled vector paths**, not `Tj` with a font resource.
   Sidesteps MuPDF's TTF-blend bug because the path operators apply
   `Difference` correctly. Requires running the TTF through a glyph-outline
   extractor (e.g. `golang.org/x/image/font/sfnt`) and emitting `m`/`l`/`c`/`f`
   per glyph. Loses font subsetting and text searchability in the PDF.
2. **Rasterize each `^FR` element to a 1-bit PNG and embed it.** Mirrors the
   PNG pipeline's separate-buffer + XOR approach. Faithful but loses vector
   crispness for reverse-printed text.
3. **Resolve the XOR at parse time.** When an `^FR` element overlaps a
   previously-drawn black region (`^GB`, image, dense barcode), pre-flip its
   ink color and drop the `Difference` blend. Requires intersection tests
   against earlier elements; brittle for partial overlaps.

Until one of these lands, expect any label that puts `^FR` text on a black
`^GB` box to show invisible / wrong-colour text in the PDF rasterization,
and the corresponding test case to never close to <1% mismatch on the
reverse-printed region.
