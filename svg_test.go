// Copyright 2012 - 2018 The ASCIIToSVG Contributors
// All rights reserved.

package asciitosvg

import (
	"strings"
	"testing"

	"github.com/maruel/ut"
)

func TestCanvasToSVG(t *testing.T) {
	t.Parallel()
	data := []struct {
		input  []string
		length int
	}{
		// 0 Box with dashed corners and text
		{
			[]string{
				"+--.",
				"|Hi:",
				"+--+",
			},
			1677,
		},

		// 1 Box with non-existent ref
		{
			[]string{
				".-----.",
				"|[a]  |",
				"'-----'",
			},
			1763,
		},

		// 2 Box with ref, change background color of container with #RRGGBB
		{
			[]string{
				".-----.",
				"|[a]  |",
				"'-----'",
				"",
				"[a]: {\"fill\":\"#000000\"}",
			},
			1858,
		},

		// 3 Box with ref && fill, change label
		{
			[]string{
				".-----.",
				"|[a]  |",
				"'-----'",
				"",
				"[a]: {\"fill\":\"#000000\",\"a2s:label\":\"abcdefg\"}",
			},
			1826,
		},

		// 4 Box with ref && fill && label, remove ref
		{
			[]string{
				".-----.",
				"|[a]  |",
				"'-----'",
				"",
				"[a]: {\"fill\":\"#000000\",\"a2s:label\":\"abcd\",\"a2s:delref\":1}",
			},
			1764,
		},

		// 5 Ticks and dots in lines.
		{
			[]string{
				" ------x----->",
				"",
				" <-----o------",
			},
			1968,
		},

		// 6 Just text
		{
			[]string{
				" foo",
			},
			1476,
		},

		// 7 Just text with a deleting reference
		{
			[]string{
				" foo",
				"[1,0]: {\"a2s:delref\":1,\"a2s:label\":\"foo\"}",
			},
			1477,
		},

		// 8 Just text with a link
		{
			[]string{
				" foo",
				"[1,0]: {\"a2s:delref\":1, \"a2s:link\":\"https://github.com/asciitosvg/asciitosvg\"}",
			},
			1521,
		},
	}
	for i, line := range data {
		canvas, err := NewCanvas([]byte(strings.Join(line.input, "\n")), 9, false)
		if err != nil {
			t.Fatalf("Error creating canvas: %s", err)
		}
		actual := string(CanvasToSVG(canvas, false, "", 9, 16))
		// TODO(dhobsd): Use golden file? Worth postponing once output is actually
		// nice.
		ut.AssertEqualIndex(t, i, line.length, len(actual))
	}
}
