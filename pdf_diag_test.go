//go:build pdfraster

package zebrash

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/swiftpush/zebrash/drawers"
	"github.com/swiftpush/zebrash/elements"
)

// PDF-test diagnostics layered on top of TestDrawLabelAsPdf:
//
//   * 3-up composites are written by compareImagesTolerant in parser_pdf_test.go
//     for each failing case; recordPdfFailure here tracks them so we can emit
//     a single index page at the end of the run.
//
//   * TestPdfPerElementDiag (gated by -pdf-element-diag) re-runs every label
//     element through both pipelines in isolation and reports per-element
//     bbox deltas. This is the answer to "the percentage diff doesn't tell
//     me whether the content is wrong or just shifted" — element-level
//     bboxes localize the bug to a specific ZPL command and a numeric offset.

var (
	pdfElementDiag bool

	pdfFailuresMu sync.Mutex
	pdfFailures   []pdfFailureInfo
)

type pdfFailureInfo struct {
	name      string
	pct       float64
	composite string
}

func init() {
	flag.BoolVar(&pdfElementDiag, "pdf-element-diag", false,
		"Run TestPdfPerElementDiag — re-renders every element in isolation and reports per-element bbox deltas")
}

// TestMain runs after every PDF test and writes a sorted markdown index of
// failures to testdata/diff/_pdf_index.md. The index inlines each composite
// thumbnail, so a single Read of the index gives a full overview of what's
// broken.
func TestMain(m *testing.M) {
	code := m.Run()
	writePdfFailureIndex()
	os.Exit(code)
}

func recordPdfFailure(info pdfFailureInfo) {
	pdfFailuresMu.Lock()
	defer pdfFailuresMu.Unlock()
	pdfFailures = append(pdfFailures, info)
}

func writePdfFailureIndex() {
	pdfFailuresMu.Lock()
	defer pdfFailuresMu.Unlock()

	indexPath := "./testdata/diff/_pdf_index.md"

	if len(pdfFailures) == 0 {
		_ = os.Remove(indexPath)
		return
	}

	sort.Slice(pdfFailures, func(i, j int) bool {
		return pdfFailures[i].pct > pdfFailures[j].pct
	})

	var b strings.Builder
	fmt.Fprintf(&b, "# PDF backend failures (%d)\n\n", len(pdfFailures))
	b.WriteString("Sorted by mismatch %. Each composite is **want | got | colored diff**.\n\n")
	b.WriteString("Diff colors: **red** = pixel is darker in `got` than in `want` (we drew extra ink); **green** = pixel is lighter in `got` than in `want` (ink is missing).\n\n")
	for _, f := range pdfFailures {
		fmt.Fprintf(&b, "## %s — %.3f%%\n\n", f.name, f.pct)
		if f.composite == "" {
			b.WriteString("_(diff writing was disabled with -write-image-diff=false)_\n\n")
			continue
		}
		rel, err := filepath.Rel(filepath.Dir(indexPath), f.composite)
		if err != nil {
			rel = f.composite
		}
		fmt.Fprintf(&b, "![%s](%s)\n\n", f.name, rel)
	}
	if err := os.WriteFile(indexPath, []byte(b.String()), 0644); err != nil {
		fmt.Fprintln(os.Stderr, "failed to write pdf index:", err)
	}
}

// TestPdfPerElementDiag is the structural ("right thing in wrong place")
// diagnostic. It does not assert and does not contribute to CI signal — it
// only runs with `-pdf-element-diag` and is for manual investigation.
//
// For every element of every fixture it:
//
//  1. builds a single-element LabelInfo,
//  2. renders it via the PNG pipeline (golden ground truth) and via the PDF
//     pipeline (rasterized through MuPDF),
//  3. computes the dark-pixel bbox in each output, and
//  4. writes a per-fixture report with element type, ZPL position (when the
//     element exposes one via reflection), bboxes, and the (dx, dy) delta.
//
// Output goes to testdata/diff/_pdf_elements/<test>.txt. Empty bboxes (e.g.
// non-visual elements like RecalledFormat) are skipped.
func TestPdfPerElementDiag(t *testing.T) {
	if !pdfElementDiag {
		t.Skip("re-run with -pdf-element-diag to enable per-element bbox diagnostics")
	}

	outDir := "./testdata/diff/_pdf_elements"
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	for _, tC := range drawTestCases {
		if strings.Contains(tC.srcPath, "reverse") ||
			strings.Contains(tC.srcPath, "custom_ttf") ||
			tC.grayscaleOutput {
			continue
		}

		t.Run(tC.name, func(t *testing.T) {
			file := mustReadFile("./testdata/"+tC.srcPath, t)
			parser := NewParser()
			res, err := parser.Parse(file)
			if err != nil {
				t.Fatal(err)
			}
			if len(res) == 0 {
				t.Fatal("no labels parsed")
			}
			label := res[tC.labelIdx]

			opts := drawers.DrawerOptions{
				LabelWidthMm:         tC.widthMm,
				LabelHeightMm:        tC.heightMm,
				EnableInvertedLabels: tC.enableInverted,
			}

			var report strings.Builder
			fmt.Fprintf(&report, "fixture: %s\n", tC.srcPath)
			fmt.Fprintf(&report, "elements: %d\n\n", len(label.Elements))
			fmt.Fprintf(&report, "%-3s %-28s %-22s %-26s %-26s %s\n",
				"#", "type", "zpl_pos", "png_bbox", "pdf_bbox", "delta(dx,dy,dw,dh)")
			fmt.Fprintln(&report, strings.Repeat("-", 140))

			for i, el := range label.Elements {
				single := elements.LabelInfo{
					PrintWidth: label.PrintWidth,
					Inverted:   label.Inverted,
					Elements:   []any{el},
				}

				pngBuf := new(bytes.Buffer)
				if err := NewDrawer().DrawLabelAsPng(single, pngBuf, opts); err != nil {
					fmt.Fprintf(&report, "%-3d %-28s ERR png: %v\n", i, typeName(el), err)
					continue
				}
				pngImg := mustDecodePng(pngBuf.Bytes(), t)

				pdfBuf := new(bytes.Buffer)
				if err := NewDrawer().DrawLabelAsPdf(single, pdfBuf, opts); err != nil {
					fmt.Fprintf(&report, "%-3d %-28s ERR pdf: %v\n", i, typeName(el), err)
					continue
				}
				pdfImg := rasterizePdf(pdfBuf.Bytes(), t)

				pngBox := darkBBox(pngImg)
				pdfBox := darkBBox(pdfImg)

				if pngBox == nil && pdfBox == nil {
					// non-visual element — skip rather than spam the report
					continue
				}

				fmt.Fprintf(&report, "%-3d %-28s %-22s %-26s %-26s %s\n",
					i, typeName(el), zplPos(el),
					bboxStr(pngBox), bboxStr(pdfBox), deltaStr(pngBox, pdfBox))
			}

			outPath := filepath.Join(outDir, sanitizeName(t.Name())+".txt")
			if err := os.WriteFile(outPath, []byte(report.String()), 0644); err != nil {
				t.Fatalf("write report: %v", err)
			}
			t.Logf("element diag → %s", outPath)
		})
	}
}

// darkBBox returns the bounding box of pixels darker than 200 (out of 255),
// in image coordinates. Returns nil if the image has no dark pixels.
func darkBBox(img image.Image) *image.Rectangle {
	gray, ok := img.(*image.Gray)
	if !ok {
		return nil
	}
	bounds := gray.Bounds()
	minX, minY := bounds.Max.X, bounds.Max.Y
	maxX, maxY := bounds.Min.X-1, bounds.Min.Y-1
	found := false
	const darkThreshold = 200
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if gray.GrayAt(x, y).Y < darkThreshold {
				found = true
				if x < minX {
					minX = x
				}
				if y < minY {
					minY = y
				}
				if x > maxX {
					maxX = x
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}
	if !found {
		return nil
	}
	r := image.Rect(minX, minY, maxX+1, maxY+1)
	return &r
}

func bboxStr(r *image.Rectangle) string {
	if r == nil {
		return "(empty)"
	}
	return fmt.Sprintf("(%d,%d %dx%d)", r.Min.X, r.Min.Y, r.Dx(), r.Dy())
}

// deltaStr returns the per-axis offset and size delta of png vs pdf bboxes.
// dx>0 means PDF drew to the right of PNG; dy>0 means PDF drew below PNG.
// dw / dh > 0 means PDF rendered larger.
func deltaStr(png, pdf *image.Rectangle) string {
	if png == nil || pdf == nil {
		return "(one bbox empty)"
	}
	dx := pdf.Min.X - png.Min.X
	dy := pdf.Min.Y - png.Min.Y
	dw := pdf.Dx() - png.Dx()
	dh := pdf.Dy() - png.Dy()
	return fmt.Sprintf("(%+d,%+d, %+d,%+d)", dx, dy, dw, dh)
}

func typeName(el any) string {
	t := reflect.TypeOf(el)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.String()
}

// zplPos uses reflection to fish out a Position field from elements that
// have one (most do via LabelPosition). Returns "-" if the element has no
// .Position.X / .Position.Y pair.
func zplPos(el any) string {
	v := reflect.ValueOf(el)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return "-"
	}
	pos := v.FieldByName("Position")
	if !pos.IsValid() || pos.Kind() != reflect.Struct {
		return "-"
	}
	x := pos.FieldByName("X")
	y := pos.FieldByName("Y")
	if !x.IsValid() || !y.IsValid() {
		return "-"
	}
	return fmt.Sprintf("(%d,%d)", x.Int(), y.Int())
}

func sanitizeName(s string) string {
	repl := strings.NewReplacer("/", "_", " ", "_", "'", "")
	return repl.Replace(s)
}
