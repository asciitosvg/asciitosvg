// Copyright 2012 - 2018 The ASCIIToSVG Contributors
// All rights reserved.

package asciitosvg

import "fmt"

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

// A Point is an X,Y coordinate in the diagram's grid. The grid represents (0, 0) as the top-left
// of the diagram. The Point also provides hints to the renderer as to how it should be interpreted.
type Point struct {
	// The X coordinate of this point.
	X int
	// The Y coordinate of this point.
	Y int
	// Hints for the renderer.
	Hint RenderHint
}

// String implements fmt.Stringer on Point.
func (p Point) String() string {
	return fmt.Sprintf("(%d,%d)", p.X, p.Y)
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
