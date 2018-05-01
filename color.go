// Copyright 2012 - 2018 The ASCIIToSVG Contributors
// All rights reserved.

package asciitosvg

import (
	"fmt"
	"strconv"
)

func parseHexColor(c string) (r, g, b int, err error) {
	var pr, pg, pb int64

	switch len(c) {
	case 4:
		pr, err = strconv.ParseInt(string(c[1]), 16, 0)
		if err != nil {
			return 0, 0, 0, err
		}

		pg, err = strconv.ParseInt(string(c[2]), 16, 0)
		if err != nil {
			return 0, 0, 0, err
		}

		pb, err = strconv.ParseInt(string(c[3]), 16, 0)
		if err != nil {
			return 0, 0, 0, err
		}

		pr *= 17
		pg *= 17
		pb *= 17
	case 7:
		pr, err = strconv.ParseInt(string(c[1:3]), 16, 0)
		if err != nil {
			return 0, 0, 0, err
		}

		pg, err = strconv.ParseInt(string(c[3:5]), 16, 0)
		if err != nil {
			return 0, 0, 0, err
		}

		pb, err = strconv.ParseInt(string(c[5:7]), 16, 0)
		if err != nil {
			return 0, 0, 0, err
		}

	default:
		return 0, 0, 0, fmt.Errorf("color '%s' not of valid length", c)
	}

	r, g, b = int(pr), int(pg), int(pb)

	return
}

// colorToRGB matches a color string and returns its RGB components.
func colorToRGB(c string) (r, g, b int, err error) {
	if c[0] == '#' {
		return parseHexColor(c)
	}

	return 0, 0, 0, fmt.Errorf("color '%s' can't be parsed", c)
}

// textColor returns an accessible text color to use on top of a supplied background color. The
// formula used for calculating whether the contrast is accessible comes from a W3 working group
// paper on accessibility at http://www.w3.org/TR/AERT. The recommended contrast is a brightness
// difference of at least 125 and a color difference of at least 500. Folks can style their colors
// as they like, but our default text color is black, so the color difference for text is just the
// sum of the components.
func textColor(c string) (string, error) {
	r, g, b, err := colorToRGB(c)
	if err != nil {
		return "#000", err
	}

	brightness := (r*299 + g*587 + b*114) / 1000
	difference := r + g + b
	if brightness < 125 && difference < 500 {
		return "#fff", nil
	}

	return "#000", nil
}
