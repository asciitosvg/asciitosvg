// Copyright 2012 - 2015 The ASCIIToSVG Contributors
// All rights reserved.

package asciitosvg

import (
	"bytes"
	"fmt"
	"image"
	"sort"
	"unicode/utf8"
)

// Object is an interface for working with open paths (lines), closed paths (polygons), or text.
type Object interface {
	fmt.Stringer
	// Points returns all the points occupied by this Object. Every object has at least one point,
	// and all points are both in-order and contiguous.
	Points() []image.Point
	// Corners returns all the corners (change of direction) along the path.
	Corners() []image.Point
	// IsClosed is true if the object is composed of a closed path.
	IsClosed() bool
	// IsText returns true if the object is textual and does not represent a path.
	IsText() bool
	// Text returns the text associated with this Object if textual, and nil otherwise.
	Text() []rune
}

// Canvas provides methods for returning objects from an underlying textual grid.
type Canvas interface {
	// A canvas has an underlying visual representation. The fmt.Stringer interface for this
	// interface provides a view into the underlying grid.
	fmt.Stringer
	// Objects returns all the objects found in the underlying grid.
	Objects() []Object
	// Size returns the visual dimensions of the Canvas.
	Size() image.Point
}

// NewCanvas returns a new Canvas, initialized from the provided data. If tabWidth is set to a non-negative
// value, that value will be used to convert tabs to spaces within the grid.
func NewCanvas(data []byte, tabWidth int) (Canvas, error) {
	c := &canvas{}

	lines := bytes.Split(data, []byte("\n"))
	c.size.Y = len(lines)

	// Diagrams will often not be padded to a uniform width. To overcome this, we scan over
	// each line and figure out which is the longest. This becomes the width of the canvas.
	for i, line := range lines {
		if ok := utf8.Valid(line); !ok {
			return nil, fmt.Errorf("invalid UTF-8 encoding on line %d", i)
		}

		if l, err := expandTabs(line, tabWidth); err != nil {
			return nil, err
		} else {
			lines[i] = l
		}

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

	c.findObjects()
	return c, nil
}

// The expandTabs function pads tab characters to the specified width of spaces for the provided
// line of input. We cannot simply pad based on byte-offset since our input is UTF-8 encoded.
// Fortunately, we can assume that this function is called that the line contains only valid
// UTF-8 sequences. We first decode the line rune-wise, and use individual runes to figure out
// where we are within the line. When we encounter a tab character, we expand based on our rune
// index.
func expandTabs(line []byte, tabWidth int) ([]byte, error) {
	// Initial sizing of our output slice assumes no UTF-8 bytes or tabs, since this is often
	// the common case.
	out := make([]byte, 0, len(line))

	// pos tracks our position in the input byte slice, while index tracks our position in the
	// resulting output slice.
	pos := 0
	index := 0
	for _, c := range line {
		if c == '\t' {
			// Loop over the remaining space count for this particular tabstop until
			// the next, replacing each position with a space.
			for s := tabWidth - (pos % tabWidth); s > 0; s-- {
				out = append(out, ' ')
				index++
			}
			pos++
		} else {
			// We need to know the byte length of the rune at this position so that we
			// can account for our tab expansion properly. So we first decode the rune
			// at this position to get its length in bytes, plop that rune back into our
			// output slice, and account accordingly.
			r, l := utf8.DecodeRune(line[pos:])
			if r == utf8.RuneError {
				return nil, fmt.Errorf("invalid rune at byte offset %d; rune offset %d", pos, index)
			}

			enc := make([]byte, l)
			utf8.EncodeRune(enc, r)
			out = append(out, enc...)

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
}

type objects []Object

func (c *canvas) String() string {
	return fmt.Sprintf("%+v", c.grid)
}

func (c *canvas) Objects() []Object {
	return c.objects
}

func (c *canvas) Size() image.Point {
	return c.size
}

// findObjects finds all objects (lines, polygons, and text) within the underlying grid.
func (c *canvas) findObjects() {
	p := image.Point{}

	// Find any new paths by starting with a point that wasn't yet visited, beginning at the top
	// left of the grid.
	for y := 0; y < c.size.Y; y++ {
		p.Y = y
		for x := 0; x < c.size.X; x++ {
			p.X = x
			if c.isVisited(p) {
				continue
			}
			if ch := c.at(p); ch.isPathStart() {
				// Found the start of a one or multiple connected paths. Traverse all
				// connecting points. This will generate multiple objects if multiple
				// paths (either open or closed) are found.
				c.visit(p)
				objs := c.scanPath([]image.Point{p})
				for _, obj := range objs {
					// For all points in all objects found, mark the points as visited.
					for _, p := range obj.Points() {
						c.visit(p)
					}
				}
				c.objects = append(c.objects, objs...)
			}
		}
	}

	// A second pass through the grid attempts to identify any text within the grid.
	for y := 0; y < c.size.Y; y++ {
		p.Y = y
		for x := 0; x < c.size.X; x++ {
			p.X = x
			if c.isVisited(p) {
				continue
			}
			if ch := c.at(p); ch.isTextStart() {
				obj := c.scanText(p)
				for _, p := range obj.Points() {
					c.visit(p)
				}
				c.objects = append(c.objects, obj)
			}
		}
	}

	sort.Sort(c.objects)
}

// scanPath tries to complete a total path (for lines or polygons) starting with some partial path.
// It recurses when it finds multiple unvisited outgoing paths.
func (c *canvas) scanPath(points []image.Point) objects {
	cur := points[len(points)-1]
	next := c.next(cur)

	// If there are no points that can progress traversal of the path, finalize the one we're
	// working on, and return it. This is the terminal condition in the passive flow.
	if len(next) == 0 {
		if len(points) == 1 {
			// Discard 'path' of 1 point. Do not mark point as visited.
			c.unvisit(cur)
			return nil
		}

		// TODO(dhobsd): Determine if path is sharing the line with another path. If so,
		// we may want to join the objects such that we don't get weird rendering artifacts.
		o := &object{points: points}
		o.seal(c)
		return objects{o}
	}

	// If we have hit a point that can create a closed path, create an object and close
	// the path. Additionally, recurse to other progress directions in case e.g. an open
	// path spawns from this point. Paths are always closed vertically.
	if cur.X == points[0].X && cur.Y == points[0].Y+1 {
		o := &object{points: points}
		o.seal(c)
		r := objects{o}
		return append(r, c.scanPath([]image.Point{cur})...)
	}

	// We scan depth-first instead of breadth-first, making it possible to find a
	// closed path.
	var objs objects
	for _, n := range next {
		if c.isVisited(n) {
			continue
		}
		c.visit(n)
		p2 := make([]image.Point, len(points)+1)
		copy(p2, points)
		p2[len(p2)-1] = n
		objs = append(objs, c.scanPath(p2)...)
	}
	return objs
}

// The next returns the points that can be used to make progress, scanning (in order) horizontal
// progress to left or right, and vertical progress above or below. It skips any points already
// visited, and returns all of the possible progress points.
func (c *canvas) next(pos image.Point) []image.Point {
	// Our caller must have called c.visit prior to calling this function.
	if !c.isVisited(pos) {
		panic(fmt.Errorf("internal error; revisiting %s", pos))
	}

	var out []image.Point

	// Look at the current point in the grid and determine
	ch := c.at(pos)
	if ch.canHorizontal() {
		if c.canLeft(pos) {
			n := pos
			n.X--
			if !c.isVisited(n) && c.at(n).canHorizontal() {
				out = append(out, n)
			}
		}
		if c.canRight(pos) {
			n := pos
			n.X++
			if !c.isVisited(n) && c.at(n).canHorizontal() {
				out = append(out, n)
			}
		}
	}
	if ch.canVertical() {
		if c.canUp(pos) {
			n := pos
			n.Y--
			if !c.isVisited(n) && c.at(n).canVertical() {
				out = append(out, n)
			}
		}
		if c.canDown(pos) {
			n := pos
			n.Y++
			if !c.isVisited(n) && c.at(n).canVertical() {
				out = append(out, n)
			}
		}
	}
	return out
}

// scanText extracts a line of text.
func (c *canvas) scanText(start image.Point) Object {
	obj := &object{points: []image.Point{start}, isText: true}
	whiteSpaceStreak := 0
	cur := start
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
		if ch.isSpace() {
			whiteSpaceStreak++
			// Stop when we see 3 consecutive whitespace points.
			if whiteSpaceStreak > 2 {
				break
			}
		} else {
			whiteSpaceStreak = 0
		}
		obj.points = append(obj.points, cur)
	}
	// Trim the right side of the text object.
	for len(obj.points) != 0 && c.at(obj.points[len(obj.points)-1]).isSpace() {
		obj.points = obj.points[:len(obj.points)-1]
	}
	obj.seal(c)
	return obj
}

func (c *canvas) at(p image.Point) char {
	return c.grid[p.Y*c.size.X+p.X]
}

func (c *canvas) isVisited(p image.Point) bool {
	return c.visited[p.Y*c.size.X+p.X]
}

func (c *canvas) visit(p image.Point) {
	// TODO(dhobsd): Change code to ensure that visit() is called once and only
	// once per point.
	c.visited[p.Y*c.size.X+p.X] = true
}

func (c *canvas) unvisit(p image.Point) {
	o := p.Y*c.size.X + p.X
	if !c.visited[o] {
		panic(fmt.Errorf("internal error: point %+v never visited", p))
	}
	c.visited[o] = false
}

func (c *canvas) canLeft(p image.Point) bool {
	return p.X > 0
}

func (c *canvas) canRight(p image.Point) bool {
	return p.X < c.size.X-1
}

func (c *canvas) canUp(p image.Point) bool {
	return p.Y > 0
}

func (c *canvas) canDown(p image.Point) bool {
	return p.Y < c.size.Y-1
}

// object implements Object and represents one of an open path, a closed path, or text.
type object struct {
	// points always starts with the top most, then left most point, proceeding to the right.
	points   []image.Point
	isText   bool
	text     []rune
	corners  []image.Point
	isClosed bool
}

func (o *object) Points() []image.Point {
	return o.points
}

func (o *object) Corners() []image.Point {
	return o.corners
}

func (o *object) IsClosed() bool {
	return o.isClosed
}

func (o *object) IsText() bool {
	return o.isText
}

func (o *object) Text() []rune {
	return o.text
}

func (o *object) String() string {
	if o.IsText() {
		return fmt.Sprintf("Text{%s %q}", o.points[0], string(o.text))
	}
	return fmt.Sprintf("Path{%v}", o.points)
}

// seal finalizes the object, updating text, corners, and isClosed.
func (o *object) seal(c *canvas) {
	o.corners, o.isClosed = pointsToCorners(o.points)
	o.text = make([]rune, len(o.points))
	for i, p := range o.points {
		o.text[i] = rune(c.at(p))
	}
}

func (o objects) Len() int      { return len(o) }
func (o objects) Swap(i, j int) { o[i], o[j] = o[j], o[i] }

// Less returns in order top most, then left most.
func (o objects) Less(i, j int) bool {
	// TODO(dhobsd): This doesn't catch every z-index case we could possibly want. We should
	// support z-indexing of objects through an a2s tag.
	l := o[i]
	r := o[j]
	lt := l.IsText()
	rt := r.IsText()
	if lt != rt {
		return rt
	}
	lp := l.Points()[0]
	rp := r.Points()[0]
	if lp.Y != rp.Y {
		return lp.Y < rp.Y
	}
	return lp.X < rp.X
}

// isHorizontal returns if p1 and p2 are horizontally aligned.
func isHorizontal(p1, p2 image.Point) bool {
	d := p1.X - p2.X
	return d <= 1 && d >= -1 && p1.Y == p2.Y
}

// isVertical returns if p1 and p2 are vertically aligned.
func isVertical(p1, p2 image.Point) bool {
	d := p1.Y - p2.Y
	return d <= 1 && d >= -1 && p1.X == p2.X
}

// pointsToCorners returns all the corners (points at which there is a change of directionality) for
// a path. It additionally returns a truth value indicating whether the points supplied indicate a
// closed path.
func pointsToCorners(points []image.Point) ([]image.Point, bool) {
	l := len(points)
	// A path containing fewer than 3 points can neither be closed, nor change direction.
	if l < 3 {
		return points, false
	}
	out := []image.Point{points[0]}
	horiz := false
	if isHorizontal(points[0], points[1]) {
		horiz = true
	} else if isVertical(points[0], points[1]) {
		horiz = false
	} else {
		panic(fmt.Errorf("discontiguous points: %+v", points))
	}
	for i := 2; i < l; i++ {
		if isHorizontal(points[i-1], points[i]) {
			if !horiz {
				out = append(out, points[i-1])
				horiz = true
			}
		} else if isVertical(points[i-1], points[i]) {
			if horiz {
				out = append(out, points[i-1])
				horiz = false
			}
		} else {
			panic(fmt.Errorf("discontiguous points: %+v", points))
		}
	}

	// Check if the points indicate a closed path. If not, append the last point.
	last := points[l-1]
	closed := true
	if isHorizontal(points[0], last) {
		if !horiz {
			closed = false
			out = append(out, last)
		}
	} else if isVertical(points[0], last) {
		if horiz {
			closed = false
			out = append(out, last)
		}
	} else {
		closed = false
		out = append(out, last)
	}
	/* TODO(dhobsd): Something's broken.
	if !isHorizontal(points[0], last) && !isVertical(points[0], last) {
		closed = false
		out = append(out, last)
	}
	*/
	return out, closed
}
