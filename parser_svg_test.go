//go:build svgraster

package zebrash

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"strings"
	"testing"

	"github.com/ingridhq/zebrash/drawers"
	"github.com/ingridhq/zebrash/internal/svgraster"
)

// TestDrawLabelAsSvg renders every fixture in drawTestCases through the SVG
// backend, rasterizes the result via resvg (WASM, no cgo), and compares it
// against the existing PNG golden with a tolerant pixel diff.
//
// Build tag: enabled with `go test -tags=svgraster -run TestDrawLabelAsSvg`.
// The SVG backend is younger than the PDF one — expect many fixtures to
// fail v1; the harness is here to give actionable diff signal, not to gate
// on perfection. See docs/svg-backend.md.
func TestDrawLabelAsSvg(t *testing.T) {
	const (
		pixelDeltaTolerance = 16
		mismatchPercentCap  = 1.0
	)

	for _, tC := range drawTestCases {
		// v1 skips match the PDF harness exactly. Once the reverse-print
		// XOR filter is verified through resvg, lift the "reverse" skip.
		if strings.Contains(tC.srcPath, "reverse") ||
			strings.Contains(tC.srcPath, "custom_ttf") ||
			tC.grayscaleOutput {
			continue
		}

		t.Run(tC.name, func(t *testing.T) {
			fullSrcPath := "./testdata/" + tC.srcPath
			fullDstPath := "./testdata/" + tC.dstPath
			baseName := strings.TrimSuffix(tC.dstPath, ".png")
			fullDiffPath := "./testdata/diff/" + baseName + "_svg.png"
			fullSvgPath := "./testdata/svg/" + baseName + ".svg"

			file := mustReadFile(fullSrcPath, t)
			parser := NewParser()
			res, err := parser.Parse(file)
			if err != nil {
				t.Fatal(err)
			}
			if len(res) == 0 {
				t.Fatal("no labels in the response")
			}

			drawer := NewDrawer()
			svgBuf := new(bytes.Buffer)
			err = drawer.DrawLabelAsSvg(res[tC.labelIdx], svgBuf, drawers.DrawerOptions{
				LabelWidthMm:         tC.widthMm,
				LabelHeightMm:        tC.heightMm,
				EnableInvertedLabels: tC.enableInverted,
			})
			if err != nil {
				t.Fatal(err)
			}

			mustWriteFile(fullSvgPath, svgBuf.Bytes(), t)

			gotImg, err := svgraster.RasterizeSVG(svgBuf.Bytes(), 203.0)
			if err != nil {
				t.Fatalf("rasterize svg: %v", err)
			}
			wantImg := mustDecodePng(mustReadFile(fullDstPath, t), t)

			compareImagesTolerantSvg(gotImg, wantImg, fullDiffPath, pixelDeltaTolerance, mismatchPercentCap, t)
		})
	}
}

// compareImagesTolerantSvg is the parser_pdf_test.go's compareImagesTolerant
// with `_svg` naming. Kept as a near-copy to avoid coupling the two harnesses
// — they're free to diverge as the SVG backend matures.
func compareImagesTolerantSvg(got, want image.Image, fullDiffPath string, pixelDelta int, mismatchPct float64, t *testing.T) {
	gotBounds := got.Bounds()
	wantBounds := want.Bounds()

	if math.Abs(float64(gotBounds.Dx()-wantBounds.Dx())) > 4 ||
		math.Abs(float64(gotBounds.Dy()-wantBounds.Dy())) > 4 {
		t.Fatalf("Image bounds differ beyond DPI rounding: got=%v want=%v", gotBounds, wantBounds)
	}

	width := minIntSvg(gotBounds.Dx(), wantBounds.Dx())
	height := minIntSvg(gotBounds.Dy(), wantBounds.Dy())

	gotGray, ok := got.(*image.Gray)
	if !ok {
		t.Fatalf("got is not grayscale image")
	}
	wantGray, ok := want.(*image.Gray)
	if !ok {
		t.Fatalf("want is not grayscale image")
	}

	diffImg := image.NewRGBA(image.Rect(0, 0, width, height))
	mismatched := 0
	total := width * height

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			gv := gotGray.GrayAt(x, y).Y
			wv := wantGray.GrayAt(x, y).Y

			delta := int(gv) - int(wv)
			if delta < 0 {
				delta = -delta
			}

			if delta <= pixelDelta {
				diffImg.Set(x, y, color.RGBA{R: gv, G: gv, B: gv, A: 255})
				continue
			}

			mismatched++
			if wv > gv {
				diffImg.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
			} else {
				diffImg.Set(x, y, color.RGBA{R: 0, G: 255, B: 0, A: 255})
			}
		}
	}

	pct := 100.0 * float64(mismatched) / float64(total)
	if pct <= mismatchPct {
		return
	}

	t.Errorf("SVG rasterization differs from PNG golden: %.3f%% mismatched (cap %.3f%%)", pct, mismatchPct)

	if !writeImageDiff {
		recordSvgFailure(svgFailureInfo{name: t.Name(), pct: pct, composite: ""})
		return
	}

	composite := buildSvgComposite(wantGray, gotGray, diffImg, width, height)

	buff := new(bytes.Buffer)
	if err := png.Encode(buff, composite); err != nil {
		t.Fatalf("Failed to encode diff image: %v", err)
	}
	mustWriteFile(fullDiffPath, buff.Bytes(), t)
	t.Logf("Composite (want|got|diff) saved to %s", fullDiffPath)

	recordSvgFailure(svgFailureInfo{name: t.Name(), pct: pct, composite: fullDiffPath})
}

// buildSvgComposite mirrors parser_pdf_test.go's buildPdfComposite. Layout:
// [ want | got | colored diff ], 4-pixel black separators.
func buildSvgComposite(want, got *image.Gray, diff *image.RGBA, width, height int) *image.RGBA {
	const sep = 4
	totalW := width*3 + sep*2
	totalH := height

	out := image.NewRGBA(image.Rect(0, 0, totalW, totalH))
	for y := 0; y < totalH; y++ {
		for x := 0; x < totalW; x++ {
			out.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	black := color.RGBA{A: 255}
	for y := 0; y < totalH; y++ {
		for s := 0; s < sep; s++ {
			out.Set(width+s, y, black)
			out.Set(width*2+sep+s, y, black)
		}
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			wv := want.GrayAt(x, y).Y
			out.Set(x, y, color.RGBA{R: wv, G: wv, B: wv, A: 255})

			gv := got.GrayAt(x, y).Y
			out.Set(width+sep+x, y, color.RGBA{R: gv, G: gv, B: gv, A: 255})

			out.Set(width*2+sep*2+x, y, diff.RGBAAt(x, y))
		}
	}
	return out
}

func minIntSvg(a, b int) int {
	if a < b {
		return a
	}
	return b
}
