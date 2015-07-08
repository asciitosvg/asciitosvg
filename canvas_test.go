// Copyright 2012 - 2015 The ASCIIToSVG Contributors
// All rights reserved.

package asciitosvg

import (
	"image"
	"strings"
	"testing"

	"github.com/maruel/ut"
)

func TestNewCanvas(t *testing.T) {
	t.Parallel()
	data := []struct {
		input   []string
		strings []string
		texts   []string
		corners [][]image.Point
	}{
		// 0 Small box
		{
			[]string{
				"+-+",
				"| |",
				"+-+",
			},
			[]string{"Path{(0,0)}"},
			[]string{""},
			[][]image.Point{{{X: 0, Y: 0}, {X: 2, Y: 0}, {X: 2, Y: 2}, {X: 0, Y: 2}}},
		},

		// 1 Tight box
		{
			[]string{
				"++",
				"++",
			},
			[]string{"Path{(0,0)}"},
			[]string{""},
			[][]image.Point{
				{
					{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 1, Y: 1}, {X: 0, Y: 1},
				},
			},
		},

		// 2 Indented box
		{
			[]string{
				"",
				" +-+",
				" | |",
				" +-+",
			},
			[]string{"Path{(1,1)}"},
			[]string{""},
			[][]image.Point{{{X: 1, Y: 1}, {X: 3, Y: 1}, {X: 3, Y: 3}, {X: 1, Y: 3}}},
		},

		// 3 Free flow text
		{
			[]string{
				"",
				" foo bar ",
				"b  baz   bee",
			},
			[]string{"Text{(1,1) \"foo bar\"}", "Text{(0,2) \"b  baz\"}", "Text{(9,2) \"bee\"}"},
			[]string{"foo bar", "b  baz", "bee"},
			[][]image.Point{
				{{X: 1, Y: 1}, {X: 7, Y: 1}},
				{{X: 0, Y: 2}, {X: 5, Y: 2}},
				{{X: 9, Y: 2}, {X: 11, Y: 2}},
			},
		},

		// 4 Text in a box
		{
			[]string{
				"+--+",
				"|Hi|",
				"+--+",
			},
			[]string{"Path{(0,0)}", "Text{(1,1) \"Hi\"}"},
			[]string{"", "Hi"},
			[][]image.Point{
				{{X: 0, Y: 0}, {X: 3, Y: 0}, {X: 3, Y: 2}, {X: 0, Y: 2}},
				{{X: 1, Y: 1}, {X: 2, Y: 1}},
			},
		},

		// 5 Concave pieces
		{
			[]string{
				"    +----+",
				"    |    |",
				"+---+    +----+",
				"|             |",
				"+-------------+",
				"", // 5
				"+----+",
				"|    |",
				"|    +---+",
				"|        |",
				"|    +---+", // 10
				"|    |",
				"+----+",
				"",
				"    +----+",
				"    |    |", // 15
				"+---+    |",
				"|        |",
				"+---+    |",
				"    |    |",
				"    +----+",
			},
			[]string{"Path{(4,0)}", "Path{(0,6)}", "Path{(4,14)}"},
			[]string{"", "", ""},
			[][]image.Point{
				{
					{X: 4, Y: 0}, {X: 9, Y: 0}, {X: 9, Y: 2}, {X: 14, Y: 2},
					{X: 14, Y: 4}, {X: 0, Y: 4}, {X: 0, Y: 2}, {X: 4, Y: 2},
				},
				{
					{X: 0, Y: 6}, {X: 5, Y: 6}, {X: 5, Y: 8}, {X: 9, Y: 8},
					{X: 9, Y: 10}, {X: 5, Y: 10}, {X: 5, Y: 12}, {X: 0, Y: 12},
				},
				{
					{X: 4, Y: 14}, {X: 9, Y: 14}, {X: 9, Y: 20}, {X: 4, Y: 20},
					{X: 4, Y: 18}, {X: 0, Y: 18}, {X: 0, Y: 16}, {X: 4, Y: 16},
				},
			},
		},

		// 6 Inner boxes
		{
			[]string{
				"+-----+",
				"|     |",
				"| +-+ |",
				"| | | |",
				"| +-+ |",
				"|     |",
				"+-----+",
			},
			[]string{"Path{(0,0)}", "Path{(2,2)}"},
			[]string{"", ""},
			[][]image.Point{
				{{X: 0, Y: 0}, {X: 6, Y: 0}, {X: 6, Y: 6}, {X: 0, Y: 6}},
				{{X: 2, Y: 2}, {X: 4, Y: 2}, {X: 4, Y: 4}, {X: 2, Y: 4}},
			},
		},

		// 7 Real world diagram example
		{
			[]string{
				//         1         2         3
				"      +------+",
				"      |Editor|-------------+--------+",
				"      +------+             |        |",
				"          |                |        v",
				"          v                |   +--------+",
				"      +------+             |   |Document|", // 5
				"      |Window|             |   +--------+",
				"      +------+             |",
				"         |                 |",
				"   +-----+-------+         |",
				"   |             |         |", // 10
				"   v             v         |",
				"+------+     +------+      |",
				"|Window|     |Window|      |",
				"+------+     +------+      |",
				"                |          |", // 15
				"                v          |",
				"              +----+       |",
				"              |View|       |",
				"              +----+       |",
				"                |          |", // 20
				"                v          |",
				"            +--------+     |",
				"            |Document|<----+",
				"            +--------+",
			},
			[]string{
				"Path{(6,0)}",
				"Path{(14,1)}",
				"Path{(14,1)}",
				"Path{(10,3)}",
				"Path{(31,4)}",
				"Path{(6,5)}",
				"Path{(9,8)}",
				"Path{(9,8)}",
				"Path{(0,12)}",
				"Path{(13,12)}",
				"Path{(16,15)}",
				"Path{(14,17)}",
				"Path{(16,20)}",
				"Path{(12,22)}",
				"Text{(7,1) \"Editor\"}",
				"Text{(32,5) \"Document\"}",
				"Text{(7,6) \"Window\"}",
				"Text{(1,13) \"Window\"}",
				"Text{(14,13) \"Window\"}",
				"Text{(15,18) \"View\"}",
				"Text{(13,23) \"Document\"}",
			},
			[]string{
				"", "", "", "", "", "", "", "", "", "", "", "", "", "", "Editor",
				"Document", "Window", "Window", "Window", "View", "Document",
			},
			[][]image.Point{
				{{X: 6, Y: 0}, {X: 13, Y: 0}, {X: 13, Y: 2}, {X: 6, Y: 2}},
				{{X: 14, Y: 1}, {X: 36, Y: 1}, {X: 36, Y: 3}},
				{{X: 14, Y: 1}, {X: 27, Y: 1}, {X: 27, Y: 23}, {X: 22, Y: 23}},
				{{X: 10, Y: 3}, {X: 10, Y: 4}},
				{{X: 31, Y: 4}, {X: 40, Y: 4}, {X: 40, Y: 6}, {X: 31, Y: 6}},
				{{X: 6, Y: 5}, {X: 13, Y: 5}, {X: 13, Y: 7}, {X: 6, Y: 7}},
				{{X: 9, Y: 8}, {X: 9, Y: 9}, {X: 3, Y: 9}, {X: 3, Y: 11}},
				{{X: 9, Y: 8}, {X: 9, Y: 9}, {X: 17, Y: 9}, {X: 17, Y: 11}},
				{{X: 0, Y: 12}, {X: 7, Y: 12}, {X: 7, Y: 14}, {X: 0, Y: 14}},
				{{X: 13, Y: 12}, {X: 20, Y: 12}, {X: 20, Y: 14}, {X: 13, Y: 14}},
				{{X: 16, Y: 15}, {X: 16, Y: 16}},
				{{X: 14, Y: 17}, {X: 19, Y: 17}, {X: 19, Y: 19}, {X: 14, Y: 19}},
				{{X: 16, Y: 20}, {X: 16, Y: 21}},
				{{X: 12, Y: 22}, {X: 21, Y: 22}, {X: 21, Y: 24}, {X: 12, Y: 24}},
				{{X: 7, Y: 1}, {X: 12, Y: 1}},
				{{X: 32, Y: 5}, {X: 39, Y: 5}},
				{{X: 7, Y: 6}, {X: 12, Y: 6}},
				{{X: 1, Y: 13}, {X: 6, Y: 13}},
				{{X: 14, Y: 13}, {X: 19, Y: 13}},
				{{X: 15, Y: 18}, {X: 18, Y: 18}},
				{{X: 13, Y: 23}, {X: 20, Y: 23}},
			},
		},

		// 8 Interwined lines.
		{
			[]string{
				"             +-----+-------+",
				"             |     |       |",
				"             |     |       |",
				"        +----+-----+----   |",
				"--------+----+-----+-------+---+",
				"        |    |     |       |   |",
				"        |    |     |       |   |     |   |",
				"        |    |     |       |   |     |   |",
				"        |    |     |       |   |     |   |",
				"--------+----+-----+-------+---+-----+---+--+",
				"        |    |     |       |   |     |   |  |",
				"        |    |     |       |   |     |   |  |",
				"        |   -+-----+-------+---+-----+   |  |",
				"        |    |     |       |   |     |   |  |",
				"        |    |     |       |   +-----+---+--+",
				"             |     |       |         |   |",
				"             |     |       |         |   |",
				"     --------+-----+-------+---------+---+-----",
				"             |     |       |         |   |",
				"             +-----+-------+---------+---+",
			},
			// TODO(maruel): it's a tad overwhelming.
			nil,
			nil,
			nil,
		},
	}
	for i, line := range data {
		objs := Parse([]byte(strings.Join(line.input, "\n")), 9).Objects()
		if line.strings != nil {
			ut.AssertEqualIndex(t, i, line.strings, getStrings(objs))
		}
		if line.texts != nil {
			ut.AssertEqualIndex(t, i, line.texts, getTexts(objs))
		}
		if line.corners != nil {
			ut.AssertEqualIndex(t, i, line.corners, getCorners(objs))
		}
	}
}

func TestNewCanvasBroken(t *testing.T) {
	// These are the ones that do not give the desired result.
	t.Parallel()
	data := []struct {
		input   []string
		strings []string
		texts   []string
		corners [][]image.Point
	}{
		// 0 Indented box
		{
			[]string{
				"",
				"\t+-+",
				"\t| |",
				"\t+-+",
			},
			[]string{"Path{(1,1)}"},
			[]string{""},
			[][]image.Point{{{X: 1, Y: 1}, {X: 3, Y: 1}, {X: 3, Y: 3}, {X: 1, Y: 3}}},
		},

		// 1 URL
		{
			[]string{
				"github.com/foo/bar",
			},
			[]string{"Text{(0,0) \"github.com/foo/bar\"}"},
			[]string{"github.com/foo/bar"},
			[][]image.Point{{{X: 0, Y: 0}, {X: 17, Y: 0}}},
		},

		// 2 Merged boxes
		{
			[]string{
				"+-+-+",
				"| | |",
				"+-+-+",
			},
			[]string{"Path{(0,0)}", "Path{(0,0)}"},
			[]string{"", ""},
			// TODO(maruel): BROKEN.
			[][]image.Point{
				{{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 4, Y: 2}, {X: 0, Y: 2}},
				{{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 4, Y: 2}, {X: 2, Y: 2}, {X: 2, Y: 1}},
			},
		},

		// 3 Adjascent boxes
		{
			// TODO(maruel): BROKEN. This one is hard, as it can be seen as 3 boxes
			// but that is not what is desired.
			[]string{
				"+-++-+",
				"| || |",
				"+-++-+",
			},
			[]string{"Path{(0,0)}", "Path{(0,0)}", "Path{(0,0)}"},
			[]string{"", "", ""},
			[][]image.Point{
				{{X: 0, Y: 0}, {X: 5, Y: 0}, {X: 5, Y: 2}, {X: 0, Y: 2}},
				{{X: 0, Y: 0}, {X: 5, Y: 0}, {X: 5, Y: 2}, {X: 2, Y: 2}, {X: 2, Y: 1}},
				{{X: 0, Y: 0}, {X: 5, Y: 0}, {X: 5, Y: 2}, {X: 3, Y: 2}, {X: 3, Y: 1}},
			},
		},
	}
	for i, line := range data {
		objs := Parse([]byte(strings.Join(line.input, "\n")), 9).Objects()
		if line.strings != nil {
			ut.AssertEqualIndex(t, i, line.strings, getStrings(objs))
		}
		if line.texts != nil {
			ut.AssertEqualIndex(t, i, line.texts, getTexts(objs))
		}
		if line.corners != nil {
			ut.AssertEqualIndex(t, i, line.corners, getCorners(objs))
		}
	}
}

func TestPointsToCorners(t *testing.T) {
	t.Parallel()
	data := []struct {
		in       []image.Point
		expected []image.Point
		closed   bool
	}{
		{
			[]image.Point{{X: 0, Y: 0}, {X: 1, Y: 0}},
			[]image.Point{{X: 0, Y: 0}, {X: 1, Y: 0}},
			false,
		},
		{
			[]image.Point{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 2, Y: 0}},
			[]image.Point{{X: 0, Y: 0}, {X: 2, Y: 0}},
			false,
		},
		{
			[]image.Point{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 1, Y: 1}},
			[]image.Point{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 1, Y: 1}},
			false,
		},
		{
			[]image.Point{
				{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 2, Y: 0}, {X: 2, Y: 1}, {X: 2, Y: 2},
				{X: 1, Y: 2}, {X: 0, Y: 2}, {X: 0, Y: 1},
			},
			[]image.Point{{X: 0, Y: 0}, {X: 2, Y: 0}, {X: 2, Y: 2}, {X: 0, Y: 2}},
			true,
		},
		{
			[]image.Point{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 1, Y: 1}, {X: 0, Y: 1}},
			[]image.Point{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 1, Y: 1}, {X: 0, Y: 1}},
			// TODO(maruel): Unexpected; broken.
			false,
		},
	}
	for i, line := range data {
		p, c := pointsToCorners(line.in)
		ut.AssertEqualIndex(t, i, line.expected, p)
		ut.AssertEqualIndex(t, i, line.closed, c)
	}
}

func BenchmarkT(b *testing.B) {
	data := []string{
		"             +-----+-------+",
		"             |     |       |",
		"             |     |       |",
		"        +----+-----+----   |",
		"--------+----+-----+-------+---+",
		"        |    |     |       |   |",
		"        |    |     |       |   |     |   |",
		"        |    |     |       |   |     |   |",
		"        |    |     |       |   |     |   |",
		"--------+----+-----+-------+---+-----+---+--+",
		"        |    |     |       |   |     |   |  |",
		"        |    |     |       |   |     |   |  |",
		"        |   -+-----+-------+---+-----+   |  |",
		"        |    |     |       |   |     |   |  |",
		"        |    |     |       |   +-----+---+--+",
		"             |     |       |         |   |",
		"             |     |       |         |   |",
		"     --------+-----+-------+---------+---+-----",
		"             |     |       |         |   |",
		"             +-----+-------+---------+---+",
		"",
		"",
	}
	chunk := []byte(strings.Join(data, "\n"))
	input := make([]byte, 0, len(chunk)*b.N)
	for i := 0; i < b.N; i++ {
		input = append(input, chunk...)
	}
	expected := 30 * b.N
	b.ResetTimer()
	objs := Parse(input, 8).Objects()
	if len(objs) != expected {
		b.Fatalf("%d != %d", len(objs), expected)
	}
}

// Private details.

func getPoints(objs []Object) [][]image.Point {
	out := [][]image.Point{}
	for _, obj := range objs {
		out = append(out, obj.Points())
	}
	return out
}

func getTexts(objs []Object) []string {
	out := []string{}
	for _, obj := range objs {
		t := obj.Text()
		if !obj.IsText() {
			out = append(out, "")
		} else if len(t) > 0 {
			out = append(out, string(t))
		} else {
			panic("failed")
		}
	}
	return out
}

func getStrings(objs []Object) []string {
	out := []string{}
	for _, obj := range objs {
		out = append(out, obj.String())
	}
	return out
}

func getCorners(objs []Object) [][]image.Point {
	out := make([][]image.Point, len(objs))
	for i, obj := range objs {
		out[i] = obj.Corners()
	}
	return out
}
