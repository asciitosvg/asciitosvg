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
	isTagDef bool
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
	if !o.isClosed {
		return false
	}
	hasPoint := false
	ncorners := len(o.corners)
	j := ncorners - 1
	for i := 0; i < ncorners; i++ {
		ci := o.corners[i]
		cj := o.corners[j]
		if (ci.Y < p.Y && cj.Y >= p.Y || cj.Y < p.Y && ci.Y >= p.Y) && (ci.X <= p.X || cj.X <= p.X) {
			if float64(ci.X)+float64(p.Y-ci.Y)/float64(cj.Y-ci.Y)*float64(cj.X-ci.X) < float64(p.X) {
				hasPoint = !hasPoint
			}
		}
		j = i
	}

	return hasPoint
}

func arrowDir(c char) (int, int) {
	switch c {
	case '<':
		return -1, 0
	case '>':
		return 1, 0
	case '^':
		return 0, -1
	case 'v':
		return 0, 1
	default:
		return 0, 0
	}
}

// seal finalizes the object, setting its text, its corners, and its various rendering hints.
func (o *object) seal(c *canvas) {
	corners, closed := pointsToCorners(o.points)
	o.sealWith(c, corners, closed)
}

func (o *object) sealPath(c *canvas, closed bool) {
	corners := pointsToCornersWithClosed(o.points, closed)
	o.sealWith(c, corners, closed)
}

func (o *object) sealWith(c *canvas, corners []Point, closed bool) {
	if !o.isText && len(o.points) >= 2 {
		if ch := c.at(o.points[0]); ch.isArrow() {
			ax, ay := arrowDir(ch)
			dx := o.points[1].X - o.points[0].X
			dy := o.points[1].Y - o.points[0].Y
			if ax*dx+ay*dy > 0 {
				o.points[0].Hint = StartMarkerForward
			} else {
				o.points[0].Hint = StartMarker
			}
		}

		last := len(o.points) - 1
		if ch := c.at(o.points[last]); ch.isArrow() {
			ax, ay := arrowDir(ch)
			dx := o.points[last].X - o.points[last-1].X
			dy := o.points[last].Y - o.points[last-1].Y
			if ax*dx+ay*dy < 0 {
				o.points[last].Hint = EndMarkerReverse
			} else {
				o.points[last].Hint = EndMarker
			}
		}
		// Recompute corners if start/end hints changed so corners preserve marker hints.
		if o.points[0].Hint != None || o.points[last].Hint != None {
			corners = pointsToCornersWithClosed(o.points, closed)
		}
	}

	o.corners = corners
	o.isClosed = closed
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

			if c.at(p).isRoundedCorner() && (o.isClosed || (i > 0 && i < len(o.points)-1)) {
				for _, corner := range o.corners {
					if corner.X == p.X && corner.Y == p.Y {
						o.points[i].Hint = RoundedCorner
						break
					}
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
	dirE           // East (right)
	dirW           // West (left)
	dirS           // South (down)
	dirN           // North (up)
	dirSE          // South-East
	dirSW          // South-West
	dirNW          // North-West
	dirNE          // North-East
)

func stepDir(from, to Point) int {
	dx := to.X - from.X
	dy := to.Y - from.Y
	switch {
	case dx == 1 && dy == 0:
		return dirE
	case dx == -1 && dy == 0:
		return dirW
	case dx == 0 && dy == 1:
		return dirS
	case dx == 0 && dy == -1:
		return dirN
	case dx == 1 && dy == 1:
		return dirSE
	case dx == -1 && dy == 1:
		return dirSW
	case dx == -1 && dy == -1:
		return dirNW
	case dx == 1 && dy == -1:
		return dirNE
	default:
		return dirNone
	}
}

func signedArea2(points []Point) int {
	n := len(points)
	if n < 3 {
		return 0
	}
	area2 := 0
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		area2 += points[i].X*points[j].Y - points[j].X*points[i].Y
	}
	return area2
}

// pointsToCorners returns all the corners (points at which there is a change of directionality) for
// a path. It additionally returns a truth value indicating whether the points supplied indicate a
// closed path.
func pointsToCorners(points []Point) ([]Point, bool) {
	l := len(points)
	// A path containing fewer than 3 points can neither be closed, nor change direction.
	if l < 3 {
		return points, false
	}
	last := points[l-1]
	closingDir := stepDir(last, points[0])
	closed := false
	if closingDir != dirNone && (l >= 4 || closingDir == dirNE) {
		closed = true
	}
	return pointsToCornersWithClosed(points, closed), closed
}

func pointsToCornersWithClosed(points []Point, closed bool) []Point {
	l := len(points)
	if l < 3 {
		return points
	}
	out := []Point{points[0]}

	dir := stepDir(points[0], points[1])
	if dir == dirNone {
		panic(fmt.Errorf("discontiguous points: %+v", points))
	}

	// Starting from the third point, check to see if the directionality between points P and
	// P-1 has changed.
	for i := 2; i < l; i++ {
		newDir := stepDir(points[i-1], points[i])
		if newDir == dirNone {
			panic(fmt.Errorf("discontiguous points: %+v", points))
		}
		if dir != newDir {
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
	if closingDir == dirNone || dir != closingDir {
		out = append(out, last)
	}
	return out
}
