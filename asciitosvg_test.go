// Copyright 2012 - 2015 The ASCIIToSVG Contributors
// All rights reserved.

package asciitosvg

import "testing"

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

`
	c := NewCanvas([]byte(data))
	c.FindBoxes()
}
