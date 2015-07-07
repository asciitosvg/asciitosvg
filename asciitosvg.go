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
	// Text returns the text associated with this Object if it represents text.
	// Otherwise returns nil.
	Text() []rune
}

// Objects is all objects found.
type Objects []Object

func (o Objects) Len() int      { return len(o) }
func (o Objects) Swap(i, j int) { o[i], o[j] = o[j], o[i] }

// Less returns in order top most, then left most.
func (o Objects) Less(i, j int) bool {
	l := o[i]
	r := o[j]
	lt := l.Text() != nil
	rt := r.Text() != nil
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

// ToSVG is scaffolding.
func (o Objects) ToSVG(noBlur bool, font string, scaleX, scaleY int) []byte {
	// TODO(maruel): Create XML then serialize it.
	return nil
}

// Canvas is a processed objects.
// TODO(maruel): Likely to not need this interface at all, merge NewCanvas()
// and FindObjects().
type Canvas interface {
	// FindObjects returns all the objects found.
	FindObjects() Objects
	Size() image.Point
}

// NewCanvas returns an initialized Canvas.
//
// Expands tabs to tabWidth as whitespace.
func NewCanvas(data []byte, tabWidth int) Canvas {
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
	return c
}

// PointsToCorners returns all the corners (points where there is a change of
// directionality) for a path.
func PointsToCorners(points []image.Point) []image.Point {
	l := len(points)
	if l == 1 || l == 2 {
		return points
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
	if isHorizontal(points[0], last) {
		if !horiz {
			out = append(out, last)
		}
	} else if isVertical(points[0], last) {
		if horiz {
			out = append(out, last)
		}
	} else {
		out = append(out, last)
	}
	/*
		if !IsClosed(points) {
			out = append(out, points[l-1])
		}
	*/
	return out
}

// IsClosed returns true if the set of points represents a closed path.
func IsClosed(points []image.Point) bool {
	if len(points) < 4 {
		return false
	}
	last := points[len(points)-1]
	return isHorizontal(points[0], last) || isVertical(points[0], last)
}

// Private details.

// canvas is the parsed source data.
type canvas struct {
	// (0,0) is top left.
	grid    []char
	visited []bool
	size    image.Point
}

func (c *canvas) FindObjects() Objects {
	var out Objects
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
				out = append(out, objs...)
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
				out = append(out, obj)
			}
		}
	}

	sort.Sort(out)
	return out
}

func (c *canvas) Size() image.Point {
	return c.size
}

// scanPath tries to complete one or multiple path or box starting with the
// partial path. It recursively calls itself when it finds multiple unvisited
// out-going paths.
func (c *canvas) scanPath(points []image.Point) Objects {
	var out Objects
	next := c.next(points[len(points)-1])
	if len(next) == 0 {
		// TODO(maruel): Determine if open. In particular in case of adjascent
		// closedpaths sharing a side.
		return Objects{&openPath{objectBase{points}}}
	}
	for _, n := range next {
		// Go depth first instead of bread first, this makes it workable for closed
		// path.
		if c.isVisited(n) {
			// TODO(maruel): Closed path.
			continue
		}
		c.visit(n)
		p2 := make([]image.Point, len(points)+1)
		copy(p2, points)
		p2[len(p2)-1] = n
		out = append(out, c.scanPath(p2)...)
	}
	return out
}

// next returns the next points that can be used to make progress.
//
// Look at top, left, right, down, skipping visited points and returns all the
// possibilities.
func (c *canvas) next(pos image.Point) []image.Point {
	var out []image.Point
	if !c.isVisited(pos) {
		panic("Internal error")
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
	t := &text{text: []rune{rune(c.at(start))}}
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
		t.text = append(t.text, rune(ch))
	}
	// TrimRight space.
	for len(t.text) != 0 && unicode.IsSpace(t.text[len(t.text)-1]) {
		t.text = t.text[:len(t.text)-1]
	}

	t.points = make([]image.Point, len(t.text))
	cur = start
	for i := 0; i < len(t.text); i++ {
		cur.X = start.X + i
		t.points[i] = cur
	}
	return t
}

func (c *canvas) at(p image.Point) char {
	return c.grid[p.Y*c.size.X+p.X]
}

func (c *canvas) isVisited(p image.Point) bool {
	return c.visited[p.Y*c.size.X+p.X]
}

func (c *canvas) visit(p image.Point) {
	c.visited[p.Y*c.size.X+p.X] = true
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

// objectBase is the common code between path and box.
type objectBase struct {
	// points always starts with the top most, then left most point, starting to
	// the right.
	points []image.Point
}

func (l *objectBase) Points() []image.Point {
	return l.points
}

func (l *objectBase) Text() []rune {
	return nil
}

// openPath is an open line. Likely a line between two closed paths (boxes).
type openPath struct {
	objectBase
}

func (p *openPath) String() string {
	return fmt.Sprintf("Path{%s}", p.points[0])
}

// closedPath is a closed path, e.g. a polygon, like a rectangle.
type closedPath struct {
	objectBase
}

func (b *closedPath) String() string {
	return fmt.Sprintf("Path{%s}", b.points[0])
}

type text struct {
	objectBase
	text []rune
}

func (t *text) String() string {
	return fmt.Sprintf("Text{%s %q}", t.points[0], string(t.text))
}

func (t *text) Text() []rune {
	return t.text
}

type char rune

func (c char) isTextStart() bool {
	r := rune(c)
	return unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSymbol(r)
}

func (c char) isTextCont() bool {
	r := rune(c)
	return unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSymbol(r) || unicode.IsSpace(r)
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
