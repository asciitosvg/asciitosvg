// Copyright 2012 - 2015 The ASCIIToSVG Contributors
// All rights reserved.

package asciitosvg

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
)

const (
	defaultFont = "Consolas,Monaco,Anonymous Pro,Anonymous,Bitstream Sans Mono,monospace"
	header      = "<!DOCTYPE svg PUBLIC \"-//W3C//DTD SVG 1.1//EN\" \"http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd\">\n"
	watermark   = "<!-- Created with ASCIItoSVG -->\n"
	svgTag      = "<svg width=\"%dpx\" height=\"%dpx\" version=\"1.1\" xmlns=\"http://www.w3.org/2000/svg\" xmlns:xlink=\"http://www.w3.org/1999/xlink\">\n"

	// Path related tag.
	pathTag       = "    <path id=\"%s%d\" %sd=\"%s\" />\n"
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
			opts := ""
			if obj.IsDashed() {
				opts = "stroke-dasharray=\"5 5\" "
			}
			_, _ = fmt.Fprintf(b, pathTag, "closed", i, opts, flatten(obj.Points(), scaleX, scaleY)+"Z")
		}
	}
	_, _ = io.WriteString(b, "  </g>\n")

	_, _ = io.WriteString(b, "  <g id=\"lines\" stroke=\"#000\" stroke-width=\"2\" fill=\"none\">\n")
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
			_, _ = fmt.Fprintf(b, pathTag, "open", i, opts, flatten(points, scaleX, scaleY))
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
