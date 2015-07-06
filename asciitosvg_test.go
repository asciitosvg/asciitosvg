// Copyright 2012 - 2015 The ASCIIToSVG Contributors
// All rights reserved.

package asciitosvg

import (
	"testing"

	"github.com/maruel/ut"
)

func TestNewCanvas(t *testing.T) {
	data := `
      +------+
      |Editor|-------------+--------+
      +------+             |        |
          |                |        v
          v                |   +--------+
      +------+             |   |Document|
      |Window|             |   +--------+
      +------+             |
         |                 |
   +-----+-------+         |
   |             |         |
   v             v         |
+------+     +------+      |
|Window|     |Window|      |
+------+     +------+      |
                |          |
                v          |
              +----+       |
              |View|       |
              +----+       |
                |          |
                v          |
            +--------+     |
            |Document|<----+
            +--------+
    +----+
    |    |
+---+    +----+
|             |
+-------------+

+----+
|    |
|    +---+
|        |
|    +---+
|    |
+----+

    +----+
    |    |
+---+    |
|        |
+---+    |
    |    |
    +----+

             +-----+-------+
             |     |       |
             |     |       |
        +----+-----+----   |
--------+----+-----+-------+---+
        |    |     |       |   |
        |    |     |       |   |     |   |
        |    |     |       |   |     |   |
        |    |     |       |   |     |   |
--------+----+-----+-------+---+-----+---+--+
        |    |     |       |   |     |   |  |
        |    |     |       |   |     |   |  |
        |   -+-----+-------+---+-----+   |  |
        |    |     |       |   |     |   |  |
        |    |     |       |   +-----+---+--+
             |     |       |         |   |
             |     |       |         |   |
     --------+-----+-------+---------+---+-----
             |     |       |         |   |
             +-----+-------+---------+---+
`
	NewCanvas([]byte(data))
}

func TestNewCanvasText(t *testing.T) {
	n := NewCanvas([]byte("\n foo bar \nb"))
	o := n.FindObjects()
	ut.AssertEqual(t, 2, len(o))
	ut.AssertEqual(t, []rune("b"), o[0].Text())
	ut.AssertEqual(t, []rune("foo bar"), o[1].Text())
}
