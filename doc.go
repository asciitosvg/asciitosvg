// Copyright 2012 - 2018 The ASCIIToSVG Contributors
// All rights reserved.

// Package asciitosvg provides functionality for parsing ASCII diagrams. It supports diagrams
// containing UTF-8 content, custom styling of polygons, line segments, and text.
//
// The main interface to the library is through the Canvas. A Canvas is parsed from a byte slice
// representing the diagram. The byte slice is interpreted as a newline-delimited file, each line
// representing a row of the diagram. Tabs within the diagram are expanded to spaces based on a
// specified tab width.
//
// Example usage:
//
//     import (
//         "fmt"
//         "io"
//
//         a2s "github.com/asciitosvg/asciitosvg"
//     )
//
//     ...
//
//         c, err := a2s.NewCanvas(diagram, 8)
//         if err != nil {
//             fmt.Printf("Couldn't create a Canvas: %s\n", err)
//         }
//         svg := a2s.CanvasToSVG(c, false, "", 16, 9)
//         written, err := fd.Write(svg)
//
//     ...
package asciitosvg
