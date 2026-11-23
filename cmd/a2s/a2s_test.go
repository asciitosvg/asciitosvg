// Copyright 2012 - 2018 The ASCIIToSVG Contributors
// All rights reserved.

package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestScaleFlagCompatibility(t *testing.T) {
	t.Parallel()
	input := "+-+\n| |\n+-+"

	// "-s 18,32" must produce identical SVG output to "-x 18 -y 32".
	var outXY, outS, outAttached, errBuf bytes.Buffer
	if err := run("a2s", []string{"-x", "18", "-y", "32"}, strings.NewReader(input), &outXY, &errBuf); err != nil {
		t.Fatalf("unexpected error with -x/-y: %v", err)
	}
	if err := run("a2s", []string{"-s", "18,32"}, strings.NewReader(input), &outS, &errBuf); err != nil {
		t.Fatalf("unexpected error with -s 18,32: %v", err)
	}
	if err := run("a2s", []string{"-s18,32"}, strings.NewReader(input), &outAttached, &errBuf); err != nil {
		t.Fatalf("unexpected error with -s18,32: %v", err)
	}

	if outXY.String() != outS.String() {
		t.Fatalf("-s 18,32 output differed from -x 18 -y 32")
	}
	if outXY.String() != outAttached.String() {
		t.Fatalf("-s18,32 output differed from -x 18 -y 32")
	}
	if !strings.Contains(outS.String(), `<svg width="72px" height="128px"`) {
		t.Fatalf("expected scaled 72x128 SVG dimensions, got:\n%s", outS.String())
	}
}

func TestInvalidScaleFlag(t *testing.T) {
	t.Parallel()
	badArgs := [][]string{
		{"-s", "9"},
		{"-s", "9,abc"},
		{"-s", "0,16"},
		{"-s", "9,-5"},
		{"-x", "0"},
	}
	for _, args := range badArgs {
		var outBuf, errBuf bytes.Buffer
		if err := run("a2s", args, strings.NewReader("+-+\n+-+"), &outBuf, &errBuf); err == nil {
			t.Errorf("expected error for args %v, got nil", args)
		}
	}
}

func TestVersionFlag(t *testing.T) {
	t.Parallel()
	var outBuf, errBuf bytes.Buffer
	if err := run("a2s", []string{"-v"}, strings.NewReader(""), &outBuf, &errBuf); err != nil {
		t.Fatalf("unexpected error for -v: %v", err)
	}
	if !strings.HasPrefix(outBuf.String(), "a2s ") {
		t.Fatalf("unexpected version output: %q", outBuf.String())
	}
}
