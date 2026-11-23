// Copyright 2012 - 2018 The ASCIIToSVG Contributors
// All rights reserved.

// The a2s tool generates SVG output given an ASCII diagram input.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/asciitosvg/asciitosvg"
)

const logo = ` .-------------------------.
 |                         |
 | .---.-. .-----. .-----. |
 | | .-. | +-->  | |  <--+ |
 | | '-' | |  <--+ +-->  | |
 | '---'-' '-----' '-----' |
 |  ascii     2      svg   |
 |                         |
 '-------------------------'

https://github.com/asciitosvg

[1,0]: {"fill":"#88d","a2s:delref":1}
`

func versionString() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "devel"
	}
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" && len(s.Value) >= 7 {
			return "devel-" + s.Value[:7]
		}
	}
	return "devel"
}

// normalizeArgs rewrites legacy PHP-style attached "-sX,Y" flags into "-s=X,Y"
// so Go's standard flag parser accepts both "-s 9,16" and "-s9,16".
func normalizeArgs(args []string) []string {
	out := make([]string, len(args))
	copy(out, args)
	for i, a := range out {
		if a == "--" {
			break
		}
		if strings.HasPrefix(a, "-s") && len(a) > 2 && a[2] != '=' {
			out[i] = "-s=" + a[2:]
		}
	}
	return out
}

func run(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet(prog, flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "%s\n", logo)
		fmt.Fprintf(stderr, "Usage of %s:\n", prog)
		fs.PrintDefaults()
	}

	in := fs.String("i", "-", "Path to input text file. If set to \"-\" (hyphen), stdin is used.")
	out := fs.String("o", "-", "Path to output SVG file. If set to \"-\" (hyphen), stdout is used.")
	noBlur := fs.Bool("b", false, "Disable drop-shadow blur.")
	font := fs.String("f", "Consolas,Monaco,Anonymous Pro,Anonymous,Bitstream Vera Sans Mono,Liberation Mono,DejaVu Sans Mono,monospace", "Font family to use.")
	scaleX := fs.Int("x", 9, "X grid scale in pixels.")
	scaleY := fs.Int("y", 16, "Y grid scale in pixels.")
	fs.Func("s", "X and Y grid scale in pixels formatted as \"X,Y\" (shorthand for -x and -y).", func(val string) error {
		parts := strings.Split(val, ",")
		if len(parts) != 2 {
			return fmt.Errorf("invalid scaling factor %q (expected X,Y)", val)
		}
		sx, errX := strconv.Atoi(strings.TrimSpace(parts[0]))
		sy, errY := strconv.Atoi(strings.TrimSpace(parts[1]))
		if errX != nil || errY != nil || sx <= 0 || sy <= 0 {
			return fmt.Errorf("invalid scaling factor %q (expected positive integers X,Y)", val)
		}
		*scaleX = sx
		*scaleY = sy
		return nil
	})
	tabWidth := fs.Int("t", 8, "Tab width.")
	doLogo := fs.Bool("L", false, "Generate SVG of the a2s logo.")
	showVersion := fs.Bool("v", false, "Print version information and exit.")

	if err := fs.Parse(normalizeArgs(args)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if *showVersion {
		_, err := fmt.Fprintf(stdout, "a2s %s\n", versionString())
		return err
	}

	if *scaleX <= 0 || *scaleY <= 0 {
		return fmt.Errorf("grid scale must be positive (got %dx%d)", *scaleX, *scaleY)
	}

	var input []byte
	var err error
	if *doLogo {
		input = []byte(logo)
	} else {
		if *in == "-" {
			input, err = io.ReadAll(stdin)
		} else {
			input, err = os.ReadFile(*in)
		}
	}
	if err != nil {
		return err
	}

	canvas, err := asciitosvg.NewCanvas(input, *tabWidth, *noBlur)
	if err != nil {
		return err
	}
	svg := asciitosvg.CanvasToSVG(canvas, *noBlur, *font, *scaleX, *scaleY)
	if *out == "-" {
		_, err := stdout.Write(svg)
		return err
	}
	return os.WriteFile(*out, svg, 0666)
}

func main() {
	if err := run(os.Args[0], os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "a2s: %s\n", err)
		os.Exit(1)
	}
}
