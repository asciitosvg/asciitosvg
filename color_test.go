// Copyright 2012 - 2018 The ASCIIToSVG Contributors
// All rights reserved.

package asciitosvg

import (
	"testing"

	"github.com/maruel/ut"
)

func TestParseHexColor(t *testing.T) {
	t.Parallel()
	data := []struct {
		color   string
		rgb     []int
		isError bool
	}{
		{"#fff", []int{255, 255, 255}, false},
		{"#FFF", []int{255, 255, 255}, false},
		{"#ffffff", []int{255, 255, 255}, false},
		{"#FFFFFF", []int{255, 255, 255}, false},
		{"#fFfFFf", []int{255, 255, 255}, false},
		{"#notacolor", nil, true},
		{"alsonotacolor", nil, true},
		{"#ffg", nil, true},
		{"#FFG", nil, true},
		{"#fffffg", nil, true},
		{"#FFFFFG", nil, true},
	}

	for i, v := range data {
		r, g, b, err := colorToRGB(v.color)

		switch v.isError {
		case true:
			if err == nil {
				t.Fatalf("Test %d (%s): wanted error, got no error", i, v.color)
			}
		case false:
			ut.AssertEqualIndex(t, i, err, nil)

			rgb := []int{r, g, b}
			ut.AssertEqualIndex(t, i, v.rgb, rgb)
		}
	}
}
