// Copyright 2012 - 2015 The ASCIIToSVG Contributors
// All rights reserved.

package asciitosvg

import (
	"bytes"
	"image"
	"sync"
	"unicode"
	"unicode/utf8"
)

type Point image.Point

// Canvas is the parsed source data.
type Canvas struct {
	rawData []byte
	grid    [][]char
	size    Point
	ch      chan []Point
}

// NewCanvas returns an initialized Canvas.
func NewCanvas(data []byte) *Canvas {
	c := &Canvas{}

	c.rawData = data
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

	c.ch = make(chan []Point)

	return c
}

var wg sync.WaitGroup

type direction int

const (
	up direction = iota
	down
	left
	right
)

func (c *Canvas) runTrace(dir direction, cur Point, path []Point) {
	wg.Add(1)
	cpath := make([]Point, len(path))
	copy(cpath, path)
	go c.tracePolygon(dir, cur, cpath)
}

func (p *Point) move(dir direction) {
	switch dir {
	case up:
		p.Y--
	case down:
		p.Y++
	case left:
		p.X--
	case right:
		p.X++
	}
}

// TODO(maruel): Migrate below.
func (c *Canvas) tracePolygon(dir direction, cur Point, path []Point) {
	defer wg.Done()

	for {
		// Ignore polygons where we started from any point other than the top left
		if cur.Y < path[0].Y {
			return
		}

		if c.at(cur).isCorner() {
			if cur == path[0] {
				for _, pt := range path {
					if pt.Y < cur.Y || (pt.Y == cur.Y && pt.X < cur.X) {
						return
					}
				}
				c.ch <- path
				return
			}

			for _, pt := range path {
				// If we cycled without getting to our original point, bail.
				if pt == cur {
					return
				}
			}

			path = append(path, cur)

			if dir == right || dir == left {
				// Moving right, we can move up or down
				if c.canUp(cur) && c.at(c.above(cur)).isVertical() {
					c.runTrace(up, c.above(cur), path)
				}
				if c.canDown(cur) && c.at(c.below(cur)).isVertical() {
					c.runTrace(down, c.below(cur), path)
				}
			} else if dir == up || dir == down {
				// Moving down, we can move left or right
				if c.canLeft(cur) && c.at(c.leftOf(cur)).isHorizontal() {
					c.runTrace(left, c.leftOf(cur), path)
				}
				if c.canRight(cur) && c.at(c.rightOf(cur)).isHorizontal() {
					c.runTrace(right, c.rightOf(cur), path)
				}
			}

			return
		} else if c.canMove(dir, cur) {
			cur.move(dir)
		} else {
			return
		}
	}
}

func (c *Canvas) FindPolygons() Objects {
	p := Point{}
	for y := 0; y < c.size.Y; y++ {
		p.Y = y
		for x := 0; x < c.size.X; x++ {
			p.X = x
			ch := c.at(p)
			if ch.isCorner() && c.at(c.rightOf(p)).isHorizontal() {
				c.runTrace(right, Point{X: p.X + 1, Y: p.Y}, []Point{p})
			}
		}
	}

	go func() {
		wg.Wait()
		close(c.ch)
	}()

	objs := make(Objects, 0)
	for {
		v := <-c.ch
		if v == nil {
			return objs
		}

		objs = append(objs, v)
	}

	return nil
}

func (c *Canvas) at(p Point) char {
	return c.grid[p.Y][p.X]
}

func (c *Canvas) above(p Point) Point {
	p.Y--
	return p
}

func (c *Canvas) below(p Point) Point {
	p.Y++
	return p
}

func (c *Canvas) leftOf(p Point) Point {
	p.X--
	return p
}

func (c *Canvas) rightOf(p Point) Point {
	p.X++
	return p
}

func (c *Canvas) canLeft(p Point) bool {
	return p.X > 0
}

func (c *Canvas) canRight(p Point) bool {
	return p.X < c.size.X-1
}

func (c *Canvas) canUp(p Point) bool {
	return p.Y > 0
}

func (c *Canvas) canDown(p Point) bool {
	return p.Y < c.size.Y-1
}

func (c *Canvas) canMove(dir direction, p Point) bool {
	if c.at(p).isHorizontal() {
		if dir == right && c.canRight(p) {
			return true
		}
		if dir == left && c.canLeft(p) {
			return true
		}
	} else if c.at(p).isVertical() {
		if dir == up && c.canUp(p) {
			return true
		}
		if dir == down && c.canDown(p) {
			return true
		}
	}
	return false
}

type Object struct {
}

// Boxes is scaffolding.
type Objects [][]Point

// ToSVG is scaffolding.
func (b Object) ToSVG(noBlur bool, font string, scaleX, scaleY int) []byte {
	return nil
}

// Private details.

type char rune

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

func (c char) isTextStart() bool {
	r := rune(c)
	return unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSymbol(r)
}

func (c char) isTextCont() bool {
	r := rune(c)
	return unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSymbol(r) || unicode.IsSpace(r)
}
