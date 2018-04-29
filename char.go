// Copyright 2012 - 2015 The ASCIIToSVG Contributors
// All rights reserved.

package asciitosvg

import "unicode"

type charType int

const (
	point   charType = 0x1
	control charType = 0x2
	smarker charType = 0x4
	imarker charType = 0x8
	tick    charType = 0x10
	dot     charType = 0x20
)

type direction int

const (
	dirUp    direction = 0x1
	dirDown  direction = 0x2
	dirLeft  direction = 0x4
	dirRight direction = 0x8
	dirNe    direction = 0x10
	dirSe    direction = 0x20
)

// TODO(dhobsd): Add charType as a flag, make it a struct.
type char rune

func (c char) isTextStart() bool {
	r := rune(c)
	return unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSymbol(r)
}

func (c char) isTextCont() bool {
	return unicode.IsPrint(rune(c))
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
	return c == '-' || c == '+'
}

func (c char) isVertical() bool {
	return c == '|' || c == '+'
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

func (c char) isArrow() bool {
	return c.isArrowHorizontal() || c.isArrowVertical()
}

// TODO(dhobsd): Diagonal.

func (c char) canHorizontal() bool {
	return c.isHorizontal() || c.isCorner() || c.isArrowHorizontal()
}

func (c char) canVertical() bool {
	return c.isVertical() || c.isCorner() || c.isArrowVertical()
}
