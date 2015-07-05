/*
 * a2s: CLI utility for ASCIIToSVG
 * Copyright © 2012 Devon H. O'Dell <devon.odell@gmail.com>
 * Copyright © 2015 Mateusz Czapliński <czapkofan@gmail.com>
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions
 * are met:
 *
 *  o Redistributions of source code must retain the above copyright notice,
 *    this list of conditions and the following disclaimer.
 *  o Redistributions in binary form must reproduce the above copyright notice,
 *    this list of conditions and the following disclaimer in the documentation
 *    and/or other materials provided with the distribution.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
 * AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 * IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
 * ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE
 * LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
 * CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
 * SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
 * INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
 * CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
 * ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
 * POSSIBILITY OF SUCH DAMAGE.
 */

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

func main() {
	err := run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s", err)
		os.Exit(1)
	}
}

func run() error {
	noblur := flag.Bool("b", false, "Disable drop-shadow blur.")
	fontfamily := flag.String("f", "", "Font family to use.")
	help := flag.Bool("h", false, "This usage screen.")
	input := flag.String("i", "-", `Path to input text file. If unspecified, or set to "-" (hyphen), stdin is used.`)
	output := flag.String("o", "-", `Path to output SVG file. If unspecified or set to "-" (hyphen), stdout is used.`)
	scale := flag.String("s", "9,16", `Grid scale in pixels. If unspecified, each grid unit on the X axis is set to 9 pixels; each grid unit on the Y axis is 16 pixels.`)
	flag.Parse()
	if *help {
		flag.Usage()
		os.Exit(1)
	}

	var err error
	instream := os.Stdin
	if *input != "-" {
		instream, err = os.Open(*input)
		if err != nil {
			return err
		}
		// TODO(akavel): defer .Close(), or ignore?
	}
	inbuf, err := ioutil.ReadAll(instream)
	if err != nil {
		return err
	}

	outstream := os.Stdout
	if *output != "-" {
		outstream, err = os.OpenFile(*output, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
		if err != nil {
			return err
		}
	}
	defer outstream.Close() // TODO(akavel): do we need this?

	scales := strings.Split(*scale, ",")
	var scalex, scaley int
	if len(scales) != 2 {
		return fmt.Errorf("Invalid scaling factor %q", scale)
	}
	scalex, err = strconv.Atoi(scales[0])
	if err != nil {
		return err
	}
	scaley, err = strconv.Atoi(scales[1])
	if err != nil {
		return err
	}

	o := NewASCIIToSVG(inbuf)
	o.DisableBlurDropShadow = *noblur
	if *fontfamily != "" {
		o.FontFamily = *fontfamily
	}
	o.SetDimensionScale(Coord(scalex), Coord(scaley))
	o.ParseGrid()

	_, err = outstream.Write([]byte(o.Render()))
	if err != nil {
		return err
	}

	return nil
}
