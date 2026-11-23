// Copyright 2012 - 2018 The ASCIIToSVG Contributors
// All rights reserved.

package asciitosvg

import "fmt"

// A RenderHint suggests ways the SVG renderer may appropriately represent this point.
type RenderHint int

const (
	// None indicates no hints are provided for this point.
	None RenderHint = iota
	// RoundedCorner indicates the renderer should smooth corners on this path.
	RoundedCorner
	// StartMarker indicates this point should have an SVG marker-start attribute.
	StartMarker
	// EndMarker indicates this point should have an SVG marker-end attribute.
	EndMarker
	// Tick indicates the renderer should mark a tick in the path at this point.
	Tick
	// Dot indicates the renderer should insert a filled dot in the path at this point.
	Dot
	// StartMarkerForward indicates the start point has an arrow pointing forward along the path.
	StartMarkerForward
	// EndMarkerReverse indicates the end point has an arrow pointing backward along the path.
	EndMarkerReverse
	// RoundedCuspDown indicates an unambiguous mid-edge '.' on a horizontal run that dips downward.
	RoundedCuspDown
	// RoundedCuspUp indicates an unambiguous mid-edge '\'' on a horizontal run that rises upward.
	RoundedCuspUp
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
