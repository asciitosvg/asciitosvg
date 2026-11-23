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
	return c.isTextCont() && !c.isSpace()
}

func (c char) isTextCont() bool {
	return unicode.IsPrint(rune(c))
}

func (c char) isSpace() bool {
	return unicode.IsSpace(rune(c))
}

func (c char) isAlnum() bool {
	r := rune(c)
	return unicode.IsLetter(r) || unicode.IsNumber(r)
}

func (c char) isAngledCorner() bool {
	return c == '+' || c == '#'
}

func (c char) isCorner() bool {
	return c.isRoundedCorner() || c.isAngledCorner()
}

func (c char) isRoundedCorner() bool {
	return c == '.' || c == '\''
}

func (c char) isDashedHorizontal() bool {
	return c == '='
}

func (c char) isHorizontalLine() bool {
	return c == '-' || c.isDashedHorizontal()
}

func (c char) isDashedVertical() bool {
	return c == ':'
}

func (c char) isVerticalLine() bool {
	return c == '|' || c.isDashedVertical()
}

func (c char) isDashed() bool {
	return c.isDashedHorizontal() || c.isDashedVertical()
}

func (c char) isArrowHorizontal() bool {
	return c == '<' || c == '>'
}

func (c char) isArrowVertical() bool {
	return c == '^' || c == 'v'
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

func (c char) isTick() bool {
	return c == 'x'
}

func (c char) isDot() bool {
	return c == 'o'
}
