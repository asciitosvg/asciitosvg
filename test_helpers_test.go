// Copyright 2012 - 2018 The ASCIIToSVG Contributors
// All rights reserved.

package asciitosvg

import (
	"reflect"
	"testing"
)

func assertEqualIndex(t *testing.T, index int, expected, actual interface{}) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Test %d:\n  expected: %#v\n  actual:   %#v", index, expected, actual)
	}
}
