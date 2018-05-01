// Copyright 2012 - 2018 The ASCIIToSVG Contributors
// All rights reserved.

package asciitosvg

import "unicode"

type char rune

func (c char) isObjectStartTag() bool {
	return c == '['
}

func (c char) isObjectEndTag() bool {
	return c == ']'
}

func (c char) isTagDefinitionSeparator() bool {
	return c == ':'
}

func (c char) isTextStart() bool {
	r := rune(c)
	return c.isObjectStartTag() || unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSymbol(r)
}

func (c char) isTextCont() bool {
	return unicode.IsPrint(rune(c))
}

func (c char) isSpace() bool {
	return unicode.IsSpace(rune(c))
}

// isPathStart returns true on any form of ascii art that can start a graph.
func (c char) isPathStart() bool {
	return (c.isCorner() || c.isHorizontal() || c.isVertical() || c.isArrowHorizontalLeft() || c.isArrowVerticalUp() || c.isDiagonal()) && !c.isTick() && !c.isDot()
}

func (c char) isCorner() bool {
	return c == '.' || c == '\'' || c == '+'
}

func (c char) isRoundedCorner() bool {
	return c == '.' || c == '\''
}

func (c char) isDashedHorizontal() bool {
	return c == '='
}

func (c char) isHorizontal() bool {
	return c.isDashedHorizontal() || c.isTick() || c.isDot() || c == '-'
}

func (c char) isDashedVertical() bool {
	return c == ':'
}

func (c char) isVertical() bool {
	return c.isDashedVertical() || c.isTick() || c.isDot() || c == '|'
}

func (c char) isDashed() bool {
	return c.isDashedHorizontal() || c.isDashedVertical()
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

func (c char) isDiagonalNorthEast() bool {
	return c == '/'
}

func (c char) isDiagonalSouthEast() bool {
	return c == '\\'
}

func (c char) isDiagonal() bool {
	return c.isDiagonalNorthEast() || c.isDiagonalSouthEast()
}

func (c char) isTick() bool {
	return c == 'x'
}

func (c char) isDot() bool {
	return c == 'o'
}

// Diagonal transitions are special: you can move lines diagonally, you can move diagonally from
// corners to edges or lines, but you cannot move diagonally between corners.
func (c char) canDiagonalFrom(from char) bool {
	if from.isArrowVertical() || from.isCorner() {
		return c.isDiagonal()
	} else if from.isDiagonal() {
		return c.isDiagonal() || c.isCorner() || c.isArrowVertical() || c.isHorizontal() || c.isVertical()
	} else if from.isHorizontal() || from.isVertical() {
		return c.isDiagonal()
	}
	return false
}

func (c char) canHorizontal() bool {
	return c.isHorizontal() || c.isCorner() || c.isArrowHorizontal()
}

func (c char) canVertical() bool {
	return c.isVertical() || c.isCorner() || c.isArrowVertical()
}
