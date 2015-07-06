// Copyright 2012 - 2015 The ASCIIToSVG Contributors
// All rights reserved.

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/asciitosvg/asciitosvg"
)

const logo = `.-------------------------.
|                         |
| .---.-. .-----. .-----. |
| | .-. | +-->  | |  <--| |
| | '-' | |  <--| +-->  | |
| '---'-' '-----' '-----' |
|  ascii     2      svg   |
|                         |
'-------------------------'
`

func mainImpl() error {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s\n", logo)
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}

	in := flag.String("i", "-", "Path to input text file. If set to \"-\" (hyphen), stdin is used.")
	out := flag.String("o", "-", "Path to output SVG file. If set to \"-\" (hyphen), stdout is used.")
	noBlur := flag.Bool("b", false, "Disable drop-shadow blur.")
	font := flag.String("f", "Consolas,Monaco,Anonymous Pro,Anonymous,Bitstream Sans Mono,monospace", "font family to use")
	scaleX := flag.Int("x", 9, "X grid scale in pixels.")
	scaleY := flag.Int("y", 16, "Y grid scale in pixels.")
	flag.Parse()

	var input []byte
	var err error
	if *in == "-" {
		input, err = ioutil.ReadAll(os.Stdin)
	} else {
		input, err = ioutil.ReadFile(*in)
	}
	if err != nil {
		return err
	}

	canvas := asciitosvg.NewCanvas(input)
	boxes := canvas.FindObjects()
	svg := boxes.ToSVG(*noBlur, *font, *scaleX, *scaleY)
	if *out == "-" {
		_, err := os.Stdout.Write(svg)
		return err
	}
	return ioutil.WriteFile(*out, svg, 0666)
}

func main() {
	if err := mainImpl(); err != nil {
		fmt.Fprintf(os.Stderr, "a2s: %s\n", err)
		os.Exit(1)
	}
}
