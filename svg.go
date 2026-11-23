// Copyright 2012 - 2018 The ASCIIToSVG Contributors
// All rights reserved.

package asciitosvg

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
)

const (
	defaultFont = "Consolas,Monaco,Anonymous Pro,Anonymous,Bitstream Vera Sans Mono,Liberation Mono,DejaVu Sans Mono,monospace"
	header      = "<!DOCTYPE svg PUBLIC \"-//W3C//DTD SVG 1.1//EN\" \"http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd\">\n"
	watermark   = "<!-- Created with ASCIItoSVG -->\n"
	svgTag      = "<svg width=\"%dpx\" height=\"%dpx\" version=\"1.1\" xmlns=\"http://www.w3.org/2000/svg\" xmlns:xlink=\"http://www.w3.org/1999/xlink\">\n"

	// Path related tag.
	pathTag              = "    %s<path id=\"%s%d\" %sd=\"%s\" />%s\n"
	pathMarkStart        = "marker-start=\"url(#iPointer)\" "
	pathMarkStartForward = "marker-start=\"url(#Pointer)\" "
	pathMarkEnd          = "marker-end=\"url(#Pointer)\" "
	pathMarkEndReverse   = "marker-end=\"url(#iPointer)\" "

	// Text related tag.
	textGroupTag = "  <g id=\"text\" xml:space=\"preserve\" stroke=\"none\" style=\"font-family:%s;font-size:%gpx\" >\n"
	textTag      = "    %s<text id=\"obj%d\" x=\"%g\" y=\"%g\" fill=\"%s\">%s</text>%s\n"

	// Point effect tags.
	dotTag  = "    <circle cx=\"%g\" cy=\"%g\" r=\"3\" fill=\"#000\" />\n"
	tickTag = "    <line x1=\"%g\" y1=\"%g\" x2=\"%g\" y2=\"%g\" stroke-width=\"1\" />\n"

	blurDef = `  <defs>
    <filter id="dsFilter" filterUnits="userSpaceOnUse" x="0" y="0" width="100%%" height="100%%">
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

	options := c.Options()

	// Exclude deleted tag definition lines from inflating SVG canvas dimensions.
	sizeX, sizeY := c.Size().X, c.Size().Y
	hasDeletedDef := false
	visMaxX, visMaxY := -1, -1
	for _, obj := range c.Objects() {
		if oo, ok := obj.(*object); ok && oo.isTagDef {
			if tag := obj.Tag(); tag != "" {
				if _, del := options[tag]["a2s:delref"]; del {
					hasDeletedDef = true
					continue
				}
			}
		}
		for _, pt := range obj.Points() {
			if pt.X > visMaxX {
				visMaxX = pt.X
			}
			if pt.Y > visMaxY {
				visMaxY = pt.Y
			}
		}
	}
	if hasDeletedDef && visMaxX >= 0 && visMaxY >= 0 {
		if visMaxX+1 < sizeX {
			sizeX = visMaxX + 1
		}
		if visMaxY+1 < sizeY {
			sizeY = visMaxY + 1
		}
	}

	// TODO(dhobsd): Generating the XML manually is a tad fishy but encoding/xml
	// enforces standard XML header and the end code would be significantly
	// larger. The down side is potential escaping errors.
	b := &bytes.Buffer{}
	io.WriteString(b, header)
	io.WriteString(b, watermark)
	fmt.Fprintf(b, svgTag, (sizeX+1)*scaleX, (sizeY+1)*scaleY)
	x := float64(scaleX - 1)
	y := float64(scaleY - 1)
	fmt.Fprintf(b, blurDef, x, y, x, y)

	getOpts := func(tag string, defaultFill string) string {
		var sb strings.Builder
		hasFill := false
		if tagOpts, ok := options[tag]; ok {
			keys := make([]string, 0, len(tagOpts))
			for k := range tagOpts {
				if strings.HasPrefix(k, "a2s:") {
					continue
				}
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				if k == "fill" {
					hasFill = true
				}
				fmt.Fprintf(&sb, "%s=\"%s\" ", escape(k), escape(fmt.Sprint(tagOpts[k])))
			}
		}
		if defaultFill != "" && !hasFill {
			fmt.Fprintf(&sb, "fill=\"%s\" ", escape(defaultFill))
		}
		return sb.String()
	}

	linkTags := func(tag string) (string, string) {
		if tag == "" {
			return "", ""
		}
		if link, ok := options[tag]["a2s:link"]; ok {
			if linkStr, ok := link.(string); ok && linkStr != "" {
				return fmt.Sprintf("<a xlink:href=\"%s\">", escape(linkStr)), "</a>"
			}
		}
		return "", ""
	}

	// 3 passes: first closed paths, then open paths, then text.
	if noBlur {
		io.WriteString(b, "  <g id=\"closed\" stroke=\"#000\" stroke-width=\"2\" fill=\"none\">\n")
	} else {
		io.WriteString(b, "  <g id=\"closed\" filter=\"url(#dsFilter)\" stroke=\"#000\" stroke-width=\"2\" fill=\"none\">\n")
	}
	defaultClosedFill := "#fff"
	if f, ok := options["__a2s__closed__options__"]["fill"].(string); ok && f != "" {
		defaultClosedFill = f
	}
	for i, obj := range c.Objects() {
		if obj.IsClosed() && !obj.IsText() {
			opts := ""
			if obj.IsDashed() {
				opts = "stroke-dasharray=\"5 5\" "
			}

			tag := obj.Tag()
			lookupTag := tag
			if lookupTag == "" {
				lookupTag = "__a2s__closed__options__"
			}
			opts += getOpts(lookupTag, defaultClosedFill)

			startLink, endLink := linkTags(tag)
			fmt.Fprintf(b, pathTag, startLink, "closed", i, opts, flatten(obj.Points(), true, scaleX, scaleY)+"Z", endLink)
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
			switch points[0].Hint {
			case StartMarker:
				opts += pathMarkStart
			case StartMarkerForward:
				opts += pathMarkStartForward
			}
			switch points[len(points)-1].Hint {
			case EndMarker:
				opts += pathMarkEnd
			case EndMarkerReverse:
				opts += pathMarkEndReverse
			}

			tag := obj.Tag()
			opts += getOpts(tag, "")

			startLink, endLink := linkTags(tag)
			fmt.Fprintf(b, pathTag, startLink, "open", i, opts, flatten(points, false, scaleX, scaleY), endLink)

			for _, p := range points {
				switch p.Hint {
				case Dot:
					sp := scale(p, scaleX, scaleY)
					fmt.Fprintf(b, dotTag, sp.X, sp.Y)
				case Tick:
					sp := scale(p, scaleX, scaleY)
					fmt.Fprintf(b, tickTag, sp.X-4, sp.Y-4, sp.X+4, sp.Y+4)
					fmt.Fprintf(b, tickTag, sp.X+4, sp.Y-4, sp.X-4, sp.Y+4)
				}
			}
		}
	}
	io.WriteString(b, "  </g>\n")

	fontSize := roundCoord(math.Min(float64(scaleX)/0.6, 0.9375*float64(scaleY)))
	fmt.Fprintf(b, textGroupTag, escape(font), fontSize)

	findTextColor := func(o Object) string {
		// If the tag on the text object is a special coordinate reference, use its fill directly.
		if tag := o.Tag(); objTagRE.MatchString(tag) {
			if fill, ok := options[tag]["fill"]; ok {
				if fillStr, ok := fill.(string); ok && fillStr != "" {
					return fillStr
				}
			}
		}

		// Otherwise, find the most specific enclosing container with a non-"none" fill
		// and choose a contrasting text color.
		if containers := c.EnclosingObjects(o.Points()[0]); len(containers) > 0 {
			for _, container := range containers {
				fillStr := defaultClosedFill
				if tag := container.Tag(); tag != "" {
					if fill, ok := options[tag]["fill"]; ok {
						s, ok := fill.(string)
						if !ok || s == "" {
							continue
						}
						if s == "none" {
							continue
						}
						fillStr = s
					}
				}
				if col, err := textColor(fillStr); err == nil {
					return col
				}
			}
		}

		// Default to black.
		return "#000"
	}

	for i, obj := range c.Objects() {
		if obj.IsText() {
			text := string(obj.Text())
			tag := obj.Tag()
			isTagDef := false
			if oo, ok := obj.(*object); ok {
				isTagDef = oo.isTagDef
			}

			startLink, endLink := "", ""
			if tag != "" {
				if isTagDef {
					if _, ok := options[tag]["a2s:delref"]; ok {
						continue
					}
				} else {
					if label, ok := options[tag]["a2s:label"]; ok {
						labelStr := fmt.Sprint(label)
						refPrefix := "[" + tag + "]"
						if strings.HasPrefix(text, refPrefix) {
							text = labelStr + strings.TrimPrefix(text, refPrefix)
						} else {
							text = labelStr
						}
					} else if _, ok := options[tag]["a2s:delref"]; ok {
						refPrefix := "[" + tag + "]"
						if strings.HasPrefix(text, refPrefix) {
							text = strings.TrimLeft(strings.TrimPrefix(text, refPrefix), " ")
						}
					}
					if text == "" {
						continue
					}
					startLink, endLink = linkTags(tag)
				}
			}

			color := findTextColor(obj)
			pt := obj.Points()[0]
			tx := roundCoord(float64(pt.X) * float64(scaleX))
			ty := roundCoord((float64(pt.Y) + 0.8125) * float64(scaleY))
			fmt.Fprintf(b, textTag, startLink, i, tx, ty, color, escape(text), endLink)
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

func roundCoord(v float64) float64 {
	r := math.Round(v*1000) / 1000
	if r == 0 {
		return 0
	}
	return r
}

func flattenVertices(points []Point, closed bool) []Point {
	l := len(points)
	if l < 3 {
		return points
	}
	out := []Point{points[0]}
	dir := stepDir(points[0], points[1])
	for i := 2; i < l; i++ {
		newDir := stepDir(points[i-1], points[i])
		h := points[i-1].Hint
		if dir != newDir || h == RoundedCuspDown || h == RoundedCuspUp {
			out = append(out, points[i-1])
			dir = newDir
		}
	}
	last := points[l-1]
	if !closed {
		out = append(out, last)
		return out
	}
	closingDir := stepDir(last, points[0])
	if closingDir == dirNone || dir != closingDir || last.Hint == RoundedCuspDown || last.Hint == RoundedCuspUp {
		out = append(out, last)
	}
	return out
}

func flatten(points []Point, closed bool, scaleX, scaleY int) string {
	verts := flattenVertices(points, closed)
	n := len(verts)
	if n == 0 {
		return ""
	}
	if n == 1 {
		p := scale(verts[0], scaleX, scaleY)
		return fmt.Sprintf("M %g %g ", roundCoord(p.X), roundCoord(p.Y))
	}

	rTarget := 10.0 * math.Min(float64(scaleX)/9.0, float64(scaleY)/16.0)
	if rTarget < 0 {
		rTarget = 0
	}
	rCusp := math.Min(rTarget, 0.5*float64(scaleY))

	var sb strings.Builder
	for i, cp := range verts {
		cmd := "L"
		if i == 0 {
			cmd = "M"
		}
		p := scale(cp, scaleX, scaleY)
		interior := closed || (i > 0 && i < n-1)
		isCusp := (cp.Hint == RoundedCuspDown || cp.Hint == RoundedCuspUp) && interior
		isRound := cp.Hint == RoundedCorner && interior
		if !isRound && !isCusp {
			fmt.Fprintf(&sb, "%s %g %g ", cmd, roundCoord(p.X), roundCoord(p.Y))
			continue
		}

		pp := scale(verts[(i-1+n)%n], scaleX, scaleY)
		np := scale(verts[(i+1)%n], scaleX, scaleY)
		dxIn, dyIn := pp.X-p.X, pp.Y-p.Y
		dxOut, dyOut := np.X-p.X, np.Y-p.Y
		lenIn := math.Hypot(dxIn, dyIn)
		lenOut := math.Hypot(dxOut, dyOut)
		if lenIn == 0 || lenOut == 0 {
			fmt.Fprintf(&sb, "%s %g %g ", cmd, roundCoord(p.X), roundCoord(p.Y))
			continue
		}

		if isCusp {
			rIn := math.Min(rTarget, lenIn/2)
			rOut := math.Min(rTarget, lenOut/2)
			cuspSign := 1.0
			if cp.Hint == RoundedCuspUp {
				cuspSign = -1.0
			}
			sx := roundCoord(p.X + dxIn*(rIn/lenIn))
			sy := roundCoord(p.Y + dyIn*(rIn/lenIn))
			cx := roundCoord(p.X)
			cy := roundCoord(p.Y)
			my := roundCoord(p.Y + cuspSign*rCusp)
			ex := roundCoord(p.X + dxOut*(rOut/lenOut))
			ey := roundCoord(p.Y + dyOut*(rOut/lenOut))
			fmt.Fprintf(&sb, "%s %g %g Q %g %g %g %g Q %g %g %g %g ", cmd, sx, sy, cx, cy, cx, my, cx, cy, ex, ey)
			continue
		}

		r := math.Min(rTarget, math.Min(lenIn/2, lenOut/2))
		sx := roundCoord(p.X + dxIn*(r/lenIn))
		sy := roundCoord(p.Y + dyIn*(r/lenIn))
		cx := roundCoord(p.X)
		cy := roundCoord(p.Y)
		ex := roundCoord(p.X + dxOut*(r/lenOut))
		ey := roundCoord(p.Y + dyOut*(r/lenOut))

		fmt.Fprintf(&sb, "%s %g %g Q %g %g %g %g ", cmd, sx, sy, cx, cy, ex, ey)
	}

	return sb.String()
}
