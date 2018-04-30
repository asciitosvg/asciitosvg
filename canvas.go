// Copyright 2012 - 2018 The ASCIIToSVG Contributors
// All rights reserved.

package asciitosvg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"regexp"
	"sort"
	"strconv"
	"unicode/utf8"
)

// Object is an interface for working with open paths (lines), closed paths (polygons), or text.
type Object interface {
	fmt.Stringer
	// Points returns all the points occupied by this Object. Every object has at least one point,
	// and all points are both in-order and contiguous.
	Points() []Point
	// HasPoint returns true if the object contains the supplied Point coordinates.
	HasPoint(Point) bool
	// Corners returns all the corners (change of direction) along the path.
	Corners() []Point
	// IsClosed is true if the object is composed of a closed path.
	IsClosed() bool
	// IsDashed is true if this object is a path object, and lines should be drawn dashed.
	IsDashed() bool
	// IsText returns true if the object is textual and does not represent a path.
	IsText() bool
	// Text returns the text associated with this Object if textual, and nil otherwise.
	Text() []rune
	// SetTag sets an options tag on this Object so the renderer may look up options.
	SetTag(string)
	// Tag returns the tag of this object, if any.
	Tag() string
}

// Canvas provides methods for returning objects from an underlying textual grid.
type Canvas interface {
	// A canvas has an underlying visual representation. The fmt.Stringer interface for this
	// interface provides a view into the underlying grid.
	fmt.Stringer
	// Objects returns all the objects found in the underlying grid.
	Objects() []Object
	// TextContainer returns the Object that contains the supplied Text object, if any.
	TextContainer(s, e Point) Object
	// Size returns the visual dimensions of the Canvas.
	Size() image.Point
	// Options returns a map of options to apply to Objects based on the object's tag. This
	// maps tag name to a map of option names to options.
	Options() map[string]map[string]interface{}
}

// NewCanvas returns a new Canvas, initialized from the provided data. If tabWidth is set to a non-negative
// value, that value will be used to convert tabs to spaces within the grid.
func NewCanvas(data []byte, tabWidth int) (Canvas, error) {
	c := &canvas{options: make(map[string]map[string]interface{})}

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
	options map[string]map[string]interface{}
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

func (c *canvas) Options() map[string]map[string]interface{} {
	return c.options
}

func (c *canvas) TextContainer(start, end Point) Object {
	maxTL := Point{X: -1, Y: -1}

	var target Object
	for _, o := range c.objects {
		// Text can't be in an open path, or another text object.
		if !o.IsClosed() {
			continue
		}

		// If the object can fully encapsulate this text tag, mark it as the
		// target. The maxTL check allows us to find the most specific object
		// in the case of nested polygons.
		if o.HasPoint(start) && o.HasPoint(end) && o.Corners()[0].X > maxTL.X && o.Corners()[0].Y > maxTL.Y {
			target = o
			maxTL.X = o.Corners()[0].X
			maxTL.Y = o.Corners()[0].Y
		}
	}

	return target
}

// A RenderHint suggests ways the SVG renderer may appropriately represent this point.
type RenderHint int

const (
	// No hints are provided for this point.
	None RenderHint = iota
	// This point represents a corner that should be rounded.
	RoundedCorner
	// This point should have an SVG marker-start attribute associated with it.
	StartMarker
	// This point should have an SVG marker-end attribute associated with it.
	EndMarker
	// This is a path component that should have a strikethrough at this point.
	Tick
	// This is a path component that should have a dot at this point.
	Dot
)

type Point struct {
	X    int
	Y    int
	Hint RenderHint
}

func (p Point) String() string {
	return fmt.Sprintf("(%d,%d)", p.X, p.Y)
}

// findObjects finds all objects (lines, polygons, and text) within the underlying grid.
func (c *canvas) findObjects() {
	p := Point{}

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
				objs := c.scanPath([]Point{p})
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

				// scanText will return nil if the text at this area is simply
				// setting options on a container object.
				if obj == nil {
					continue
				}
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
func (c *canvas) scanPath(points []Point) objects {
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
		return append(r, c.scanPath([]Point{cur})...)
	}

	// We scan depth-first instead of breadth-first, making it possible to find a
	// closed path.
	var objs objects
	for _, n := range next {
		if c.isVisited(n) {
			continue
		}
		c.visit(n)
		p2 := make([]Point, len(points)+1)
		copy(p2, points)
		p2[len(p2)-1] = n
		objs = append(objs, c.scanPath(p2)...)
	}
	return objs
}

// The next returns the points that can be used to make progress, scanning (in order) horizontal
// progress to the left or right, vertical progress above or below, or diagonal progress to NW,
// NE, SW, and SE. It skips any points already visited, and returns all of the possible progress
// points.
func (c *canvas) next(pos Point) []Point {
	// Our caller must have called c.visit prior to calling this function.
	if !c.isVisited(pos) {
		panic(fmt.Errorf("internal error; revisiting %s", pos))
	}

	var out []Point

	ch := c.at(pos)
	if ch.canHorizontal() {
		nextHorizontal := func(p Point) {
			if !c.isVisited(p) && c.at(p).canHorizontal() {
				out = append(out, p)
			}
		}
		if c.canLeft(pos) {
			n := pos
			n.X--
			nextHorizontal(n)
		}
		if c.canRight(pos) {
			n := pos
			n.X++
			nextHorizontal(n)
		}
	}
	if ch.canVertical() {
		nextVertical := func(p Point) {
			if !c.isVisited(p) && c.at(p).canVertical() {
				out = append(out, p)
			}
		}
		if c.canUp(pos) {
			n := pos
			n.Y--
			nextVertical(n)
		}
		if c.canDown(pos) {
			n := pos
			n.Y++
			nextVertical(n)
		}
	}
	if c.canDiagonal(pos) {
		nextDiagonal := func(from, to Point) {
			if !c.isVisited(to) && c.at(to).canDiagonalFrom(c.at(from)) {
				out = append(out, to)
			}
		}
		if c.canUp(pos) {
			if c.canLeft(pos) {
				n := pos
				n.X--
				n.Y--
				nextDiagonal(pos, n)
			}
			if c.canRight(pos) {
				n := pos
				n.X++
				n.Y--
				nextDiagonal(pos, n)
			}
		}
		if c.canDown(pos) {
			if c.canLeft(pos) {
				n := pos
				n.X--
				n.Y++
				nextDiagonal(pos, n)
			}
			if c.canRight(pos) {
				n := pos
				n.X++
				n.Y++
				nextDiagonal(pos, n)
			}
		}
	}

	return out
}

// Used for matching [X, Y]: {...} tag definitions. These definitions target specific objects.
var objTagRE = regexp.MustCompile(`(\d+)\s*,\s*(\d+)$`)

// scanText extracts a line of text.
func (c *canvas) scanText(start Point) Object {
	obj := &object{points: []Point{start}, isText: true}
	whiteSpaceStreak := 0
	cur, end := start, start

	tagged := 0
	tag := []rune{}
	tagDef := []rune{}

	for c.canRight(cur) {
		if cur.X == start.X && c.at(cur).isObjectStartTag() {
			tagged++
		} else if cur.X > start.X && c.at(cur).isObjectEndTag() {
			end = cur
			tagged++
		}

		cur.X++
		if c.isVisited(cur) {
			// If the point is already visited, we hit a polygon or a line.
			break
		}
		ch := c.at(cur)
		if !ch.isTextCont() {
			break
		}
		if tagged == 0 && ch.isSpace() {
			whiteSpaceStreak++
			// Stop when we see 3 consecutive whitespace points.
			if whiteSpaceStreak > 2 {
				break
			}
		} else {
			whiteSpaceStreak = 0
		}

		switch tagged {
		case 1:
			if !c.at(cur).isObjectEndTag() {
				tag = append(tag, rune(ch))
			}
		case 2:
			if c.at(cur).isTagDefinitionSeparator() {
				tagged++
			} else {
				tagged = -1
			}
		case 3:
			tagDef = append(tagDef, rune(ch))
		}

		obj.points = append(obj.points, cur)
	}

	// If we found a start and end tag marker, we either need to assign the tag to the object,
	// or we need to assign the specified options to the global canvas option space.
	if tagged == 2 {
		t := string(tag)
		if container := c.TextContainer(start, end); container != nil {
			container.SetTag(t)
		}

		// The tag applies to the text object as well so that properties like
		// a2s:label can be set.
		obj.SetTag(t)
	} else if tagged == 3 {
		t := string(tag)

		// A tag definition targeting an object will not be found within any object; we need
		// to do that calculation here.
		if matches := objTagRE.FindStringSubmatch(t); matches != nil {
			if targetX, err := strconv.ParseInt(matches[1], 10, 0); err == nil {
				if targetY, err := strconv.ParseInt(matches[2], 10, 0); err == nil {
					for i, o := range c.objects {
						corner := o.Corners()[0]
						if corner.X == int(targetX) && corner.Y == int(targetY) {
							c.objects[i].SetTag(t)
							break
						}
					}
				}
			}
		}
		// This is a tag definition. Parse the JSON and assign the options to the canvas.
		var m interface{}
		def := []byte(string(tagDef))
		if err := json.Unmarshal(def, &m); err != nil {
			// TODO(dhobsd): Gross.
			panic(err)
		}

		// The tag applies to the reference object as well, so that properties like
		// a2s:delref can be set.
		obj.SetTag(t)
		c.options[t] = m.(map[string]interface{})
	}

	// Trim the right side of the text object.
	for len(obj.points) != 0 && c.at(obj.points[len(obj.points)-1]).isSpace() {
		obj.points = obj.points[:len(obj.points)-1]
	}

	obj.seal(c)
	return obj
}

func (c *canvas) at(p Point) char {
	return c.grid[p.Y*c.size.X+p.X]
}

func (c *canvas) isVisited(p Point) bool {
	return c.visited[p.Y*c.size.X+p.X]
}

func (c *canvas) visit(p Point) {
	// TODO(dhobsd): Change code to ensure that visit() is called once and only
	// once per point.
	c.visited[p.Y*c.size.X+p.X] = true
}

func (c *canvas) unvisit(p Point) {
	o := p.Y*c.size.X + p.X
	if !c.visited[o] {
		panic(fmt.Errorf("internal error: point %+v never visited", p))
	}
	c.visited[o] = false
}

func (c *canvas) canLeft(p Point) bool {
	return p.X > 0
}

func (c *canvas) canRight(p Point) bool {
	return p.X < c.size.X-1
}

func (c *canvas) canUp(p Point) bool {
	return p.Y > 0
}

func (c *canvas) canDown(p Point) bool {
	return p.Y < c.size.Y-1
}

func (c *canvas) canDiagonal(p Point) bool {
	return (c.canLeft(p) || c.canRight(p)) && (c.canUp(p) || c.canDown(p))
}

// object implements Object and represents one of an open path, a closed path, or text.
type object struct {
	// points always starts with the top most, then left most point, proceeding to the right.
	points   []Point
	isText   bool
	text     []rune
	corners  []Point
	isClosed bool
	isDashed bool
	tag      string
}

func (o *object) Points() []Point {
	return o.points
}

func (o *object) Corners() []Point {
	return o.corners
}

func (o *object) IsClosed() bool {
	return o.isClosed
}

func (o *object) IsText() bool {
	return o.isText
}

func (o *object) IsDashed() bool {
	return o.isDashed
}

func (o *object) Text() []rune {
	return o.text
}

func (o *object) SetTag(s string) {
	o.tag = s
}

func (o *object) Tag() string {
	return o.tag
}

func (o *object) String() string {
	if o.IsText() {
		return fmt.Sprintf("Text{%s %q}", o.points[0], string(o.text))
	}
	return fmt.Sprintf("Path{%v}", o.points)
}

// HasPoint determines whether the supplied point lives inside the object. Since we support complex
// convex and concave polygons, we need to do a full point-in-polygon test. The algorithm implemented
// comes from the more efficient, less-clever version at http://alienryderflex.com/polygon/.
func (o *object) HasPoint(p Point) bool {
	hasPoint := false
	ncorners := len(o.corners)
	j := ncorners - 1
	for i := 0; i < ncorners; i++ {
		if (o.corners[i].Y < p.Y && o.corners[j].Y >= p.Y || o.corners[j].Y < p.Y && o.corners[i].Y >= p.Y) && (o.corners[i].X <= p.X || o.corners[j].X <= p.X) {
			if o.corners[i].X+(p.Y-o.corners[i].Y)/(o.corners[j].Y-o.corners[i].Y)*(o.corners[j].X-o.corners[i].X) < p.X {
				hasPoint = !hasPoint
			}
		}
		j = i
	}

	return hasPoint
}

// seal finalizes the object, setting its text, its corners, and its various rendering hints.
func (o *object) seal(c *canvas) {
	if c.at(o.points[0]).isArrow() {
		o.points[0].Hint = StartMarker
	}

	if c.at(o.points[len(o.points)-1]).isArrow() {
		o.points[len(o.points)-1].Hint = EndMarker
	}

	o.corners, o.isClosed = pointsToCorners(o.points)
	o.text = make([]rune, len(o.points))

	for i, p := range o.points {
		if !o.IsText() {
			if c.at(p).isTick() {
				o.points[i].Hint = Tick
			} else if c.at(p).isDot() {
				o.points[i].Hint = Dot
			}

			if c.at(p).isDashed() {
				o.isDashed = true
			}

			// TODO(dhobsd): Only do this for corners.
			if c.at(p).isRoundedCorner() {
				o.points[i].Hint = RoundedCorner
			}
		}
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

// isHorizontal returns true if p1 and p2 are horizontally aligned.
func isHorizontal(p1, p2 Point) bool {
	d := p1.X - p2.X
	return d <= 1 && d >= -1 && p1.Y == p2.Y
}

// isVertical returns true if p1 and p2 are vertically aligned.
func isVertical(p1, p2 Point) bool {
	d := p1.Y - p2.Y
	return d <= 1 && d >= -1 && p1.X == p2.X
}

// The following functions return true when the diagonals are connected in various compass directions.
func isDiagonalSE(p1, p2 Point) bool {
	return p1.X-p2.X == -1 && p1.Y-p2.Y == -1
}
func isDiagonalSW(p1, p2 Point) bool {
	return p1.X-p2.X == 1 && p1.Y-p2.Y == -1
}
func isDiagonalNW(p1, p2 Point) bool {
	return p1.X-p2.X == 1 && p1.Y-p2.Y == 1
}
func isDiagonalNE(p1, p2 Point) bool {
	return p1.X-p2.X == -1 && p1.Y-p2.Y == 1
}

const (
	dirNone = iota // No directionality
	dirH           // Horizontal
	dirV           // Vertical
	dirSE          // South-East
	dirSW          // South-West
	dirNW          // North-West
	dirNE          // North-East
)

// pointsToCorners returns all the corners (points at which there is a change of directionality) for
// a path. It additionally returns a truth value indicating whether the points supplied indicate a
// closed path.
func pointsToCorners(points []Point) ([]Point, bool) {
	l := len(points)
	// A path containing fewer than 3 points can neither be closed, nor change direction.
	if l < 3 {
		return points, false
	}
	out := []Point{points[0]}

	dir := dirNone
	if isHorizontal(points[0], points[1]) {
		dir = dirH
	} else if isVertical(points[0], points[1]) {
		dir = dirV
	} else if isDiagonalSE(points[0], points[1]) {
		dir = dirSE
	} else if isDiagonalSW(points[0], points[1]) {
		dir = dirSW
	} else if isDiagonalNW(points[0], points[1]) {
		dir = dirNW
	} else if isDiagonalNE(points[0], points[1]) {
		dir = dirNE
	} else {
		panic(fmt.Errorf("discontiguous points: %+v", points))
	}

	// Starting from the third point, check to see if the directionality between points P and
	// P-1 has changed.
	for i := 2; i < l; i++ {
		cornerFunc := func(idx, newDir int) {
			if dir != newDir {
				out = append(out, points[idx-1])
				dir = newDir
			}
		}
		if isHorizontal(points[i-1], points[i]) {
			cornerFunc(i, dirH)
		} else if isVertical(points[i-1], points[i]) {
			cornerFunc(i, dirV)
		} else if isDiagonalSE(points[i-1], points[i]) {
			cornerFunc(i, dirSE)
		} else if isDiagonalSW(points[i-1], points[i]) {
			cornerFunc(i, dirSW)
		} else if isDiagonalNW(points[i-1], points[i]) {
			cornerFunc(i, dirNW)
		} else if isDiagonalNE(points[i-1], points[i]) {
			cornerFunc(i, dirNE)
		} else {
			panic(fmt.Errorf("discontiguous points: %+v", points))
		}
	}

	// Check if the points indicate a closed path. If not, append the last point.
	last := points[l-1]
	closed := true
	closedFunc := func(newDir int) {
		if dir != newDir {
			closed = false
			out = append(out, last)
		}
	}
	if isHorizontal(points[0], last) {
		closedFunc(dirH)
	} else if isVertical(points[0], last) {
		closedFunc(dirV)
	} else if isDiagonalNE(last, points[0]) {
		closedFunc(dirNE)
	} else {
		// Note: we'll always find any closed polygon from its top-left-most point. If it
		// is closed, it must be closed in the north-easterly direction, thus we don't test
		// for any other types of polygone closure.
		closed = false
		out = append(out, last)
	}

	return out, closed
}
