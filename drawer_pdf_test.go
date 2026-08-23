//go:build pdfraster

package zebrash

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"strings"
	"testing"

	"github.com/gen2brain/go-fitz"

	"github.com/swiftpush/zebrash/drawers"
)

// TestDrawLabelAsPdf renders every fixture in drawTestCases through the PDF
// backend, rasterizes the result via MuPDF (go-fitz, cgo), and compares it
// against the existing PNG golden with a tolerant pixel diff.
//
// Build tag: enabled with `go test -tags=pdfraster ./...`. Default test runs
// skip this so contributors don't need MuPDF system libs to validate the
// raster pipeline.
//
// Tolerance: anti-aliasing differences between gg and MuPDF and sub-pixel bar
// boundaries mean exact pixel match isn't achievable. The current thresholds
// (delta ≤ 16 per channel, mismatch ≤ 1.0%) are starting points and likely
// need tuning per-fixture as we iterate.
func TestDrawLabelAsPdf(t *testing.T) {
	const (
		pixelDeltaTolerance = 16
		mismatchPercentCap  = 1.0
	)

	for _, tC := range drawTestCases {
		// v1 PDF backend known limitations:
		//   - reverse-print uses a Difference blend mode that round-trips
		//     differently than the raster XOR for AA edges.
		//   - custom TTF (^DU) fonts aren't embedded in PDF v1.
		//   - grayscale-output mode is meaningless for vector PDF.
		if strings.Contains(tC.srcPath, "reverse") ||
			strings.Contains(tC.srcPath, "custom_ttf") ||
			tC.grayscaleOutput {
			continue
		}

		t.Run(tC.name, func(t *testing.T) {
			fullSrcPath := "./testdata/" + tC.srcPath
			fullDstPath := "./testdata/" + tC.dstPath
			baseName := strings.TrimSuffix(tC.dstPath, ".png")
			fullDiffPath := "./testdata/diff/" + baseName + "_pdf.png"
			fullPdfPath := "./testdata/pdf/" + baseName + ".pdf"

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
			pdfBuf := new(bytes.Buffer)
			err = drawer.DrawLabelAsPdf(res[tC.labelIdx], pdfBuf, drawers.DrawerOptions{
				LabelWidthMm:         tC.widthMm,
				LabelHeightMm:        tC.heightMm,
				EnableInvertedLabels: tC.enableInverted,
			})
			if err != nil {
				t.Fatal(err)
			}

			mustWriteFile(fullPdfPath, pdfBuf.Bytes(), t)

			gotImg := rasterizePdf(pdfBuf.Bytes(), t)
			wantImg := mustDecodePng(mustReadFile(fullDstPath, t), t)

			compareImagesTolerant(gotImg, wantImg, fullDiffPath, pixelDeltaTolerance, mismatchPercentCap, t)
		})
	}
}

// rasterizePdf turns a single-page PDF into a grayscale image at 203 dpi —
// the same density the raster pipeline targets by default — using MuPDF.
func rasterizePdf(pdfBytes []byte, t *testing.T) image.Image {
	doc, err := fitz.NewFromMemory(pdfBytes)
	if err != nil {
		t.Fatalf("failed to open generated pdf: %v", err)
	}
	defer doc.Close()

	if doc.NumPage() == 0 {
		t.Fatal("generated pdf has no pages")
	}

	const dpi = 203.0
	img, err := doc.ImageDPI(0, dpi)
	if err != nil {
		t.Fatalf("failed to rasterize pdf: %v", err)
	}

	bounds := img.Bounds()
	gray := image.NewGray(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			lum := uint8((299*int(r>>8) + 587*int(g>>8) + 114*int(b>>8)) / 1000)
			gray.SetGray(x, y, color.Gray{Y: lum})
		}
	}
	return gray
}

// compareImagesTolerant is the PNG harness's compareImages with a per-pixel
// delta threshold and a mismatch-percentage cap on top.
//
// On failure, writes a 3-up composite (want | got | colored-diff) to
// fullDiffPath. The composite is the harness's primary debugging signal —
// it makes "right shape, wrong place" bugs visually obvious in a single
// image, instead of forcing the reader to mentally overlay three files.
func compareImagesTolerant(got, want image.Image, fullDiffPath string, pixelDelta int, mismatchPct float64, t *testing.T) {
	gotBounds := got.Bounds()
	wantBounds := want.Bounds()

	// Allow up to a few pixels of bounds drift — go-fitz's DPI rounding
	// can produce off-by-one rasters compared to the integer-pixel raster.
	if math.Abs(float64(gotBounds.Dx()-wantBounds.Dx())) > 4 ||
		math.Abs(float64(gotBounds.Dy()-wantBounds.Dy())) > 4 {
		t.Fatalf("Image bounds differ beyond DPI rounding: got=%v want=%v", gotBounds, wantBounds)
	}

	width := minInt(gotBounds.Dx(), wantBounds.Dx())
	height := minInt(gotBounds.Dy(), wantBounds.Dy())

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

	const maxReportedSampleMismatches = 5
	var sampleMismatches []string

	for y := range height {
		for x := range width {
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
			if len(sampleMismatches) < maxReportedSampleMismatches {
				sampleMismatches = append(sampleMismatches, fmt.Sprintf("(x: %d, y: %d): got=GRAY(%d) want=GRAY(%d)", x, y, gv, wv))
			}

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

	t.Errorf("PDF rasterization differs from PNG golden: %.3f%% mismatched (cap %.3f%%)", pct, mismatchPct)
	t.Errorf("First %d sample mismatches: \n%s", maxReportedSampleMismatches, strings.Join(sampleMismatches, "\n"))

	if !writeImageDiff {
		recordPdfFailure(pdfFailureInfo{name: t.Name(), pct: pct, composite: ""})
		return
	}

	composite := buildPdfComposite(wantGray, gotGray, diffImg, width, height)

	buff := new(bytes.Buffer)
	if err := png.Encode(buff, composite); err != nil {
		t.Fatalf("Failed to encode diff image: %v", err)
	}
	mustWriteFile(fullDiffPath, buff.Bytes(), t)
	t.Logf("Composite (want|got|diff) saved to %s", fullDiffPath)

	recordPdfFailure(pdfFailureInfo{name: t.Name(), pct: pct, composite: fullDiffPath})
}

// buildPdfComposite lays out three images horizontally with 4-pixel black
// separators: golden (want), rasterized PDF (got), colored diff. Sized to
// max(want.h, got.h, diff.h) so jagged DPI rounding doesn't crop content.
func buildPdfComposite(want, got *image.Gray, diff *image.RGBA, width, height int) *image.RGBA {
	const sep = 4
	totalW := width*3 + sep*2
	totalH := height

	out := image.NewRGBA(image.Rect(0, 0, totalW, totalH))
	// White background.
	for y := range totalH {
		for x := range totalW {
			out.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	// Black separators.
	black := color.RGBA{A: 255}
	for y := range totalH {
		for s := range sep {
			out.Set(width+s, y, black)
			out.Set(width*2+sep+s, y, black)
		}
	}

	for y := range height {
		for x := range width {
			wv := want.GrayAt(x, y).Y
			out.Set(x, y, color.RGBA{R: wv, G: wv, B: wv, A: 255})

			gv := got.GrayAt(x, y).Y
			out.Set(width+sep+x, y, color.RGBA{R: gv, G: gv, B: gv, A: 255})

			out.Set(width*2+sep*2+x, y, diff.RGBAAt(x, y))
		}
	}
	return out
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
