// Copyright 2012 - 2018 The ASCIIToSVG Contributors
// All rights reserved.

package asciitosvg

import (
	"fmt"
)

func parseHexColor(c string) (r, g, b int, err error) {
	switch len(c) {
	case 4:
		// Short #rgb or #RGB form
		n, _ := fmt.Sscanf(c, "#%1x%1x%1x", &r, &g, &b)
		if n == 0 {
			n, _ = fmt.Sscanf(c, "#%1X%1X%1X", &r, &g, &b)
			if n != 3 {
				err = fmt.Errorf("color '%s' not valid #rgb or #RGB form", c)
			}
		}
	case 7:
		// Normal ##rrggbb form
		n, _ := fmt.Sscanf(c, "#%02x%02x%02x", &r, &g, &b)
		if n == 0 {
			n, _ = fmt.Sscanf(c, "#%02X%02X%02X", &r, &g, &b)
			if n != 3 {
				err = fmt.Errorf("color '%s' not in valid #rrggbb or #RRGGBB form", c)
			}
		}
	}

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
