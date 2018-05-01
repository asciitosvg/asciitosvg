// Copyright 2012 - 2018 The ASCIIToSVG Contributors
// All rights reserved.

package asciitosvg

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	// TODO(dhobsd): Investigate using SVGo?
)

const (
	defaultFont = "Consolas,Monaco,Anonymous Pro,Anonymous,Bitstream Sans Mono,monospace"
	header      = "<!DOCTYPE svg PUBLIC \"-//W3C//DTD SVG 1.1//EN\" \"http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd\">\n"
	watermark   = "<!-- Created with ASCIItoSVG -->\n"
	svgTag      = "<svg width=\"%dpx\" height=\"%dpx\" version=\"1.1\" xmlns=\"http://www.w3.org/2000/svg\" xmlns:xlink=\"http://www.w3.org/1999/xlink\">\n"

	// Path related tag.
	pathTag       = "    %s<path id=\"%s%d\" %sd=\"%s\" />%s\n"
	pathMarkStart = "marker-start=\"url(#iPointer)\" "
	pathMarkEnd   = "marker-end=\"url(#Pointer)\" "

	// Text related tag.
	textGroupTag = "  <g id=\"text\" stroke=\"none\" style=\"font-family:%s;font-size:15.2px\" >\n"
	textTag      = "    %s<text id=\"obj%d\" x=\"%g\" y=\"%g\" fill=\"%s\">%s</text>%s\n"

	// Point effect tags.
	dotTag  = "    <circle cx=\"%g\" cy=\"%g\" r=\"3\" fill=\"#000\" />\n"
	tickTag = "    <line x1=\"%g\" y1=\"%g\" x2=\"%g\" y2=\"%g\" stroke-width=\"1\" />\n"

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

// CanvasToSVG renders the supplied asciitosvg.Canvas to SVG, based on the supplied options.
func CanvasToSVG(c Canvas, noBlur bool, font string, scaleX, scaleY int) []byte {
	if len(font) == 0 {
		font = defaultFont
	}

	// TODO(dhobsd): Generating the XML manually is a tad fishy but encoding/xml
	// enforces standard XML header and the end code would be significantly
	// larger. The down side is potential escaping errors.
	b := &bytes.Buffer{}
	io.WriteString(b, header)
	io.WriteString(b, watermark)
	fmt.Fprintf(b, svgTag, (c.Size().X+1)*scaleX, (c.Size().Y+1)*scaleY)
	x := float64(scaleX - 1)
	y := float64(scaleY - 1)
	fmt.Fprintf(b, blurDef, x, y, x, y)

	options := c.Options()
	getOpts := func(tag string) string {
		opts := ""
		if options, ok := options[tag]; ok {
			for k, v := range options {
				if strings.HasPrefix(k, "a2s:") {
					continue
				}

				switch v.(type) {
				case string:
					opts += fmt.Sprintf("%s=\"%s\" ", k, v.(string))
				default:
					// TODO(dhobsd): Implement.
					opts += fmt.Sprintf("%s=\"UNIMPLEMENTED\" ", k)
				}
			}
		}

		return opts
	}

	// 3 passes, first closed paths, then open paths, then text.
	io.WriteString(b, "  <g id=\"closed\" filter=\"url(#dsFilter)\" stroke=\"#000\" stroke-width=\"2\" fill=\"none\">\n")
	for i, obj := range c.Objects() {
		if obj.IsClosed() && !obj.IsText() {
			opts := ""
			if obj.IsDashed() {
				opts = "stroke-dasharray=\"5 5\" "
			}

			tag := obj.Tag()
			if tag == "" {
				tag = "__a2s__closed__options__"
			}
			opts += getOpts(tag)

			startLink, endLink := "", ""
			if link, ok := options[tag]["a2s:link"]; ok {
				startLink = link.(string)
				endLink = "</a>"
			}

			fmt.Fprintf(b, pathTag, startLink, "closed", i, opts, flatten(obj.Points(), scaleX, scaleY)+"Z", endLink)
		}
	}
	io.WriteString(b, "  </g>\n")

	io.WriteString(b, "  <g id=\"lines\" stroke=\"#000\" stroke-width=\"2\" fill=\"none\">\n")
	for i, obj := range c.Objects() {
		if !obj.IsClosed() && !obj.IsText() {
			points := obj.Points()

			opts := ""
			if obj.IsDashed() {
				opts += "stroke-dasharray=\"5 5\" "
			}
			if points[0].Hint == StartMarker {
				opts += pathMarkStart
			}
			if points[len(points)-1].Hint == EndMarker {
				opts += pathMarkEnd
			}

			for _, p := range points {
				switch p.Hint {
				case Dot:
					sp := scale(p, scaleX, scaleY)
					fmt.Fprintf(b, dotTag, sp.X, sp.Y)
				case Tick:
					p := scale(p, scaleX, scaleY)
					p1, p2 := p, p
					p1.X -= 4
					p1.Y -= 4
					p2.X += 4
					p2.Y += 4
					fmt.Fprintf(b, tickTag, p1.X, p1.Y, p2.X, p2.Y)

					p1, p2 = p, p
					p1.X += 4
					p1.Y -= 4
					p2.X -= 4
					p2.Y += 4
					fmt.Fprintf(b, tickTag, p1.X, p1.Y, p2.X, p2.Y)
				}
			}

			tag := obj.Tag()
			opts += getOpts(tag)

			startLink, endLink := "", ""
			if link, ok := options[tag]["a2s:link"]; ok {
				startLink = link.(string)
				endLink = "</a>"
			}
			fmt.Fprintf(b, pathTag, startLink, "open", i, opts, flatten(points, scaleX, scaleY), endLink)
		}
	}
	io.WriteString(b, "  </g>\n")

	fmt.Fprintf(b, textGroupTag, escape(string(font)))

	findTextColor := func(o Object) (string, error) {
		// If the tag on the text object is a special reference, that's the color we should use
		// for the text.
		if tag := o.Tag(); objTagRE.MatchString(tag) {
			if fill, ok := options[tag]["fill"]; ok {
				return fill.(string), nil
			}
		}

		// Otherwise, find the most specific fill and calibrate the color based on that.
		if containers := c.EnclosingObjects(o.Points()[0]); containers != nil {
			for _, container := range containers {
				if tag := container.Tag(); tag != "" {
					if fill, ok := options[tag]["fill"]; ok {
						if fill == "none" {
							continue
						}

						return textColor(fill.(string))
					}
				}
			}
		}

		// Default to black.
		return "#000", nil
	}

	for i, obj := range c.Objects() {
		if obj.IsText() {
			// Look up the fill of the containing box to determine what text color to use.
			color, err := findTextColor(obj)
			if err != nil {
				fmt.Printf("Error figuring out text color: %s\n", err)
			}

			startLink, endLink := "", ""
			text := string(obj.Text())
			tag := obj.Tag()
			if tag != "" {
				if label, ok := options[tag]["a2s:label"]; ok {
					text = label.(string)
				}

				// If we're a reference, the a2s:delref tag informs us to remove our reference.
				// TODO(dhobsd): If text is on column 0 but is not a special reference,
				// we can't really detect that here.
				if obj.Corners()[0].X == 0 {
					if _, ok := options[tag]["a2s:delref"]; ok {
						continue
					}
				}

				if link, ok := options[tag]["a2s:link"]; ok {
					startLink = link.(string)
					endLink = "</a>"
				}
			}
			sp := scale(obj.Points()[0], scaleX, scaleY)
			fmt.Fprintf(b, textTag, startLink, i, sp.X, sp.Y, color, escape(text), endLink)
		}
	}
	io.WriteString(b, "  </g>\n")

	io.WriteString(b, "</svg>\n")
	return b.Bytes()
}

func escape(s string) string {
	b := &bytes.Buffer{}
	if err := xml.EscapeText(b, []byte(s)); err != nil {
		panic(err)
	}
	return b.String()
}

type scaledPoint struct {
	X    float64
	Y    float64
	Hint RenderHint
}

func scale(p Point, scaleX, scaleY int) scaledPoint {
	return scaledPoint{
		X:    (float64(p.X) + .5) * float64(scaleX),
		Y:    (float64(p.Y) + .5) * float64(scaleY),
		Hint: p.Hint,
	}
}

func flatten(points []Point, scaleX, scaleY int) string {
	out := ""

	// Scaled start point, and previous point (which is always initially the start point).
	sp := scale(points[0], scaleX, scaleY)
	pp := sp

	for i, cp := range points {
		p := scale(cp, scaleX, scaleY)

		// Our start point is represented by a single moveto command (unless the start point
		// is a rounded corner) as the shape will be closed with the Z command automatically
		// if we have a closed polygon. If our start point is a rounded corner, we have to go
		// ahead and draw that curve.
		if i == 0 {
			if cp.Hint == RoundedCorner {
				out += fmt.Sprintf("M %g %g Q %g %g %g %g ", p.X, p.Y+10, p.X, p.Y, p.X+10, p.Y)
				continue
			}

			out += fmt.Sprintf("M %g %g ", p.X, p.Y)
			continue
		}

		// If this point has a rounded corner, we need to calculate the curve. This algorithm
		// only works when the shapes are drawn in a clockwise manner.
		if cp.Hint == RoundedCorner {
			// The control point is always the original corner.
			cx := p.X
			cy := p.Y

			sx, sy, ex, ey := 0., 0., 0., 0.

			// We need to know the next point to determine which way to turn.
			var np scaledPoint
			if i == len(points)-1 {
				np = sp
			} else {
				np = scale(points[i+1], scaleX, scaleY)
			}

			if pp.X == p.X {
				// If we're on the same vertical axis, our starting X coordinate is
				// the same as the control point coordinate
				sx = p.X

				// Offset start point from control point in the proper direction.
				if pp.Y < p.Y {
					sy = p.Y - 10
				} else {
					sy = p.Y + 10
				}

				ey = p.Y
				// Offset endpoint from control point in the proper direction.
				if np.X < p.X {
					ex = p.X - 10
				} else {
					ex = p.X + 10
				}
			} else if pp.Y == p.Y {
				// Horizontal decisions mirror vertical's above.
				sy = p.Y
				if pp.X < p.X {
					sx = p.X - 10
				} else {
					sx = p.X + 10
				}
				ex = p.X
				if np.Y <= p.Y {
					ey = p.Y - 10
				} else {
					ey = p.Y + 10
				}
			}

			out += fmt.Sprintf("L %g %g Q %g %g %g %g ", sx, sy, cx, cy, ex, ey)
		} else {
			// Oh, the horrors of drawing a straight line...
			out += fmt.Sprintf("L %g %g ", p.X, p.Y)
		}

		pp = p
	}

	return out
}
