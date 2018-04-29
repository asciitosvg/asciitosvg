// Copyright 2012 - 2015 The ASCIIToSVG Contributors
// All rights reserved.

package asciitosvg

import (
	"strings"
	"testing"

	"github.com/maruel/ut"
)

func TestCanvasToSVG(t *testing.T) {
	t.Parallel()
	data := []string{
		"+--+",
		"|Hi|",
		"+--+",
	}
	canvas, err := NewCanvas([]byte(strings.Join(data, "\n")), 9)
	if err != nil {
		t.Fatalf("Error creating canvas: %s", err)
	}
	actual := string(CanvasToSVG(canvas, false, "", 9, 16))
	// TODO(dhobsd): Use golden file? Worth postponing once output is actually
	// nice.
	ut.AssertEqual(t, 1598, len(actual))
}
