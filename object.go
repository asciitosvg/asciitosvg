// Copyright 2012 - 2018 The ASCIIToSVG Contributors
// All rights reserved.

package asciitosvg

import "fmt"

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

			for _, corner := range o.corners {
				if corner.X == p.X && corner.Y == p.Y && c.at(p).isRoundedCorner() {
					o.points[i].Hint = RoundedCorner
				}
			}
		}
		o.text[i] = rune(c.at(p))
	}
}

// objects implements a sortable collection of Object interfaces.
type objects []Object

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
