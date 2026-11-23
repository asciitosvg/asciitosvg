// Copyright 2012 - 2018 The ASCIIToSVG Contributors
// All rights reserved.

package asciitosvg

import (
	"encoding/xml"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var renderGallery = flag.Bool("render-gallery", false, "write rendered SVGs (and @2x PNGs if rsvg-convert is installed) plus TEST_CASES.md")

type galleryCase struct {
	section string
	title   string
	noBlur  bool
	input   string
}

var galleryCases = []galleryCase{
	{
		section: "Canvas & Pathing Cases",
		title:   "Small box",
		noBlur:  false,
		input:   "+-+\n| |\n+-+",
	},
	{
		section: "Canvas & Pathing Cases",
		title:   "Tight box (2x2)",
		noBlur:  false,
		input:   "++\n++",
	},
	{
		section: "Canvas & Pathing Cases",
		title:   "Indented box",
		noBlur:  false,
		input:   "\n +-+\n | |\n +-+",
	},
	{
		section: "Canvas & Pathing Cases",
		title:   "Free flow text",
		noBlur:  false,
		input:   "\n foo bar \nb  baz   bee",
	},
	{
		section: "Canvas & Pathing Cases",
		title:   "Text in a box",
		noBlur:  false,
		input:   "+--+\n|Hi|\n+--+",
	},
	{
		section: "Canvas & Pathing Cases",
		title:   "Concave pieces",
		noBlur:  false,
		input:   "    +----+\n    |    |\n+---+    +----+\n|             |\n+-------------+\n\n+----+\n|    |\n|    +---+\n|        |\n|    +---+\n|    |\n+----+\n\n    +----+\n    |    |\n+---+    |\n|        |\n+---+    |\n    |    |\n    +----+",
	},
	{
		section: "Canvas & Pathing Cases",
		title:   "Inner boxes",
		noBlur:  false,
		input:   "+-----+\n|     |\n| +-+ |\n| | | |\n| +-+ |\n|     |\n+-----+",
	},
	{
		section: "Canvas & Pathing Cases",
		title:   "Real world diagram",
		noBlur:  false,
		input:   "      +------+\n      |Editor|-------------+--------+\n      +------+             |        |\n          |                |        v\n          v                |   +--------+\n      +------+             |   |Document|\n      |Window|             |   +--------+\n      +------+             |\n         |                 |\n   +-----+-------+         |\n   |             |         |\n   v             v         |\n+------+     +------+      |\n|Window|     |Window|      |\n+------+     +------+      |\n                |          |\n                v          |\n              +----+       |\n              |View|       |\n              +----+       |\n                |          |\n                v          |\n            +--------+     |\n            |Document|<----+\n            +--------+",
	},
	{
		section: "Canvas & Pathing Cases",
		title:   "Intertwined lines",
		noBlur:  false,
		input:   "             +-----+-------+\n             |     |       |\n             |     |       |\n        +----+-----+----   |\n--------+----+-----+-------+---+\n        |    |     |       |   |\n        |    |     |       |   |     |   |\n        |    |     |       |   |     |   |\n        |    |     |       |   |     |   |\n--------+----+-----+-------+---+-----+---+--+\n        |    |     |       |   |     |   |  |\n        |    |     |       |   |     |   |  |\n        |   -+-----+-------+---+-----+   |  |\n        |    |     |       |   |     |   |  |\n        |    |     |       |   +-----+---+--+\n             |     |       |         |   |\n             |     |       |         |   |\n     --------+-----+-------+---------+---+-----\n             |     |       |         |   |\n             +-----+-------+---------+---+",
	},
	{
		section: "Canvas & Pathing Cases",
		title:   "Tab-indented box",
		noBlur:  false,
		input:   "\n\t+-+\n\t| |\n\t+-+",
	},
	{
		section: "Canvas & Pathing Cases",
		title:   "Diagonal lines with arrows",
		noBlur:  false,
		input:   "^          ^\n \\        /\n  \\      /\n   \\    /\n    v  v",
	},
	{
		section: "Canvas & Pathing Cases",
		title:   "Diagonal octagon with rounded corners",
		noBlur:  false,
		input:   "   .-----.\n  /       \\\n /         \\\n+           +\n|           |\n|           |\n+           +\n \\         /\n  \\       /\n   '-----'",
	},
	{
		section: "Canvas & Pathing Cases",
		title:   "A2S logo (colored via [1,0] tag)",
		noBlur:  false,
		input:   " .-------------------------.\n |                         |\n | .---.-. .-----. .-----. |\n | | .-. | +-->  | |  <--+ |\n | | '-' | |  <--+ +-->  | |\n | '---'-' '-----' '-----' |\n |  ascii     2      svg   |\n |                         |\n '-------------------------'\n\n[1,0]: {\"fill\":\"#88d\",\"a2s:delref\":1}",
	},
	{
		section: "Canvas & Pathing Cases",
		title:   "Ticks and dots",
		noBlur:  false,
		input:   " ------x----->\n\n <-----o------",
	},
	{
		section: "Canvas & Pathing Cases",
		title:   "URL text",
		noBlur:  false,
		input:   "github.com/foo/bar",
	},
	{
		section: "Canvas & Pathing Cases",
		title:   "Merged boxes",
		noBlur:  false,
		input:   "+-+-+\n| | |\n+-+-+",
	},
	{
		section: "Canvas & Pathing Cases",
		title:   "Adjacent sharp boxes",
		noBlur:  false,
		input:   "+-++-+\n| || |\n+-++-+",
	},
	{
		section: "Canvas & Pathing Cases",
		title:   "Adjacent rounded boxes",
		noBlur:  false,
		input:   ".--..--.\n|  ||  |\n'--''--'",
	},
	{
		section: "Canvas & Pathing Cases",
		title:   "Stacked sharp and rounded boxes",
		noBlur:  false,
		input:   "+--+\n|  |\n+--+\n+--+\n|  |\n+--+\n.--.\n|  |\n'--'\n.--.\n|  |\n'--'",
	},
	{
		section: "Canvas & Pathing Cases",
		title:   "Adjacent parallel vertical and diagonal lines",
		noBlur:  false,
		input:   "||  //\n|| //\n||//",
	},
	{
		section: "Canvas & Pathing Cases",
		title:   "Straight crossings and adjacent elbows",
		noBlur:  false,
		input:   "  |   ---+  \n--+--    |--\n  |      v  ",
	},
	{
		section: "Canvas & Pathing Cases",
		title:   "Text with slashes, ticks, dots, and punctuation inside a box",
		noBlur:  false,
		input:   "+-----------------+\n| /home/dho/a2s   |\n| yes / no x-axis |\n| (hello) * |x|o| |\n+-----------------+",
	},
	{
		section: "Canvas & Pathing Cases",
		title:   "Uneven split grid (3 boxes)",
		noBlur:  false,
		input:   "+---+---+\n| A | B |\n+---+   |\n| C |   |\n+---+---+",
	},
	{
		section: "Canvas & Pathing Cases",
		title:   "Nested boxes and diamond",
		noBlur:  false,
		input:   "+---------+\n| +-----+ |\n| |  x  | |\n| +-----+ |\n+---------+\n\n  +  \n / \\ \n+   +\n \\ / \n  +  ",
	},
	{
		section: "Adjacent Line Anchor Cases",
		title:   "Horizontal line with two vertical anchors below",
		noBlur:  false,
		input:   "----++----\n    ||",
	},
	{
		section: "Adjacent Line Anchor Cases",
		title:   "Horizontal line with stepped vertical anchors",
		noBlur:  false,
		input:   "    |\n----++----\n     |",
	},
	{
		section: "Adjacent Line Anchor Cases",
		title:   "Vertical shaft with two adjacent horizontal anchors",
		noBlur:  false,
		input:   "|\n+---\n+---\n|",
	},
	{
		section: "Adjacent Line Anchor Cases",
		title:   "Box wall with two adjacent horizontal lines anchored",
		noBlur:  false,
		input:   "+--+\n|  +----\n|  +----\n+--+",
	},
	{
		section: "Adjacent Line Anchor Cases",
		title:   "Double bridge between two boxes",
		noBlur:  false,
		input:   "+--+  +--+\n|  +--+  |\n|  +--+  |\n+--+  +--+",
	},
	{
		section: "Adjacent Line Anchor Cases",
		title:   "Staggered touching boxes",
		noBlur:  false,
		input:   "+--+\n|  |+--+\n|  ||  |\n+--++--+",
	},
	{
		section: "SVG Styling & Feature Cases",
		title:   "Dashed border with rounded corner",
		noBlur:  false,
		input:   "+--.\n|Hi:\n+--+",
	},
	{
		section: "SVG Styling & Feature Cases",
		title:   "Dark box fill with automatic contrasting white text",
		noBlur:  false,
		input:   ".-----.\n|[a]  |\n'-----'\n\n[a]: {\"fill\":\"#000000\"}",
	},
	{
		section: "SVG Styling & Feature Cases",
		title:   "Label replacement with delref",
		noBlur:  false,
		input:   ".-----.\n|[a]  |\n'-----'\n\n[a]: {\"fill\":\"#000000\",\"a2s:label\":\"abcd\",\"a2s:delref\":1}",
	},
	{
		section: "SVG Styling & Feature Cases",
		title:   "Clickable xlink:href link",
		noBlur:  false,
		input:   " foo\n[1,0]: {\"a2s:delref\":1, \"a2s:link\":\"https://github.com/asciitosvg/asciitosvg\"}",
	},
	{
		section: "SVG Styling & Feature Cases",
		title:   "Untagged inner box inside dark outer box",
		noBlur:  false,
		input:   ".-----------.\n|[outer]    |\n|  .-----.  |\n|  | hi  |  |\n|  '-----'  |\n'-----------'\n\n[outer]: {\"fill\":\"#000000\",\"a2s:delref\":1}",
	},
	{
		section: "SVG Styling & Feature Cases",
		title:   "Inward and outward arrow directions",
		noBlur:  false,
		input:   ">---+\n    |\n    ^\n\n+---<",
	},
	{
		section: "SVG Styling & Feature Cases",
		title:   "No drop-shadow blur (-b)",
		noBlur:  true,
		input:   "+-+\n| |\n+-+",
	},
	{
		section: "SVG Styling & Feature Cases",
		title:   "Numeric JSON attribute (stroke-width)",
		noBlur:  false,
		input:   "+---+\n|[a]|\n+---+\n[a]: {\"stroke-width\":4,\"a2s:delref\":1}",
	},
	{
		section: "Legacy Stress Test (odd_lines.txt)",
		title:   "PHP odd_lines.txt composite diagram",
		noBlur:  false,
		input:   "                 blork!\n     +---+   +---+                     ^               ^\n     |   |   |   |  blork!           -+|              /\n   --+   +---+   +-->            |    ||             /\n         |                <------+----++-->         +\n   .-.---+----.                  |    ||            |\n   | |   |    |                  v    v+------>     |\n   | +---+----+                                     v\n   | |   v    +------------>\n   | |        |            .-------+--------.\n   '-'--------'            |   some text    |\n                           +----------------+\n       <--.                | * more text    |\n          |                | * other text   |\n          |                | * that's all   |\n                           '----------------'\n.---------.\n|         | +------------------------------+\n|         | |     .-------------.          |   +\n'---------'       |[1] ^        |          |  / \\\n                  |     \\  .----'            /   \\\n    +             |      \\ |                /     +\n   / \\            #----.  \\'----.                /\n  /   \\                |   \\    |     |     \\   /\n       \\               '----+---'     |      \\ /\n        \\                    \\     +--+   ----+-----\n  +------+-----------------+  \\   /        /\n  |       \\                |   \\ /        /\n  +----    \\                    +        /\n\n\n[1]: {\"fill\":\"#933\",\"a2s:delref\":true}",
	},
}

func TestRenderGallery(t *testing.T) {
	dimRE := regexp.MustCompile(`<svg width="(\d+)px" height="(\d+)px"`)

	var outDir string
	hasRsvg := false
	if *renderGallery {
		outDir = filepath.Join("testdata", "rendered")
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			t.Fatalf("creating %s: %v", outDir, err)
		}
		if _, err := exec.LookPath("rsvg-convert"); err == nil {
			hasRsvg = true
		}
	}

	var md strings.Builder
	md.WriteString("# ASCIIToSVG Rendered Test Cases\n\nGenerated via `go test -run TestRenderGallery -render-gallery`.\n")
	lastSection := ""

	for i, tc := range galleryCases {
		c, err := NewCanvas([]byte(tc.input), 9, tc.noBlur)
		if err != nil {
			t.Fatalf("case %d (%s): NewCanvas failed: %v", i, tc.title, err)
		}
		svgBytes := CanvasToSVG(c, tc.noBlur, "", 9, 16)
		var doc interface{}
		if err := xml.Unmarshal(svgBytes, &doc); err != nil {
			t.Fatalf("case %d (%s): invalid SVG XML: %v", i, tc.title, err)
		}

		if !*renderGallery {
			continue
		}

		if tc.section != lastSection {
			lastSection = tc.section
			fmt.Fprintf(&md, "\n## %s\n", tc.section)
		}

		svgName := fmt.Sprintf("case_%02d.svg", i)
		svgPath := filepath.Join(outDir, svgName)
		if err := os.WriteFile(svgPath, svgBytes, 0o644); err != nil {
			t.Fatalf("writing %s: %v", svgPath, err)
		}

		imgTarget := svgName
		if hasRsvg {
			dm := dimRE.FindSubmatch(svgBytes)
			if dm != nil {
				w, _ := strconv.Atoi(string(dm[1]))
				h, _ := strconv.Atoi(string(dm[2]))
				const zoom, pad, minPageW = 2, 8, 640
				pageW := w + 2*pad
				if pageW < minPageW {
					pageW = minPageW
				}
				pageH := h + 2*pad
				pngName := fmt.Sprintf("case_%02d.png", i)
				pngPath := filepath.Join(outDir, pngName)
				cmd := exec.Command(
					"rsvg-convert",
					"-b", "white",
					"-z", strconv.Itoa(zoom),
					"--top", fmt.Sprintf("%dpx", pad*zoom),
					"--left", fmt.Sprintf("%dpx", pad*zoom),
					"--page-width", fmt.Sprintf("%dpx", pageW*zoom),
					"--page-height", fmt.Sprintf("%dpx", pageH*zoom),
					"-o", pngPath,
					svgPath,
				)
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("rsvg-convert case %d: %v: %s", i, err, string(out))
				}
				imgTarget = pngName
			}
		}

		fmt.Fprintf(&md, "\n### %d. %s\n\n```text\n%s\n```\n\n![%d. %s](testdata/rendered/%s)\n",
			i, tc.title, tc.input, i, tc.title, imgTarget)
	}

	if *renderGallery {
		if err := os.WriteFile("TEST_CASES.md", []byte(md.String()), 0o644); err != nil {
			t.Fatalf("writing TEST_CASES.md: %v", err)
		}
	}
}
