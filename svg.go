// Copyright 2012 - 2015 The ASCIIToSVG Contributors
// All rights reserved.

package asciitosvg

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"image"
	"io"
)

const (
	defaultFont = "Consolas,Monaco,Anonymous Pro,Anonymous,Bitstream Sans Mono,monospace"
	header      = "<!DOCTYPE svg PUBLIC \"-//W3C//DTD SVG 1.1//EN\" \"http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd\">\n"
	watermark   = "<!-- Created with ASCIItoSVG -->\n"
	svgTag      = "<svg width=\"%dpx\" height=\"%dpx\" version=\"1.1\" xmlns=\"http://www.w3.org/2000/svg\" xmlns:xlink=\"http://www.w3.org/1999/xlink\">\n"

	// Path related tag.
	pathTag       = "    <path id=\"%s%d\" %s%sd=\"%s\" />\n"
	pathMarkStart = "marker-start=\"url(#iPointer)\" "
	pathMarkEnd   = "marker-end=\"url(#Pointer)\" "

	// Text related tag.
	textGroupTag = "  <g id=\"text\" stroke=\"none\" style=\"font-family:%s;font-size:15.2px\" >\n"
	textTag      = "    <text id=\"obj%d\" x=\"%g\" y=\"%g\" fill=\"%s\">%s</text>\n"

	// TODO(dhobsd): Fine tune.
	blurDef = `  <defs>
    <filter id="dsFilter" width="150%%" height="150%%">
      <feOffset result="offOut" in="SourceGraphic" dx="2" dy="2"/>
      <feColorMatrix result="matrixOut" in="offOut" type="matrix" values="0.2 0 0 0 0 0 0.2 0 0 0 0 0 0.2 0 0 0 0 0 1 0"/>
      <feGaussianBlur result="blurOut" in="matrixOut" stdDeviation="3"/>
      <feBlend in="SourceGraphic" in2="blurOut" mode="normal"/>
    </filter>
    <marker id="iPointer"
      viewBox="0 0 10 10" refX="5" refY="5"
      markerUnits="strokeWidth"
      markerWidth="%g" markerHeight="%g"
      orient="auto">
      <path d="M 10 0 L 10 10 L 0 5 z" />
    </marker>
    <marker id="Pointer"
      viewBox="0 0 10 10" refX="5" refY="5"
      markerUnits="strokeWidth"
      markerWidth="%g" markerHeight="%g"
      orient="auto">
      <path d="M 0 0 L 10 5 L 0 10 z" />
    </marker>
  </defs>
`
)

func CanvasToSVG(c Canvas, noBlur bool, font string, scaleX, scaleY int) []byte {
	if len(font) == 0 {
		font = defaultFont
	}
	// TODO(dhobsd): Generating the XML manually is a tad fishy but encoding/xml
	// enforces standard XML header and the end code would be significantly
	// larger. The down side is potential escaping errors.
	b := &bytes.Buffer{}
	_, _ = io.WriteString(b, header)
	_, _ = io.WriteString(b, watermark)
	_, _ = fmt.Fprintf(b, svgTag, (c.Size().X+1)*scaleX, (c.Size().Y+1)*scaleY)
	x := float64(scaleX - 1)
	y := float64(scaleY - 1)
	_, _ = fmt.Fprintf(b, blurDef, x, y, x, y)

	// 3 passes, first closed paths, then open paths, then text.
	_, _ = io.WriteString(b, "  <g id=\"closed\" filter=\"url(#dsFilter)\" stroke=\"#000\" stroke-width=\"2\" fill=\"#88d\">\n")
	for i, obj := range c.Objects() {
		if obj.IsClosed() && !obj.IsText() {
			_, _ = fmt.Fprintf(b, pathTag, "closed", i, "", "", flatten(obj.Points(), scaleX, scaleY)+" Z")
		}
	}
	_, _ = io.WriteString(b, "  </g>\n")

	_, _ = io.WriteString(b, "  <g id=\"lines\" stroke=\"#000\" stroke-width=\"2\" fill=\"none\">\n")
	for i, obj := range c.Objects() {
		if !obj.IsClosed() && !obj.IsText() {
			text := obj.Text()
			start := ""
			if char(text[0]).isArrow() {
				start = pathMarkStart
			}
			end := ""
			if char(text[len(text)-1]).isArrow() {
				end = pathMarkEnd
			}
			_, _ = fmt.Fprintf(b, pathTag, "open", i, start, end, flatten(obj.Points(), scaleX, scaleY))
		}
	}
	_, _ = io.WriteString(b, "  </g>\n")

	_, _ = fmt.Fprintf(b, textGroupTag, escape(string(font)))
	for i, obj := range c.Objects() {
		if obj.IsText() {
			// If inside a box, make white, otherwise make black.
			color := "#000"
			topleft := obj.Points()[0]
			x := float64(topleft.X * scaleX)
			y := (float64(topleft.Y) + .75) * float64(scaleY)
			_, _ = fmt.Fprintf(b, textTag, i, x, y, color, escape(string(obj.Text())))
		}
	}
	_, _ = io.WriteString(b, "  </g>\n")

	_, _ = io.WriteString(b, "</svg>\n")
	return b.Bytes()
}

func escape(s string) string {
	b := &bytes.Buffer{}
	if err := xml.EscapeText(b, []byte(s)); err != nil {
		panic(err)
	}
	return b.String()
}

func flatten(points []image.Point, scaleX, scaleY int) string {
	out := ""
	for i, p := range points {
		cmd := "L"
		if i == 0 {
			cmd = "M"
		}
		sfx := " "
		if i == len(points)-1 {
			sfx = ""
		}
		out += fmt.Sprintf("%s %g %g%s", cmd, (float64(p.X)+.5)*float64(scaleX), (float64(p.Y)+.5)*float64(scaleY), sfx)
	}
	return out
}
