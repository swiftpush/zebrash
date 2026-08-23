//go:build svgraster

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
	"github.com/swiftpush/zebrash/internal/svgraster"
)

// SVG-test diagnostics layered on top of TestDrawLabelAsSvg. Mirrors the PDF
// harness's structure (drawer_pdf_test.go / drawer_pdf_diag_test.go):
//
//   * compareImagesTolerantSvg in drawer_svg_test.go writes a 3-up composite
//     for each failing case and feeds recordSvgFailure here.
//
//   * TestSvgPerElementDiag (gated by -svg-element-diag) re-renders every
//     label element in isolation through both pipelines and reports
//     per-element bbox deltas — same diagnostic shape as TestPdfPerElementDiag.

var (
	svgElementDiag bool

	svgFailuresMu sync.Mutex
	svgFailures   []svgFailureInfo
)

type svgFailureInfo struct {
	name      string
	pct       float64
	composite string
}

func init() {
	flag.BoolVar(&svgElementDiag, "svg-element-diag", false,
		"Run TestSvgPerElementDiag — re-renders every element in isolation and reports per-element bbox deltas")
}

// TestMain is defined here (rather than in drawer_svg_test.go) so the SVG
// build tag pulls in a single TestMain. The PDF harness has its own TestMain
// behind the `pdfraster` tag — Go's build constraints keep them from
// colliding because they're never compiled together.
func TestMain(m *testing.M) {
	code := m.Run()
	writeSvgFailureIndex()
	os.Exit(code)
}

func recordSvgFailure(info svgFailureInfo) {
	svgFailuresMu.Lock()
	defer svgFailuresMu.Unlock()
	svgFailures = append(svgFailures, info)
}

func writeSvgFailureIndex() {
	svgFailuresMu.Lock()
	defer svgFailuresMu.Unlock()

	indexPath := "./testdata/diff/_svg_index.md"

	if len(svgFailures) == 0 {
		_ = os.Remove(indexPath)
		return
	}

	sort.Slice(svgFailures, func(i, j int) bool {
		return svgFailures[i].pct > svgFailures[j].pct
	})

	var b strings.Builder
	fmt.Fprintf(&b, "# SVG backend failures (%d)\n\n", len(svgFailures))
	b.WriteString("Sorted by mismatch %. Each composite is **want | got | colored diff**.\n\n")
	b.WriteString("Diff colors: **red** = pixel is darker in `got` than in `want` (we drew extra ink); **green** = pixel is lighter in `got` than in `want` (ink is missing).\n\n")
	for _, f := range svgFailures {
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
		fmt.Fprintln(os.Stderr, "failed to write svg index:", err)
	}
}

// TestSvgPerElementDiag is the structural diagnostic — same shape as
// TestPdfPerElementDiag. Does not assert; only runs with -svg-element-diag.
//
// For every element of every fixture it builds a single-element LabelInfo,
// renders it via the PNG pipeline (ground truth) and the SVG pipeline
// (rasterized through resvg), computes the dark-pixel bbox in each, and
// writes a per-fixture report to testdata/diff/_svg_elements/<test>.txt.
func TestSvgPerElementDiag(t *testing.T) {
	if !svgElementDiag {
		t.Skip("re-run with -svg-element-diag to enable per-element bbox diagnostics")
	}

	const dpi = 203.0
	outDir := "./testdata/diff/_svg_elements"
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
				"#", "type", "zpl_pos", "png_bbox", "svg_bbox", "delta(dx,dy,dw,dh)")
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

				svgBuf := new(bytes.Buffer)
				if err := NewDrawer().DrawLabelAsSvg(single, svgBuf, opts); err != nil {
					fmt.Fprintf(&report, "%-3d %-28s ERR svg: %v\n", i, typeName(el), err)
					continue
				}
				svgImg, err := svgraster.RasterizeSVG(svgBuf.Bytes(), dpi)
				if err != nil {
					fmt.Fprintf(&report, "%-3d %-28s ERR rasterize: %v\n", i, typeName(el), err)
					continue
				}

				pngBox := darkBBoxSvg(pngImg)
				svgBox := darkBBoxSvg(svgImg)

				if pngBox == nil && svgBox == nil {
					continue
				}

				fmt.Fprintf(&report, "%-3d %-28s %-22s %-26s %-26s %s\n",
					i, typeName(el), zplPos(el),
					bboxStrSvg(pngBox), bboxStrSvg(svgBox), deltaStrSvg(pngBox, svgBox))
			}

			outPath := filepath.Join(outDir, sanitizeName(t.Name())+".txt")
			if err := os.WriteFile(outPath, []byte(report.String()), 0644); err != nil {
				t.Fatalf("write report: %v", err)
			}
			t.Logf("element diag → %s", outPath)
		})
	}
}

// darkBBoxSvg / bboxStrSvg / deltaStrSvg / typeName / zplPos / sanitizeName
// are direct copies of the PDF-side helpers — kept separate so the two
// build-tagged files don't try to share symbols.

func darkBBoxSvg(img image.Image) *image.Rectangle {
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

func bboxStrSvg(r *image.Rectangle) string {
	if r == nil {
		return "(empty)"
	}
	return fmt.Sprintf("(%d,%d %dx%d)", r.Min.X, r.Min.Y, r.Dx(), r.Dy())
}

func deltaStrSvg(png, svg *image.Rectangle) string {
	if png == nil || svg == nil {
		return "(one bbox empty)"
	}
	dx := svg.Min.X - png.Min.X
	dy := svg.Min.Y - png.Min.Y
	dw := svg.Dx() - png.Dx()
	dh := svg.Dy() - png.Dy()
	return fmt.Sprintf("(%+d,%+d, %+d,%+d)", dx, dy, dw, dh)
}

func typeName(el any) string {
	t := reflect.TypeOf(el)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.String()
}

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
