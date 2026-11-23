// Copyright 2012 - 2018 The ASCIIToSVG Contributors
// All rights reserved.

package asciitosvg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Canvas provides methods for returning objects from an underlying textual grid.
type Canvas interface {
	// A canvas has an underlying visual representation. The fmt.Stringer interface for this
	// interface provides a view into the underlying grid.
	fmt.Stringer
	// Objects returns all the objects found in the underlying grid.
	Objects() []Object
	// Size returns the visual dimensions of the Canvas.
	Size() image.Point
	// Options returns a map of options to apply to Objects based on the object's tag. This
	// maps tag name to a map of option names to options.
	Options() map[string]map[string]interface{}
	// EnclosingObjects returns the set of objects that contain this point in order from most
	// to least specific.
	EnclosingObjects(p Point) []Object
}

// NewCanvas returns a new Canvas, initialized from the provided data. If tabWidth is set to a non-negative
// value, that value will be used to convert tabs to spaces within the grid. Creation of the Canvas
// can fail if the diagram contains invalid UTF-8 sequences.
func NewCanvas(data []byte, tabWidth int, noBlur bool) (Canvas, error) {
	c := &canvas{
		options: map[string]map[string]interface{}{
			"__a2s__closed__options__": map[string]interface{}{
				"fill":   "#fff",
				"filter": "url(#dsFilter)",
			},
		},
	}
	if noBlur {
		c.options["__a2s__closed__options__"] = map[string]interface{}{
			"fill": "#fff",
		}
	}

	lines := bytes.Split(data, []byte("\n"))
	c.size.Y = len(lines)

	// Diagrams will often not be padded to a uniform width. To overcome this, we scan over
	// each line and figure out which is the longest. This becomes the width of the canvas.
	for i, line := range lines {
		line = bytes.TrimSuffix(line, []byte("\r"))
		if ok := utf8.Valid(line); !ok {
			return nil, fmt.Errorf("invalid UTF-8 encoding on line %d", i)
		}

		l, err := expandTabs(line, tabWidth)
		if err != nil {
			return nil, err
		}

		lines[i] = l

		if i := utf8.RuneCount(lines[i]); i > c.size.X {
			c.size.X = i
		}
	}

	c.grid = make([]char, c.size.X*c.size.Y)
	c.visited = make([]bool, c.size.X*c.size.Y)
	for y, line := range lines {
		x := 0
		for len(line) > 0 {
			r, l := utf8.DecodeRune(line)
			c.grid[y*c.size.X+x] = char(r)
			x++
			line = line[l:]
		}

		for ; x < c.size.X; x++ {
			c.grid[y*c.size.X+x] = ' '
		}
	}

	if err := c.findObjects(); err != nil {
		return nil, err
	}
	return c, nil
}

// The expandTabs function pads tab characters to the specified width of spaces for the provided
// line of input. We cannot simply pad based on byte-offset since our input is UTF-8 encoded.
// Fortunately, we can assume that this function is called when the line contains only valid
// UTF-8 sequences. We decode the line rune-wise, tracking our column rune index, and expand
// tabs to the next tabstop based on that rune index.
func expandTabs(line []byte, tabWidth int) ([]byte, error) {
	out := make([]byte, 0, len(line))

	pos := 0
	index := 0
	for pos < len(line) {
		if line[pos] == '\t' {
			if tabWidth > 0 {
				for s := tabWidth - (index % tabWidth); s > 0; s-- {
					out = append(out, ' ')
					index++
				}
			} else if tabWidth < 0 {
				out = append(out, '\t')
				index++
			}
			pos++
		} else {
			r, l := utf8.DecodeRune(line[pos:])
			if r == utf8.RuneError && l == 1 {
				return nil, fmt.Errorf("invalid rune at byte offset %d; rune offset %d", pos, index)
			}

			out = append(out, line[pos:pos+l]...)
			pos += l
			index++
		}
	}

	return out, nil
}

// canvas is the parsed source data.
type canvas struct {
	// (0,0) is top left.
	grid    []char
	visited []bool
	objects objects
	size    image.Point
	options map[string]map[string]interface{}
}

func (c *canvas) String() string {
	return fmt.Sprintf("%+v", c.grid)
}

func (c *canvas) Objects() []Object {
	return c.objects
}

func (c *canvas) Size() image.Point {
	return c.size
}

func (c *canvas) Options() map[string]map[string]interface{} {
	return c.options
}

func (c *canvas) EnclosingObjects(p Point) []Object {
	type hit struct {
		obj   Object
		area2 float64
	}
	var hits []hit
	for _, o := range c.objects {
		// An object can't really contain another unless it is a polygon.
		if !o.IsClosed() || !o.HasPoint(p) {
			continue
		}
		hits = append(hits, hit{
			obj:   o,
			area2: math.Abs(float64(signedArea2(o.Corners()))),
		})
	}
	if len(hits) == 0 {
		return nil
	}

	sort.SliceStable(hits, func(i, j int) bool {
		return hits[i].area2 < hits[j].area2
	})

	q := make([]Object, len(hits))
	for i, h := range hits {
		q[i] = h.obj
	}
	return q
}

// findObjects finds all objects (lines, polygons, and text) within the underlying grid.
func (c *canvas) findObjects() error {
	p := Point{}

	// First pass: build the diagram connectivity graph and extract closed polygons and open paths.
	g := newGridGraph(c)
	objs := g.extractObjects()
	for _, obj := range objs {
		for _, pt := range obj.Points() {
			c.visit(pt)
		}
	}
	c.objects = append(c.objects, objs...)

	// Second pass: identify any text and tag definitions within the unvisited grid cells.
	var coordTags []string
	for y := 0; y < c.size.Y; y++ {
		p.Y = y
		for x := 0; x < c.size.X; x++ {
			p.X = x
			if c.isVisited(p) {
				continue
			}
			if ch := c.at(p); ch.isTextStart() {
				obj, coordTag, err := c.scanText(p)
				if err != nil {
					return err
				}
				if coordTag != "" {
					coordTags = append(coordTags, coordTag)
				}

				if obj == nil {
					continue
				}
				for _, pt := range obj.Points() {
					c.visit(pt)
				}
				c.objects = append(c.objects, obj)
			}
		}
	}

	// Apply [X,Y] coordinate-based tags after all objects have been discovered.
	for _, t := range coordTags {
		if matches := objTagRE.FindStringSubmatch(t); matches != nil {
			targetX, errX := strconv.ParseInt(matches[1], 10, 0)
			targetY, errY := strconv.ParseInt(matches[2], 10, 0)
			if errX == nil && errY == nil {
				for _, o := range c.objects {
					if oo, ok := o.(*object); ok && oo.isTagDef {
						continue
					}
					corner := o.Corners()[0]
					if corner.X == int(targetX) && corner.Y == int(targetY) {
						o.SetTag(t)
						break
					}
				}
			}
		}
	}

	sort.Sort(c.objects)
	return nil
}

// Used for matching [X, Y]: {...} tag definitions. These definitions target specific objects.
var objTagRE = regexp.MustCompile(`^\s*(\d+)\s*,\s*(\d+)\s*$`)

// scanText extracts a line of text.
func (c *canvas) scanText(start Point) (Object, string, error) {
	obj := &object{points: []Point{start}, isText: true}
	whiteSpaceStreak := 0
	cur := start

	tagged := 0
	var tag []rune
	var tagDef []rune

	if c.at(start).isObjectStartTag() {
		tagged = 1
	}

	for c.canRight(cur) {
		cur.X++
		if c.isVisited(cur) {
			// If the point is already visited, we hit a polygon or a line.
			break
		}
		ch := c.at(cur)
		if !ch.isTextCont() {
			break
		}
		if tagged != 1 && tagged != 3 {
			if ch.isSpace() {
				whiteSpaceStreak++
				// Stop when we see 3 consecutive whitespace points.
				if whiteSpaceStreak > 2 {
					break
				}
			} else {
				whiteSpaceStreak = 0
			}
		}

		switch tagged {
		case 1:
			if ch.isObjectEndTag() {
				tagged = 2
			} else {
				tag = append(tag, rune(ch))
			}
		case 2:
			prev := obj.points[len(obj.points)-1]
			if c.at(prev).isObjectEndTag() && ch.isTagDefinitionSeparator() {
				tagged = 3
			}
		case 3:
			tagDef = append(tagDef, rune(ch))
		}

		obj.points = append(obj.points, cur)
	}

	// If we found a start and end tag marker, we either assign the tag to the enclosing
	// object, or parse the JSON tag definition into the canvas option space.
	coordTag := ""
	if tagged == 2 && len(tag) > 0 {
		t := string(tag)
		if container := c.EnclosingObjects(start); len(container) > 0 {
			container[0].SetTag(t)
		}

		// The tag applies to the text object as well so that properties like
		// a2s:label can be set.
		obj.SetTag(t)
	} else if tagged == 3 && len(tag) > 0 {
		defStr := strings.TrimSpace(string(tagDef))
		if strings.HasPrefix(defStr, "{") {
			var m map[string]interface{}
			if err := json.Unmarshal([]byte(defStr), &m); err != nil {
				return nil, "", fmt.Errorf("invalid tag definition for [%s]: %w", string(tag), err)
			}

			t := string(tag)
			if objTagRE.MatchString(t) {
				coordTag = t
			}
			// The tag applies to the reference object as well, so that properties like
			// a2s:delref can be handled.
			obj.isTagDef = true
			obj.SetTag(t)
			c.options[t] = m
		}
	}

	// Trim the right side of the text object.
	for len(obj.points) != 0 && c.at(obj.points[len(obj.points)-1]).isSpace() {
		obj.points = obj.points[:len(obj.points)-1]
	}

	obj.seal(c)
	return obj, coordTag, nil
}

func (c *canvas) at(p Point) char {
	return c.grid[p.Y*c.size.X+p.X]
}

func (c *canvas) isVisited(p Point) bool {
	return c.visited[p.Y*c.size.X+p.X]
}

func (c *canvas) visit(p Point) {
	c.visited[p.Y*c.size.X+p.X] = true
}

func (c *canvas) canRight(p Point) bool {
	return p.X < c.size.X-1
}
