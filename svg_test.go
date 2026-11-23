// Copyright 2012 - 2018 The ASCIIToSVG Contributors
// All rights reserved.

package asciitosvg

import (
	"encoding/xml"
	"strings"
	"testing"
)

func TestCanvasToSVG(t *testing.T) {
	t.Parallel()
	data := []struct {
		name     string
		input    []string
		noBlur   bool
		contains []string
		excludes []string
	}{
		{
			name: "box with dashed edge and rounded corner",
			input: []string{
				"+--.",
				"|Hi:",
				"+--+",
			},
			contains: []string{
				`stroke-dasharray="5 5"`,
				`fill="#fff"`,
				`filter="url(#dsFilter)"`,
				`M 4.5 8 L 21.5 8 Q 31.5 8 31.5 18 L 31.5 40 L 4.5 40 Z`,
				`>Hi</text>`,
			},
			excludes: []string{`L 0 0`},
		},
		{
			name: "box with non-existent ref keeps default white fill",
			input: []string{
				".-----.",
				"|[a]  |",
				"'-----'",
			},
			contains: []string{
				`fill="#fff"`,
				`M 4.5 18 Q 4.5 8 14.5 8 L 48.5 8 Q 58.5 8 58.5 18 L 58.5 30 Q 58.5 40 48.5 40 L 14.5 40 Q 4.5 40 4.5 30 Z`,
				`>[a]</text>`,
			},
		},
		{
			name: "box with dark fill sets contrasting white text",
			input: []string{
				".-----.",
				"|[a]  |",
				"'-----'",
				"",
				`[a]: {"fill":"#000000"}`,
			},
			contains: []string{
				`fill="#000000"`,
				`fill="#fff">[a]</text>`,
			},
		},
		{
			name: "box with label keeps definition text intact when not delref",
			input: []string{
				".-----.",
				"|[a]  |",
				"'-----'",
				"",
				`[a]: {"fill":"#000000","a2s:label":"abcdefg"}`,
			},
			contains: []string{
				`fill="#000000"`,
				`fill="#fff">abcdefg</text>`,
				`>[a]: {`,
			},
		},
		{
			name: "box with label and delref removes definition and shrinks canvas",
			input: []string{
				".-----.",
				"|[a]  |",
				"'-----'",
				"",
				`[a]: {"fill":"#000000","a2s:label":"abcd","a2s:delref":1}`,
			},
			contains: []string{
				`<svg width="72px" height="64px"`,
				`fill="#000000"`,
				`fill="#fff">abcd</text>`,
			},
			excludes: []string{`[a]:`},
		},
		{
			name: "ticks and dots in lines",
			input: []string{
				" ------x----->",
				"",
				" <-----o------",
			},
			contains: []string{
				`marker-end="url(#Pointer)"`,
				`marker-start="url(#iPointer)"`,
				`<line x1="63.5" y1="4" x2="71.5" y2="12"`,
				`<circle cx="67.5" cy="40" r="3"`,
			},
		},
		{
			name: "just text",
			input: []string{
				" foo",
			},
			contains: []string{
				`>foo</text>`,
			},
		},
		{
			name: "column 0 text with [0,0] delref preserves text",
			input: []string{
				"foo",
				`[0,0]: {"a2s:delref":1,"a2s:label":"bar"}`,
			},
			contains: []string{
				`>bar</text>`,
			},
			excludes: []string{`[0,0]:`},
		},
		{
			name: "link wraps text in valid xlink:href anchor",
			input: []string{
				" foo",
				`[1,0]: {"a2s:delref":1, "a2s:link":"https://github.com/asciitosvg/asciitosvg"}`,
			},
			contains: []string{
				`<a xlink:href="https://github.com/asciitosvg/asciitosvg"><text id="obj0" x="9" y="13" fill="#000">foo</text></a>`,
			},
			excludes: []string{`[1,0]:`},
		},
		{
			name: "octothorpe corner closes polygon and applies tag fill",
			input: []string{
				".-------------.   ",
				"|[1] ^        |   ",
				"|     \\  .----'   ",
				"|      \\ |        ",
				"#----.  \\'----.   ",
				"     |   \\    |   ",
				"     '----+---'   ",
				"",
				`[1]: {"fill":"#933","a2s:delref":true}`,
			},
			contains: []string{
				`<path id="closed`,
				`fill="#933"`,
			},
			excludes: []string{`>#</text>`, `[1]`},
		},
		{
			name: "diagonal rounded octagon has tangent curves without (0,0) artifacts",
			input: []string{
				"   .-----.",
				"  /       \\",
				" /         \\",
				"+           +",
				"|           |",
				"|           |",
				"+           +",
				" \\         /",
				"  \\       /",
				"   '-----'",
			},
			contains: []string{`Q 31.5 8`, `Q 85.5 8`, `Q 85.5 152`, `Q 31.5 152`},
			excludes: []string{`L 0 0`, `Q 85.5 152 0 0`},
		},
		{
			name: "adjacent merged boxes produce two closed paths with no open stubs",
			input: []string{
				"+-+-+",
				"| | |",
				"+-+-+",
			},
			contains: []string{
				`<path id="closed0"`,
				`<path id="closed1"`,
			},
			excludes: []string{`<path id="open`},
		},
		{
			name: "untagged inner box inside dark outer box uses dark text on white fill",
			input: []string{
				".-----------.",
				"|[outer]    |",
				"|  .-----.  |",
				"|  | hi  |  |",
				"|  '-----'  |",
				"'-----------'",
				"",
				`[outer]: {"fill":"#000000","a2s:delref":1}`,
			},
			contains: []string{
				`fill="#000000"`,
				`fill="#fff"`,
				`fill="#000">hi</text>`,
			},
			excludes: []string{`[outer]`},
		},
		{
			name: "inward and outward arrow directions",
			input: []string{
				">---+",
				"    |",
				"    ^",
				"",
				"+---<",
			},
			contains: []string{
				`marker-start="url(#Pointer)"`,
				`marker-end="url(#iPointer)"`,
			},
		},
		{
			name: "noBlur disables group filter",
			input: []string{
				"+-+",
				"| |",
				"+-+",
			},
			noBlur:   true,
			contains: []string{`<g id="closed" stroke="#000" stroke-width="2" fill="none">`},
			excludes: []string{`<g id="closed" filter="url(#dsFilter)"`},
		},
		{
			name: "numeric JSON option formats without UNIMPLEMENTED",
			input: []string{
				"+---+",
				"|[a]|",
				"+---+",
				`[a]: {"stroke-width":4,"a2s:delref":1}`,
			},
			contains: []string{`stroke-width="4"`, `fill="#fff"`},
			excludes: []string{`UNIMPLEMENTED`},
		},
		{
			name: "unambiguous mid-edge rounded corners render stylistic cusps",
			input: []string{
				".---.-.",
				"| .-. |",
				"| '-' |",
				"'---'-'",
			},
			contains: []string{
				`Q 40.5 8 40.5 16 Q 40.5 8 49.5 8`,
				`Q 40.5 56 40.5 48 Q 40.5 56 30.5 56`,
			},
		},
		{
			name: "mid-edge dot with attached vertical wall does not cusp",
			input: []string{
				".---.-.",
				"|   | |",
				"'---+-'",
			},
			excludes: []string{
				`Q 40.5 8 40.5 16`,
			},
		},
	}

	for _, tc := range data {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			canvas, err := NewCanvas([]byte(strings.Join(tc.input, "\n")), 9, tc.noBlur)
			if err != nil {
				t.Fatalf("error creating canvas: %s", err)
			}
			actualBytes := CanvasToSVG(canvas, tc.noBlur, "", 9, 16)
			var doc interface{}
			if err := xml.Unmarshal(actualBytes, &doc); err != nil {
				t.Fatalf("generated invalid XML: %v\n%s", err, string(actualBytes))
			}
			actual := string(actualBytes)
			for _, sub := range tc.contains {
				if !strings.Contains(actual, sub) {
					t.Errorf("expected SVG to contain %q\ngot:\n%s", sub, actual)
				}
			}
			for _, sub := range tc.excludes {
				if strings.Contains(actual, sub) {
					t.Errorf("expected SVG NOT to contain %q\ngot:\n%s", sub, actual)
				}
			}
		})
	}
}
