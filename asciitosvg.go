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

// Canvas is the parsed source data.
type Canvas struct {
	grid [][]char
	size image.Point
}

// NewCanvas returns an initialized Canvas.
func NewCanvas(data []byte) *Canvas {
	c := &Canvas{}
	lines := bytes.Split(data, []byte("\n"))
	c.size.Y = len(lines)
	for _, line := range lines {
		if i := utf8.RuneCount(line); i > c.size.X {
			c.size.X = i
		}
	}
	for _, line := range lines {
		t := make([]char, c.size.X)
		i := 0
		for len(line) > 0 {
			r, l := utf8.DecodeRune(line)
			t[i] = char(r)
			i++
			line = line[l:]
		}
		for ; i < c.size.X; i++ {
			t[i] = ' '
		}
		c.grid = append(c.grid, t)
	}
	return c
}

// FindObjects returns all the objects found.
func (c *Canvas) FindObjects() Objects {
	var out Objects
	p := image.Point{}
	for y := 0; y < c.size.Y; y++ {
		p.Y = y
		for x := 0; x < c.size.X; x++ {
			p.X = x
			ch := c.at(p)
			if ch.isCorner() {
				out = append(out, c.scanLineSet(p)...)
			}
		}
	}
	// TODO(maruel): Trim redundant boxes and paths.

	for y := 0; y < c.size.Y; y++ {
		p.Y = y
		for x := 0; x < c.size.X; x++ {
			p.X = x
			ch := c.at(p)
			if !out.IsVisited(p) && ch.isTextStart() {
				// TODO(maruel): Limit to the zones.
				out = append(out, c.scanText(p))
			}
		}
	}
	// TODO(maruel): Trim the text that is not text.
	sort.Sort(out)
	return out
}

// next returns the next paths that can be used to make progress.
//
// Look at top, left, right, down, skipping 'from' and returns all the
// possibilities.
func (c *Canvas) next(pos, from image.Point) []image.Point {
	var out []image.Point
	ch := c.at(pos)
	if ch.canHorizontal() {
		if c.canLeft(pos) {
			n := pos
			n.X--
			if n != from && c.at(n).canHorizontal() {
				out = append(out, n)
			}
		}
		if c.canRight(pos) {
			n := pos
			n.X++
			if n != from && c.at(n).canHorizontal() {
				out = append(out, n)
			}
		}
	}
	if ch.canVertical() {
		if c.canUp(pos) {
			n := pos
			n.Y--
			if n != from && c.at(n).canVertical() {
				out = append(out, n)
			}
		}
		if c.canRight(pos) {
			n := pos
			n.Y++
			if n != from && c.at(n).canVertical() {
				out = append(out, n)
			}
		}
	}
	return out
}

// scanLineSet tries to find one or multiple Path or Box starting at point
// start.
func (c *Canvas) scanLineSet(start image.Point) Objects {
	var out Objects
	// By definition, it can only find a point down or right. Ignore points up and
	// left. So look up manually instead of isuing c.next(). We also know that
	// start is a corner.
	for _, n := range c.next(start, start) {
		out = append(out, c.branch(&lineSet{points: []image.Point{start}}, n)...)
	}
	return out
}

// branch tries to complete one or multiple Path or Box starting with the
// partial path.
func (c *Canvas) branch(l *lineSet, start image.Point) Objects {
	// Calls itself recursively and returns every path or boxes found.
	var out Objects
	cur := start
	prev := cur
	for {
		next := c.next(cur, prev)
		if len(next) == 0 {
			break
		}
		for _, n := range next {
			if !l.IsVisited(n) {
				prev = cur
				cur = n
				c.branch(l, cur)
			}
		}
	}
	return out
}

func (c *Canvas) scanText(start image.Point) Object {
	t := &text{p: start, text: []rune{rune(c.at(start))}}
	for c.canRight(start) {
		start.X++
		ch := c.at(start)
		if !ch.isTextCont() {
			break
		}
		t.text = append(t.text, rune(ch))
	}
	for len(t.text) != 0 && unicode.IsSpace(t.text[len(t.text)-1]) {
		t.text = t.text[:len(t.text)-1]
	}
	return t
}

func (c *Canvas) at(p image.Point) char {
	return c.grid[p.Y][p.X]
}

func (c *Canvas) canLeft(p image.Point) bool {
	return p.X > 0
}

func (c *Canvas) canRight(p image.Point) bool {
	return p.X < c.size.X-1
}

func (c *Canvas) canUp(p image.Point) bool {
	return p.Y > 0
}

func (c *Canvas) canDown(p image.Point) bool {
	return p.Y < c.size.Y-1
}

// Object is either a path, a box or text.
type Object interface {
	fmt.Stringer
	TopLeft() image.Point
	IsText() bool
	IsVisited(p image.Point) bool
	Text() []rune
}

// Objects is all objects found.
type Objects []Object

func (o Objects) Len() int      { return len(o) }
func (o Objects) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
func (o Objects) Less(i, j int) bool {
	l := o[i]
	r := o[j]
	lt := l.IsText()
	rt := r.IsText()
	if lt != rt {
		return rt
	}
	lp := l.TopLeft()
	rp := r.TopLeft()
	if lp.X != rp.X {
		return lp.X < rp.X
	}
	return lp.Y < rp.Y
}

// IsVisited returns true if any Object returns true for IsVisited().
func (o Objects) IsVisited(p image.Point) bool {
	// Brute force.
	for _, i := range o {
		if i.IsVisited(p) {
			return true
		}
	}
	return false
}

// ToSVG is scaffolding.
func (o Objects) ToSVG(noBlur bool, font string, scaleX, scaleY int) []byte {
	return nil
}

// Private details.

// lineSet is the common code between path and box.
type lineSet struct {
	// points always starts with the top most, then left most point, starting to
	// the right.
	points  []image.Point
	corners []image.Point
}

func (l *lineSet) TopLeft() image.Point {
	return l.points[0]
}

func (l *lineSet) IsText() bool {
	return false
}

func (l *lineSet) Text() []rune {
	return nil
}

func (l *lineSet) IsVisited(p image.Point) bool {
	// Brute force.
	for _, point := range l.points {
		if p == point {
			return true
		}
	}
	return false
}

// path is an open line.
type path struct {
	lineSet
}

func (p *path) String() string {
	return fmt.Sprintf("Path{%s}", p.points[0])
}

// box is a closed Path.
type box struct {
	lineSet
}

func (b *box) String() string {
	return fmt.Sprintf("Path{%s}", b.points[0])
}

type text struct {
	p    image.Point
	text []rune
}

func (t *text) String() string {
	return fmt.Sprintf("Text{%s %q}", t.p, string(t.text))
}

func (t *text) TopLeft() image.Point {
	return t.p
}

func (t *text) IsText() bool {
	return true
}

func (t *text) IsVisited(p image.Point) bool {
	if p.Y == t.p.Y {
		d := p.X - t.p.X
		if d < 0 {
			return false
		}
		return d >= 0 && d < len(t.text)
	}
	return false
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

func (c char) isCorner() bool {
	return c == '.' || c == '\'' || c == '+'
}

func (c char) isHorizontal() bool {
	return c == '-'
}

func (c char) isVertical() bool {
	return c == '|'
}

func (c char) canVertical() bool {
	return c.isVertical() || c.isCorner()
}

func (c char) canHorizontal() bool {
	return c.isHorizontal() || c.isCorner()
}
