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
	pathTag     = "    <path id=\"%s%d\" d=\"%s\" />\n"

	textGroupTag = "  <g id=\"text\" fill=\"#000\" stroke=\"none\" style=\"font-family:%s;font-size:15.2px\" >\n"
	textShortTag = "    <text x=\"%g\" y=\"%g\" id=\"obj%d\">%s</text>\n"
	textLongTag  = "    <text x=\"%g\" y=\"%g\" id=\"obj%d\" fill=\"#000\" stroke=\"none\" style=\"font-family:%s;font-size:15.2px\">%s</text>\n"
)

func CanvasToSVG(c Canvas, noBlur bool, font string, scaleX, scaleY int) []byte {
	if len(font) == 0 {
		font = defaultFont
	}
	// TODO(maruel): Generating the XML manually is a tad fishy but encoding/xml
	// enforces standard XML header and the end code would be significantly
	// larger. The down side is potential escaping errors.
	b := &bytes.Buffer{}
	_, _ = io.WriteString(b, header)
	_, _ = io.WriteString(b, watermark)
	_, _ = fmt.Fprintf(b, svgTag, c.Size().X*scaleX, c.Size().Y*scaleY)

	// 3 passes, first closed paths, then open paths, then text.
	_, _ = io.WriteString(b, "  <g id=\"closed\" stroke=\"#000\" stroke-width=\"2\" fill=\"none\">\n")
	for i, obj := range c.Objects() {
		if obj.IsClosed() {
			if text := obj.Text(); text == nil {
				_, _ = fmt.Fprintf(b, pathTag, "closed", i, flatten(obj.Points(), scaleX, scaleY)+" Z")
			}
		}
	}
	_, _ = io.WriteString(b, "  </g>\n")

	_, _ = io.WriteString(b, "  <g id=\"lines\" stroke=\"#000\" stroke-width=\"2\" fill=\"none\">\n")
	for i, obj := range c.Objects() {
		if !obj.IsClosed() {
			if text := obj.Text(); text == nil {
				_, _ = fmt.Fprintf(b, pathTag, "open", i, flatten(obj.Points(), scaleX, scaleY))
			}
		}
	}
	_, _ = io.WriteString(b, "  </g>\n")

	_, _ = fmt.Fprintf(b, textGroupTag, escape(string(font)))
	for i, obj := range c.Objects() {
		if text := obj.Text(); text != nil {
			p := obj.Points()[0]
			//_, _ = fmt.Fprintf(b, textLongTag, float64(p.X*scaleX), float64(p.Y*scaleY), i, escape(font), escape(string(text)))
			_, _ = fmt.Fprintf(b, textShortTag, float64(p.X*scaleX), float64(p.Y*scaleY), i, escape(string(text)))
		}
	}
	_, _ = io.WriteString(b, "  </g>\n")

	_, _ = io.WriteString(b, "</svg>\n")

	/*
		// TODO(maruel): This looks painful!
		e := xml.NewEncoder(b)
		e.EncodeToken(
			&xml.StartElement{
				xml.Name{"svg", ""},
				[]xml.Attr{
					{xml.Name{"width", ""}, "300px"},
					{xml.Name{"height", ""}, "222px"},
					{xml.Name{"version", ""}, "1.1"},
					{xml.Name{"xmlns", ""}, "http://www.w3.org/2000/svg"},
					{xml.Name{"xmlns:xlink", ""}, "http://www.w3.org/1999/xlink"},
				},
			})
		e.Flush()
	*/
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
		out += fmt.Sprintf("%s %g %g%s", cmd, float64(p.X*scaleX), float64(p.Y*scaleY), sfx)
	}
	return out
}
