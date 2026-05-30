// Package svgwriter is a tiny streaming writer for the SVG subset the
// zebrash SVG drawers emit. It's intentionally not a full SVG DOM — drawers
// call imperative methods (Rect, Circle, Text, etc.) and the package writes
// them straight to the underlying io.Writer.
//
// All coordinates are written as user-units in the SVG; the caller picks the
// unit by passing the width/height to New (the SVG drawer uses millimeters,
// matching the PDF backend).
package svgwriter

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// Doc is a single in-progress SVG document. It is not safe for concurrent use.
type Doc struct {
	w           io.Writer
	openGroups  int
	rootClosed  bool
	defsClosed  bool
	imageNameCt int
}

// New starts an SVG document with width and height expressed in millimeters
// and a matching viewBox so that user units = millimeters.
func New(w io.Writer, widthMm, heightMm float64) *Doc {
	d := &Doc{w: w}
	fmt.Fprintf(w,
		`<?xml version="1.0" encoding="UTF-8"?>`+
			`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" `+
			`width="%[1]gmm" height="%[2]gmm" viewBox="0 0 %[1]g %[2]g">`,
		widthMm, heightMm)
	return d
}

// Background writes a full-document white background rect. Call it once
// directly after New; this matches the white-paper backdrop the PNG and PDF
// pipelines assume.
func (d *Doc) Background(widthMm, heightMm float64) {
	fmt.Fprintf(d.w, `<rect x="0" y="0" width="%g" height="%g" fill="white"/>`, widthMm, heightMm)
}

// Defs opens a <defs> block, calls emit, then closes it. Use this for
// @font-face rules and reusable <filter> definitions.
func (d *Doc) Defs(emit func(*Doc)) {
	io.WriteString(d.w, `<defs>`)
	emit(d)
	io.WriteString(d.w, `</defs>`)
	d.defsClosed = true
}

// Style emits a raw <style> block; the caller is responsible for the CSS body.
// Used for @font-face rules which need CSS syntax rather than SVG attributes.
func (d *Doc) Style(css string) {
	io.WriteString(d.w, `<style type="text/css"><![CDATA[`)
	io.WriteString(d.w, css)
	io.WriteString(d.w, `]]></style>`)
}

// FontFace emits an @font-face CSS rule embedding a TrueType font as a
// base64 data: URL. Must be called inside Defs (or written into Style by the
// caller). weight is the CSS font-weight to bind the face to ("normal" /
// "bold" / ""); pass "" to omit the weight descriptor.
func (d *Doc) FontFace(family, weight, mime string, ttf []byte) {
	enc := base64.StdEncoding.EncodeToString(ttf)
	weightCss := ""
	if weight != "" {
		weightCss = fmt.Sprintf(`font-weight:%s;`, weight)
	}
	css := fmt.Sprintf(`@font-face{font-family:%q;%ssrc:url("data:%s;base64,%s") format("truetype");}`,
		family, weightCss, mime, enc)
	d.Style(css)
}

// Filter emits a <filter id=...> wrapping the caller-provided inner XML.
// Used for reverse-print (XOR) implementation.
func (d *Doc) Filter(id, innerXML string) {
	fmt.Fprintf(d.w, `<filter id=%q x="-5%%" y="-5%%" width="110%%" height="110%%">%s</filter>`,
		id, innerXML)
}

// GroupTransform opens a <g transform="..."> element. Must be balanced with
// EndGroup.
func (d *Doc) GroupTransform(transform string) {
	fmt.Fprintf(d.w, `<g transform=%q>`, transform)
	d.openGroups++
}

// GroupFilter opens a <g filter="url(#id)"> element. Used to apply a
// previously-defined filter (eg reverse-print) to a span of child elements.
func (d *Doc) GroupFilter(filterID string) {
	fmt.Fprintf(d.w, `<g filter="url(#%s)">`, filterID)
	d.openGroups++
}

// EndGroup closes the most recent group opened via GroupTransform/GroupFilter.
func (d *Doc) EndGroup() {
	if d.openGroups == 0 {
		return
	}
	io.WriteString(d.w, `</g>`)
	d.openGroups--
}

// Rect draws a filled or stroked axis-aligned rectangle. Pass an empty fill
// for stroke-only output and stroke="" with strokeWidth=0 for fill-only.
func (d *Doc) Rect(x, y, w, h float64, fill, stroke string, strokeWidth float64) {
	fmt.Fprintf(d.w, `<rect x="%g" y="%g" width="%g" height="%g"`, x, y, w, h)
	d.writePaint(fill, stroke, strokeWidth)
	io.WriteString(d.w, `/>`)
}

// RoundedRect is Rect with a corner radius.
func (d *Doc) RoundedRect(x, y, w, h, r float64, fill, stroke string, strokeWidth float64) {
	fmt.Fprintf(d.w, `<rect x="%g" y="%g" width="%g" height="%g" rx="%g" ry="%g"`, x, y, w, h, r, r)
	d.writePaint(fill, stroke, strokeWidth)
	io.WriteString(d.w, `/>`)
}

// Circle draws a circle centered at (cx, cy) with radius r.
func (d *Doc) Circle(cx, cy, r float64, fill, stroke string, strokeWidth float64) {
	fmt.Fprintf(d.w, `<circle cx="%g" cy="%g" r="%g"`, cx, cy, r)
	d.writePaint(fill, stroke, strokeWidth)
	io.WriteString(d.w, `/>`)
}

// Line draws a straight stroked line between two points.
func (d *Doc) Line(x1, y1, x2, y2 float64, stroke string, strokeWidth float64) {
	fmt.Fprintf(d.w, `<line x1="%g" y1="%g" x2="%g" y2="%g"`, x1, y1, x2, y2)
	d.writePaint("none", stroke, strokeWidth)
	io.WriteString(d.w, `/>`)
}

// Polygon draws a closed filled polygon from a list of (x, y) vertices.
func (d *Doc) Polygon(points [][2]float64, fill, stroke string, strokeWidth float64) {
	var b strings.Builder
	for i, p := range points {
		if i > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprintf(&b, "%g,%g", p[0], p[1])
	}
	fmt.Fprintf(d.w, `<polygon points=%q`, b.String())
	d.writePaint(fill, stroke, strokeWidth)
	io.WriteString(d.w, `/>`)
}

// Text emits a single <text> element. y is the baseline (matching gg's
// DrawStringAnchored ay=0 and fpdf.Text behaviour). anchor maps to SVG's
// text-anchor: "start" / "middle" / "end". weight maps to font-weight
// ("normal" / "bold" / ...) — only emitted if non-empty and not "normal".
func (d *Doc) Text(x, y float64, fontFamily, fontWeight string, fontSizePt, fontSizeMm float64, fill, anchor, s string) {
	// font-size is intentionally emitted WITHOUT a unit. resvg (and several
	// other SVG renderers) take a `…mm` font-size, multiply it by 96/25.4 to
	// reach pixels, and then interpret those pixels in user-space — which in
	// our viewBox is already millimeters, so the glyph ends up ~3.78x too
	// large. Unitless values are interpreted directly as user units (mm),
	// which makes the rendered glyph match the dot-space height the PNG/PDF
	// pipelines compute.
	_ = fontSizePt
	fmt.Fprintf(d.w, `<text x="%g" y="%g" font-family=%q font-size="%g" fill=%q`,
		x, y, fontFamily, fontSizeMm, fill)
	if fontWeight != "" && fontWeight != "normal" {
		fmt.Fprintf(d.w, ` font-weight=%q`, fontWeight)
	}
	if anchor != "" && anchor != "start" {
		fmt.Fprintf(d.w, ` text-anchor=%q`, anchor)
	}
	io.WriteString(d.w, `>`)
	xml.EscapeText(d.w, []byte(s))
	io.WriteString(d.w, `</text>`)
}

// Image embeds an image (PNG bytes) as a base64 data URL at (x, y) with the
// given on-page width/height.
func (d *Doc) Image(x, y, w, h float64, pngBytes []byte) {
	enc := base64.StdEncoding.EncodeToString(pngBytes)
	fmt.Fprintf(d.w,
		`<image x="%g" y="%g" width="%g" height="%g" preserveAspectRatio="none" xlink:href="data:image/png;base64,%s"/>`,
		x, y, w, h, enc)
}

// Close writes the closing </svg>. Any still-open groups are closed first so
// that callers that forget to balance a group still produce valid SVG.
func (d *Doc) Close() error {
	for d.openGroups > 0 {
		io.WriteString(d.w, `</g>`)
		d.openGroups--
	}
	if !d.rootClosed {
		_, err := io.WriteString(d.w, `</svg>`)
		d.rootClosed = true
		return err
	}
	return nil
}

func (d *Doc) writePaint(fill, stroke string, strokeWidth float64) {
	if fill == "" {
		io.WriteString(d.w, ` fill="none"`)
	} else {
		fmt.Fprintf(d.w, ` fill=%q`, fill)
	}
	if stroke != "" {
		fmt.Fprintf(d.w, ` stroke=%q stroke-width="%g"`, stroke, strokeWidth)
	}
}
