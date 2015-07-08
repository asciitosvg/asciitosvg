// Copyright 2012 - 2015 The ASCIIToSVG Contributors
// All rights reserved.

package asciitosvg

import (
	"bytes"
	"fmt"
	"image"
	"sort"
	"unicode"
	"unicode/utf8"
)

// Object is either a open path, a closed path (polygon) or text.
type Object interface {
	fmt.Stringer
	// Points returns all the points occupied by this Object. There is at least
	// one points and all points must be in order and contiguous.
	Points() []image.Point
	// Corners returns all the corners (change of direction) along the path.
	Corners() []image.Point
	// IsClosed is true if it is a closed path.
	IsClosed() bool
	// IsText returns true if it represent text.
	IsText() bool
	// Text returns the text associated with this Object if it represents text.
	// Otherwise returns nil.
	Text() []rune
}

// Canvas is a processed objects.
type Canvas interface {
	// Objects returns all the objects found.
	Objects() []Object
	Size() image.Point
}

// Parse returns an initialized Canvas.
//
// Expands tabs to tabWidth as whitespace.
func Parse(data []byte, tabWidth int) Canvas {
	c := &canvas{}
	lines := bytes.Split(data, []byte("\n"))
	c.size.Y = len(lines)
	for i, line := range lines {
		lines[i] = expandTabs(line, tabWidth)
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
	return c
}

// Private details.

// canvas is the parsed source data.
type canvas struct {
	// (0,0) is top left.
	grid    []char
	visited []bool
	objects objects
	size    image.Point
}

func (c *canvas) Objects() []Object {
	return c.objects
}

func (c *canvas) Size() image.Point {
	return c.size
}

func (c *canvas) findObjects() {
	p := image.Point{}

	// The logic is to find any new paths by starting with a point that wasn't
	// touched yet.
	for y := 0; y < c.size.Y; y++ {
		p.Y = y
		for x := 0; x < c.size.X; x++ {
			p.X = x
			if c.isVisited(p) {
				continue
			}
			// TODO(maruel): Can't accept '>' or downArrow as starting paths.
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

// scanPath tries to complete one or multiple path or box starting with the
// partial path. It recursively calls itself when it finds multiple unvisited
// out-going paths.
func (c *canvas) scanPath(points []image.Point) objects {
	cur := points[len(points)-1]
	next := c.next(cur)
	if len(next) == 0 {
		if len(points) == 1 {
			// Discard 'path' of 1 point. Do not mark point as visited.
			c.unvisit(cur)
			return nil
		}
		// TODO(maruel): Determine if path is sharing the line another path.
		o := &object{points: points}
		o.seal(c)
		return objects{o}
	}
	var objs objects
	for _, n := range next {
		// Go depth first instead of bread first, this makes it workable for closed
		// path.
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

// next returns the next points that can be used to make progress.
//
// Look at top, left, right, down, skipping visited points and returns all the
// possibilities.
func (c *canvas) next(pos image.Point) []image.Point {
	var out []image.Point
	if !c.isVisited(pos) {
		panic(fmt.Errorf("Internal error; revisiting %s", pos))
	}
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
			// Hit a box or path.
			break
		}
		ch := c.at(cur)
		if !ch.isTextCont() {
			break
		}
		if ch.isSpace() {
			whiteSpaceStreak++
			// Stop if hit 3 consecutive whitespace.
			if whiteSpaceStreak > 2 {
				break
			}
		} else {
			whiteSpaceStreak = 0
		}
		obj.points = append(obj.points, cur)
	}
	// TrimRight space.
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
	// TODO(maruel): Change code to ensure that visit() is called once and only
	// once per point.
	c.visited[p.Y*c.size.X+p.X] = true
}

func (c *canvas) unvisit(p image.Point) {
	o := p.Y*c.size.X + p.X
	if !c.visited[o] {
		panic("Internal error")
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

// object implements Object.
//
// It can be either an open path, a closed path or text.
type object struct {
	// points always starts with the top most, then left most point, starting to
	// the right.
	points []image.Point
	isText bool

	// Updated by seal().
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
	return fmt.Sprintf("Path{%s}", o.points[0])
}

// seal finalizes the object.
//
// It updates text, corners and isClosed.
func (o *object) seal(c *canvas) {
	o.corners, o.isClosed = pointsToCorners(o.points)
	o.text = make([]rune, len(o.points))
	for i, p := range o.points {
		o.text[i] = rune(c.at(p))
	}
}

// objects is all objects found.
type objects []Object

func (o objects) Len() int      { return len(o) }
func (o objects) Swap(i, j int) { o[i], o[j] = o[j], o[i] }

// Less returns in order top most, then left most.
func (o objects) Less(i, j int) bool {
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

type char rune

func (c char) isTextStart() bool {
	r := rune(c)
	return unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSymbol(r)
}

func (c char) isTextCont() bool {
	return unicode.IsPrint(rune(c))
}

func (c char) isSpace() bool {
	return unicode.IsSpace(rune(c))
}

// isPathStart returns true on any form of ascii art that can start a graph.
func (c char) isPathStart() bool {
	return c.isCorner() || c.isHorizontal() || c.isVertical() || c.isArrowHorizontalLeft() || c.isArrowVerticalUp()
}

func (c char) isCorner() bool {
	return c == '.' || c == '\'' || c == '+'
}

func (c char) isHorizontal() bool {
	return c == '-'
}

func (c char) isVertical() bool {
	return c == '|'
}

func (c char) isArrowHorizontalLeft() bool {
	return c == '<'
}

func (c char) isArrowHorizontal() bool {
	return c.isArrowHorizontalLeft() || c == '>'
}

func (c char) isArrowVerticalUp() bool {
	return c == '^'
}

func (c char) isArrowVertical() bool {
	return c.isArrowVerticalUp() || c == 'v'
}

func (c char) isArrow() bool {
	return c.isArrowHorizontal() || c.isArrowVertical()
}

// TODO(maruel): Diagonal.

func (c char) canHorizontal() bool {
	return c.isHorizontal() || c.isCorner() || c.isArrowHorizontal()
}

func (c char) canVertical() bool {
	return c.isVertical() || c.isCorner() || c.isArrowVertical()
}

func expandTabs(line []byte, tabWidth int) []byte {
	return line
	/* TODO(maruel): Implement.
	out := make([]byte, 0, len(line))
	index := 0
	for _, c := range line {
		if c == '\t' {
			for l := (index + 1) % tabWidth; l > 0; l-- {
				out = append(out, ' ')
				index++
			}
		} else {
			index++
			out = append(out, c)
		}
	}
	return out
	*/
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

// pointsToCorners returns all the corners (points where there is a change of
// directionality) for a path. Second return value is true if the path is
// closed.
func pointsToCorners(points []image.Point) ([]image.Point, bool) {
	l := len(points)
	if l == 1 || l == 2 {
		return points, false
	}
	out := []image.Point{points[0]}
	horiz := false
	if isHorizontal(points[0], points[1]) {
		horiz = true
	} else if isVertical(points[0], points[1]) {
		horiz = false
	} else {
		panic("discontinuous points")
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
			panic("discontinuous points")
		}
	}
	// Check if a closed path or not. If not, append the last point.
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
	/* TODO(maruel): Something's broken.
	if !isHorizontal(points[0], last) && !isVertical(points[0], last) {
		closed = false
		out = append(out, last)
	}
	*/
	return out, closed
}
