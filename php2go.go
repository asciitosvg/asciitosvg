// +build none

// php2go is Copyright (c) 2015 by Mateusz Czapliński <czapkofan@gmail.com>
// Licensed under GNU GPL v2+

package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type Matcher struct {
	Text string
	M    []string
}

func (m *Matcher) Find(pattern string) bool {
	re := regexp.MustCompile(pattern)
	m.M = re.FindStringSubmatch(m.Text)
	return len(m.M) > 0
}

func (m *Matcher) ReplaceAll(pattern, replacement string) {
	re := regexp.MustCompile(pattern)
	m.Text = re.ReplaceAllString(m.Text, replacement)
}

func publicize(name string) string {
	return strings.ToUpper(string(name[0])) + name[1:]
}

func main() {
	class := ""
	replacements := map[string]string{}

	scan := bufio.NewScanner(os.Stdin)
	for scan.Scan() {
		line := Matcher{Text: scan.Text()}
		// FIXME: rescue strings and comments
		line.ReplaceAll(`\$`, ``)
		line.ReplaceAll(`->`, `.`)
		line.ReplaceAll(`<<<SVG`, "`")
		line.ReplaceAll(`^SVG;$`, "`")
		line.ReplaceAll(`\.=`, `+=`)
		line.ReplaceAll(`\bnew\s+`, `New`)
		line.ReplaceAll(`\bnull\b`, `nil`)
		line.ReplaceAll(`\bself::`, class+`::`)
		// line.ReplaceAll(`\b__construct\b`, `New`+class)
		for before, after := range replacements {
			line.ReplaceAll(before, after)
		}

		// TODO: add 'foreach(foo as bar => baz)' translation
		// TODO: add 'expr1[] = expr2' translation to 'expr1 = append(expr1, expr2)'
		// TODO: add '__construct' translation to 'NewCLASS(...) *CLASS'
		// TODO: translate "{$expr}" and "$expr" string interpolations
		// TODO: FUTURE: stop translation in '<<<SVG' ... 'SVG' region, if possible
		// TODO: FUTURE: stop translation in comment blocks

		switch {

		// class X -> type X struct
		case line.Find(`^class\s+(\w+)`):
			class = line.M[1]
			// replacements["self::"] = class + "_"
			fmt.Printf("type %s struct%s\n",
				class, line.Text[len(line.M[0]):])

		case line.Find(`^}\s*$`):
			class = ""
			fmt.Println(line.Text)

		// static fields/methods
		case line.Find(`^(\s*)(public|private)\s+static\s+(function\s+)?(\w+)`):
			oldname := fmt.Sprintf(`\b%s::%s\b`, class, line.M[4])
			if line.M[2] == "public" {
				line.M[4] = publicize(line.M[4])
			}
			newname := fmt.Sprintf("%s_%s", class, line.M[4])
			replacements[oldname] = newname
			kind := "var"
			if strings.HasPrefix(line.M[3], "function") {
				kind = "func"
			}
			fmt.Printf("%s%s %s%s\n",
				line.M[1], kind, newname, line.Text[len(line.M[0]):])

		case line.Find(`^(\s*)(public|private)\s+(function\s+)?(\w+)`):
			// TODO(akavel): find .name instead of name?
			oldname := fmt.Sprintf(`\b%s\b`, line.M[4])
			if line.M[2] == "public" {
				line.M[4] = publicize(line.M[4])
			}
			newname := line.M[4]
			replacements[oldname] = newname
			kind := ""
			if strings.HasPrefix(line.M[3], "function") {
				kind = fmt.Sprintf("func (this *%s) ", class)
			}
			fmt.Printf("%s%s%s%s\n",
				line.M[1], kind, newname, line.Text[len(line.M[0]):])

		case line.Find(`^(\s*)foreach\s*\(\s*(.+?)\s+as\s+(\w+)\s*\)`):
			array := line.M[2]
			variable := line.M[3]
			fmt.Printf("%sfor _, %s := range %s%s\n",
				line.M[1], variable, array, line.Text[len(line.M[0]):])

		// case line.Find(`^(\s*)public\s+static\s+

		// print rest of lines intact
		default:
			fmt.Println(line.Text)
		}
	}
}
