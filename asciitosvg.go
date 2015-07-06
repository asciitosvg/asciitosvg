// Copyright 2012 - 2015 The ASCIIToSVG Contributors
// All rights reserved.

package asciitosvg

import (
	"bytes"
	"image"
	"sync"
	"unicode/utf8"
)

type Canvas struct {
	rawData []byte
	grid    [][]rune
	size    image.Point
}

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
		t := make([]rune, c.size.X)
		i := 0
		for len(line) > 0 {
			r, l := utf8.DecodeRune(line)
			t[i] = r
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

func isCorner(char rune) bool {
	return char == '.' || char == '\'' || char == '+'
}

func (c *Canvas) scanBox(wg *sync.WaitGroup, p []image.Point, row, col, rowInc, colInc int) {
	defer wg.Done()

	for {
		// Avoid going off the board
		if row < 0 || col < 0 || row >= c.size.Y || col >= c.size.X {
			return
		}

		// If we find a corner, try to follow any lines back to our starting point. If
		// we aren't a corner, we just keep going in our current direction as long as
		// our lines don't run out.
		if isCorner(c.grid[row][col]) {
			if row == p[0].Y && col == p[0].X {
				// Found our original point via the path in p
				return
			}

			for _, pt := range p {
				// If we cycled without getting to our original point, bail.
				if pt.X == col && pt.Y == row {
					return
				}
			}

			p = append(p, image.Point{X: col, Y: row})

			if rowInc == 0 && colInc == 1 {
				// Moving right, we can move up or down
				if row < c.size.Y-1 && c.grid[row+1][col] == '|' {
					wg.Add(1)
					go c.scanBox(wg, p, row+1, col, 1, 0)
				}
				if row > p[0].Y && c.grid[row-1][col] == '|' {
					wg.Add(1)
					go c.scanBox(wg, p, row-1, col, -1, 0)
				}
			} else if rowInc == 1 && colInc == 0 {
				// Moving down, we can move left or right
				if c.grid[row][col+1] == '-' {
					wg.Add(1)
					go c.scanBox(wg, p, row, col+1, 0, 1)
				}
				if col > 0 && c.grid[row][col-1] == '-' {
					wg.Add(1)
					go c.scanBox(wg, p, row, col-1, 0, -1)
				}
			} else if rowInc == 0 && colInc == -1 {
				// Moving left, we can move up or down
				if row > 0 && c.grid[row-1][col] == '|' {
					wg.Add(1)
					go c.scanBox(wg, p, row-1, col, -1, 0)
				}
				if row < c.size.Y-1 && c.grid[row+1][col] == '|' {
					wg.Add(1)
					go c.scanBox(wg, p, row+1, col, -1, 0)
				}
			} else if rowInc == -1 && colInc == 0 {
				// Moving up, we can move left or right
				if c.grid[row][col+1] == '-' {
					wg.Add(1)
					go c.scanBox(wg, p, row, col+1, 0, 1)
				}
				if col > 0 && c.grid[row][col-1] == '-' {
					wg.Add(1)
					go c.scanBox(wg, p, row, col-1, 0, -1)
				}
			}

			row += rowInc
			col += colInc
		} else if c.grid[row][col] == '-' && (colInc == 1 || colInc == -1) {
			col += colInc
		} else if c.grid[row][col] == '|' && (rowInc == 1 || rowInc == -1) {
			row += rowInc
		} else {
			return
		}
	}
}

func (c *Canvas) FindBoxes() Boxes {
	wg := new(sync.WaitGroup)

	for row, line := range c.grid {
		for col, char := range line {
			// Corners appearing on the last row or column of the
			// grid do not have enough space to start a new box
			if row < c.size.Y-1 {
				// Only consider boxes starting at top-left
				if isCorner(char) && c.grid[row][col+1] == '-' && c.grid[row+1][col] == '|' {
					wg.Add(1)
					p := make([]image.Point, 0)
					p = append(p, image.Point{X: col, Y: row})
					go c.scanBox(wg, p, row, col+1, 0, 1)
				}
			}
		}
	}

	wg.Wait()
	return nil
}

// Box is scaffolding.
type Box struct {
}

// Boxes is scaffolding.
type Boxes []Box

func (b Boxes) ToSVG() []byte {
	return nil
}
