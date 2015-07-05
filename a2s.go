/*
 * ASCIIToSVG.php: ASCII diagram . SVG art generator.
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
 *
 */

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// include 'svg-path.lex.php';
// include 'colors.php';

var A2S_colors = map[string]string{}

func strpos(haystack, needle string, offset int) int {
	i := strings.Index(haystack[offset:], needle)
	if i == -1 {
		return -1
	}
	return offset + i
}
func str_replace(search, replace, subject string) string {
	return strings.Replace(subject, search, replace, -1)
}
func atof(s string) Coord {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		panic(err.Error())
	}
	return Coord(f)
}
func json_decode(buf []byte, _ bool) map[string]string {
	result := map[string]string{}
	// FIXME(akavel): keep error and do something with it (log?)
	json.Unmarshal(buf, &result)
	return result
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func strtolower(s string) string {
	return strings.ToLower(s)
}
func hexdec(s string) int {
	// TODO(akavel): do something with error; log?
	i, _ := strconv.ParseInt(s, 16, 32)
	return int(i)
}
func str_repeat(s string, n int) string {
	return strings.Repeat(s, n)
}

type Coord float64

/*
 * Scale is a singleton class that is instantiated to apply scale
 * transformations on the text . canvas grid geometry. We could probably use
 * SVG's native scaling for this, but I'm not sure how yet.
 */
type Scale struct {
	XScale Coord
	YScale Coord
}

var Scale_instance = Scale{}

func Scale_GetInstance() *Scale {
	return &Scale_instance
}

func (this *Scale) SetScale(x, y Coord) {
	o := Scale_GetInstance()
	o.XScale = x
	o.YScale = y
}

/*
 * CustomObjects allows users to create their own custom SVG paths and use
 * them as box types with a2s:type references.
 *
 * Paths must have width and height set, and must not span multiple lines.
 * Multiple paths can be specified, one path per line. All objects must
 * reside in the same directory.
 *
 * File operations are horribly slow, so we make a best effort to avoid
 * as many as possible:
 *
 *  * If the directory mtime hasn't changed, we attempt to load our
 *    objects from a cache file.
 *
 *  * If this file doesn't exist, can't be read, or the mtime has
 *    changed, we scan the directory and update files that have changed
 *    based on their mtime.
 *
 *  * We attempt to save our cache in a temporary directory. It's volatile
 *    but also requires no configuration.
 *
 * We could do a bit better by utilizing APC's shared memory storage, which
 * would help greatly when running on a server.
 *
 * Note that the path parser isn't foolproof, mostly because PHP isn't the
 * greatest language ever for implementing a parser.
 */
type CustomObjectsType map[string]struct{}

var CustomObjects_Objects = CustomObjectsType{}

/*
 * Closures / callable function names / whatever for integrating non-default
 * loading and storage functionality.
 */
var CustomObjects_LoadCacheFn func() CustomObjectsType = nil
var CustomObjects_StorCacheFn func(CustomObjectsType) = nil
var CustomObjects_LoadObjsFn func() CustomObjectsType = nil // TODO(akavel): ?

// func CustomObjects_LoadObjects() {
//   cacheFile := os.Getenv("HOME") + "/.a2s/objcache"
//   dir := os.Getenv("HOME") + "/.a2s/objects"

// 	if nil!=(CustomObjects_LoadCacheFn) {
// 		/*
// 		 * Should return exactly what was given to the storCacheFn when it was
// 		 * last called, or nil if nothing can be loaded.
// 		 */
//      fn := CustomObjects_LoadCacheFn
// 		CustomObjects_Objects = fn()
// 		return
// 	}
// 		if is_readable(cacheFile) && is_readable(dir) {
// 			cacheTime = filemtime(cacheFile)

// 			if filemtime(dir) <= filemtime(cacheFile) {
// 				CustomObjects_Objects = unserialize(file_get_contents(cacheFile))
// 				return
// 			}
// 		} else if file_exists(cacheFile) {
// 			return
// 		}

// 	if nil!=(CustomObjects_LoadObjsFn) {
// 		/*
// 		 * Returns an array of arrays of path information. The innermost arrays
// 		 * (containing the path information) contain the path name, the width of
// 		 * the bounding box, the height of the bounding box, and the path
// 		 * command. This interface does *not* want the path's XML tag. An array
// 		 * returned from here containing two objects that each have 1 line would
// 		 * look like:
// 		 *
// 		 * array (
// 		 *   array (
// 		 *     name => 'pathA',
// 		 *     paths => array (
// 		 *       array ('width' => 10, 'height' => 10, 'path' => 'M 0 0 L 10 10'),
// 		 *       array ('width' => 10, 'height' => 10, 'path' => 'M 0 10 L 10 0'),
// 		 *     ),
// 		 *   ),
// 		 *   array (
// 		 *     name => 'pathB',
// 		 *     paths => array (
// 		 *       array ('width' => 10, 'height' => 10, 'path' => 'M 0 5 L 5 10'),
// 		 *       array ('width' => 10, 'height' => 10, 'path' => 'M 5 10 L 10 5'),
// 		 *     ),
// 		 *   ),
// 		 * );
// 		 */
// 		fn = CustomObjects_LoadObjsFn
// 		objs = fn()

// 		i = 0
// 		for _, obj := range objs {
// 			for _, path := range obj["paths"] {
// 				CustomObjects_Objects[obj["name"]][i]["width"] = path["width"]
// 				CustomObjects_Objects[obj["name"]][i]["height"] = path["height"]
// 				CustomObjects_Objects[obj["name"]][i]["path"] =
// 					CustomObjects_parsePath(path["path"])
// 				i++
// 			}
// 		}
// 	} else {
// 		ents = scandir(dir)
// 		for _, ent := range ents {
// 			file = dir + "/" + ent
// 			base = substr(ent, 0, -5)
// 			if substr(ent, -5) == ".path" && is_readable(file) {
// 				if isset(CustomObjects_Objects[base]) &&
// 					filemtime(file) <= CustomObjects_cacheTime {
// 					continue
// 				}

// 				lines = file(file)

// 				i = 0
// 				for _, line := range lines {
// 					preg_match(`/width="(\d+)/`, line, m)
// 					width = m[1]
// 					preg_match(`/height="(\d+)/`, line, m)
// 					height = m[1]
// 					preg_match(`/d="([^"]+)"/`, line, m)
// 					path = m[1]

// 					CustomObjects_Objects[base][i][`width`] = width
// 					CustomObjects_Objects[base][i][`height`] = height
// 					CustomObjects_Objects[base][i][`path`] = CustomObjects_parsePath(path)
// 					i++
// 				}
// 			}
// 		}
// 	}

// 	if is_callable(CustomObjects_StorCacheFn) {
// 		fn = CustomObjects_StorCacheFn
// 		fn(CustomObjects_Objects)
// 	} else {
// 		file_put_contents(cacheFile, serialize(CustomObjects_Objects))
// 	}
// }

// func CustomObjects_parsePath(path) {
//   stream = fopen("data://text/plain,"+path, 'r');

//   P = NewA2S_SVGPathParser();
//   S = NewA2S_Yylex(stream);

//   while (t = S.nextToken()) {
//     P.A2S_SVGPath(t.type, t);
//   }
//   /* Force shift/reduce of last token. */
//   P.A2S_SVGPath(0);

//   fclose(stream);

//   cmdArr = array();
//   i = 0;
//   for _, cmd := range P.commands {
//     for _, arg := range cmd {
//       arg = (array)arg;
//       cmdArr[i][] = arg['value'];
//     }
//     i++;
//   }

//   return cmdArr;
// }

type Point_Flags int

/*
 * All lines and polygons are represented as a series of point coordinates
 * along a path. Points can have different properties; markers appear on
 * edges of lines and control points denote that a bezier curve should be
 * calculated for the corner represented by this point.
 */
type Point struct {
	GridX int
	GridY int

	X Coord
	Y Coord

	Flags Point_Flags
}

const Point_POINT = 0x1
const Point_CONTROL = 0x2
const Point_SMARKER = 0x4
const Point_IMARKER = 0x8
const Point_TICK = 0x10
const Point_DOT = 0x20

func NewPoint(X, Y Coord) Point {
	this := Point{}
	this.Flags = 0

	s := Scale_GetInstance()
	this.X = (X * s.XScale) + (s.XScale / 2)
	this.Y = (Y * s.YScale) + (s.YScale / 2)

	this.GridX = int(.5 + X)
	this.GridY = int(.5 + Y)
	return this
}

type Object interface {
	Render() string
}

/*
 * Groups objects together and sets common properties for the objects in the
 * group.
 */
type SVGGroup struct {
	groups     map[string][]Object
	curGroup   string
	groupStack []string
	options    map[string]map[string]string
}

func NewSVGGroup() *SVGGroup {
	return &SVGGroup{
		groups:     map[string][]Object{},
		groupStack: []string{},
		options:    map[string]map[string]string{},
	}
}

func (this *SVGGroup) GetGroup(groupName string) []Object {
	return this.groups[groupName]
}

func (this *SVGGroup) PushGroup(groupName string) {
	if nil == (this.groups[groupName]) {
		this.groups[groupName] = []Object{}
		this.options[groupName] = map[string]string{}
	}

	this.groupStack = append(this.groupStack, groupName)
	this.curGroup = groupName
}

func (this *SVGGroup) PopGroup() {
	/*
	 * Remove the last group and fetch the current one. array_pop will return
	 * NULL for an empty array, so this is safe to do when only one element
	 * is left.
	 */
	n := len(this.groupStack)
	if n < 2 {
		this.groupStack = this.groupStack[:0]
		return
	}
	this.curGroup = this.groupStack[n-2]
	this.groupStack = this.groupStack[:n-2]
}

func (this *SVGGroup) AddObject(o Object) {
	this.groups[this.curGroup] = append(
		this.groups[this.curGroup], o)
}

func (this *SVGGroup) SetOption(opt, val string) {
	this.options[this.curGroup][opt] = val
}

func (this *SVGGroup) Render() string {
	out := ``

	for groupName, objects := range this.groups {
		out += "<g id=\"" + groupName + "\" "
		for opt, val := range this.options[groupName] {
			if strpos(opt, `a2s:`, 0) == 0 {
				continue
			}
			out += opt + "=\"" + val + "\" "
		}
		out += ">\n"

		for _, obj := range objects {
			out += obj.Render()
		}

		out += "</g>\n"
	}

	return out
}

var _SVGPath_id = 0

func SVGPath_id() string {
	s := fmt.Sprintf("%d", _SVGPath_id)
	_SVGPath_id++
	return s
}

/*
 * The Path class represents lines and polygons.
 */
type SVGPath struct {
	options map[string]string
	points  []Point
	ticks   []Point
	Flags   Point_Flags
	text    []*SVGText
	name    string
}

const SVGPath_CLOSED = 0x1

func NewSVGPath() *SVGPath {
	return &SVGPath{
		options: map[string]string{},
		points:  []Point{},
		ticks:   []Point{},
		Flags:   0,
		text:    []*SVGText{},
		name:    SVGPath_id(),
	}
}

/*
 * Making sure that we always started at the top left coordinate
 * makes so many things so much easier. First, find the lowest Y
 * position. Then, of all matching Y positions, find the lowest X
 * position. This is the top left.
 *
 * As far as the points are considered, they're definitely on the
 * top somewhere, but not necessarily the most left. This could
 * happen if there was a corner connector in the top edge (perhaps
 * for a line to connect to). Since we couldn't turn right there,
 * we have to try now.
 *
 * This should only be called when we close a polygon.
 */
func (this *SVGPath) OrderPoints() {
	pPoints := len(this.points)

	minY := this.points[0].Y
	minX := this.points[0].X
	minIdx := 0
	for i := 1; i < pPoints; i++ {
		if this.points[i].Y <= minY {
			minY = this.points[i].Y

			if this.points[i].X < minX {
				minX = this.points[i].X
				minIdx = i
			}
		}
	}

	/*
	 * If our top left isn't at the 0th index, it is at the end. If
	 * there are bits after it, we need to cut those and put them at
	 * the front.
	 */
	if minIdx != 0 {
		this.points = append(this.points[minIdx:], this.points[:minIdx]...)
	}
}

/*
 * Useful for recursive walkers when speculatively trying a direction.
 */
func (this *SVGPath) PopPoint() {
	if len(this.points) > 0 {
		this.points = this.points[:len(this.points)-1]
	}
}

// FIXME(akavel): func (this *SVGPath) AddPoint(X, Y, Flags = Point_POINT) {
func (this *SVGPath) AddPoint(X, Y Coord, Flags Point_Flags) bool {
	p := NewPoint(X, Y)

	/*
	 * If we attempt to add our original point back to the path, the polygon
	 * must be closed.
	 */
	if len(this.points) > 0 {
		if this.points[0].X == p.X && this.points[0].Y == p.Y {
			this.Flags |= SVGPath_CLOSED
			return true
		}

		/*
		 * For the purposes of this library, paths should never intersect each
		 * other. Even in the case of closing the polygon, we do not store the
		 * final coordinate twice.
		 */
		for _, point := range this.points {
			if point.X == p.X && point.Y == p.Y {
				return true
			}
		}
	}

	p.Flags |= Flags
	this.points = append(this.points, p)

	return false
}

/*
 * It's useful to be able to know the points in a shape.
 */
func (this *SVGPath) GetPoints() []Point {
	return this.points
}

/*
 * Add a marker to a line. The third argument specifies which marker to use,
 * and this depends on the orientation of the line. Due to the way the line
 * parser works, we may have to use an inverted representation.
 */
func (this *SVGPath) AddMarker(X, Y Coord, t Point_Flags) {
	p := NewPoint(X, Y)
	p.Flags |= t
	this.points = append(this.points, p)
}

func (this *SVGPath) AddTick(X, Y Coord, t Point_Flags) {
	p := NewPoint(X, Y)
	p.Flags |= t
	this.ticks = append(this.ticks, p)
}

/*
 * Is this path closed?
 */
func (this *SVGPath) IsClosed() bool {
	return (this.Flags & SVGPath_CLOSED) != 0
}

func (this *SVGPath) AddText(t *SVGText) {
	this.text = append(this.text, t)
}

func (this *SVGPath) GetText() []*SVGText {
	return this.text
}

func (this *SVGPath) SetID(id string) {
	this.name = str_replace(` `, `_`, str_replace(`"`, `_`, id))
}

func (this *SVGPath) GetID() string {
	return this.name
}

/*
 * Set options as a JSON string. Specified as a merge operation so that it
 * can be called after an individual SetOption call.
 */
func (this *SVGPath) SetOptions(opt map[string]string) {
	for k, v := range opt {
		this.options[k] = v
	}
}

func (this *SVGPath) SetOption(opt, val string) {
	this.options[opt] = val
}

func (this *SVGPath) GetOption(opt string) string {
	return this.options[opt]
}

/*
 * Does the given point exist within this polygon? Since we can
 * theoretically have some complex concave and convex polygon edges in the
 * same shape, we need to do a full point-in-polygon test. This algorithm
 * seems like the standard one. See: http://alienryderflex.com/polygon/
 */
func (this *SVGPath) HasPoint(X, Y Coord) bool {
	if this.IsClosed() == false {
		return false
	}

	oddNodes := false

	bound := len(this.points)
	for i, j := 0, len(this.points)-1; i < bound; i++ {
		if (Coord(this.points[i].GridY) < Y && Coord(this.points[j].GridY) >= Y ||
			Coord(this.points[j].GridY) < Y && Coord(this.points[i].GridY) >= Y) &&
			(Coord(this.points[i].GridX) <= X || Coord(this.points[j].GridX) <= X) {
			if Coord(this.points[i].GridX)+(Y-Coord(this.points[i].GridY))/
				Coord(this.points[j].GridY-this.points[i].GridY)*
				Coord(this.points[j].GridX-this.points[i].GridX) < X {
				oddNodes = !oddNodes
			}
		}

		j = i
	}

	return oddNodes
}

type Matrix [][]Coord

/*
 * Apply a matrix transformation to the coordinates (X, Y). The
 * multiplication is implemented on the matrices:
 *
 * | a b c |   | X |
 * | d e f | * | Y |
 * | 0 0 1 |   | 1 |
 *
 * Additional information on the transformations and what each R,C in the
 * transformation matrix represents, see:
 *
 * http://www.w3.org/TR/SVG/coords.html#TransformMatrixDefined
 */
func (this *SVGPath) matrixTransform(matrix Matrix, X, Y Coord) (Coord, Coord) {
	xyMat := Matrix{{X}, {Y}, {1}}
	newXY := Matrix{{0}, {0}, {0}}

	for i := 0; i < 3; i++ {
		for j := 0; j < 1; j++ {
			sum := Coord(0)

			for k := 0; k < 3; k++ {
				sum += matrix[i][k] * xyMat[k][j]
			}

			newXY[i][j] = sum
		}
	}

	/* Return the coordinates as a vector */
	return newXY[0][0], newXY[1][0]
	// return array(newXY[0][0], newXY[1][0], newXY[2][0])
}

/*
 * Translate the X and Y coordinates. tX and tY specify the distance to
 * transform.
 */
func (this *SVGPath) translateTransform(tX, tY, X, Y Coord) (Coord, Coord) {
	matrix := Matrix{{1, 0, tX}, {0, 1, tY}, {0, 0, 1}}
	return this.matrixTransform(matrix, X, Y)
}

/*
 * Scale transformations are implemented by applying the scale to the X and
 * Y coordinates. One unit in the Newcoordinate system equals s[XY] units
 * in the old system. Thus, if you want to double the size of an object on
 * both axes, you sould call scaleTransform(0.5, 0.5, X, Y)
 */
func (this *SVGPath) scaleTransform(sX, sY, X, Y Coord) (Coord, Coord) {
	matrix := Matrix{{sX, 0, 0}, {0, sY, 0}, {0, 0, 1}}
	return this.matrixTransform(matrix, X, Y)
}

/*
 * Rotate the coordinates around the center point cX and cY. If these
 * are not specified, the coordinate is rotated around 0,0. The angle
 * is specified in degrees.
 */
// FIXME(akavel): func (this *SVGPath) rotateTransform(angle, X, Y, cX = 0, cY = 0) {
func (this *SVGPath) rotateTransform(angle float64, X, Y, cX, cY Coord) (Coord, Coord) {
	angle = angle * (math.Pi / 180.0)
	if cX != 0 || cY != 0 {
		X, Y = this.translateTransform(cX, cY, X, Y)
	}

	matrix := Matrix{{Coord(math.Cos(angle)), Coord(-math.Sin(angle)), 0},
		{Coord(math.Sin(angle)), Coord(math.Cos(angle)), 0},
		{0, 0, 1}}
	X, Y = this.matrixTransform(matrix, X, Y)

	if cX != 0 || cY != 0 {
		X, Y = this.translateTransform(-cX, -cY, X, Y)
	}

	return X, Y
}

/*
 * Skews along the X axis at specified angle. The angle is specified in
 * degrees.
 */
func (this *SVGPath) skewXTransform(angle float64, X, Y Coord) (Coord, Coord) {
	angle = angle * (math.Pi / 180.0)
	matrix := Matrix{{1, Coord(math.Tan(angle)), 0}, {0, 1, 0}, {0, 0, 1}}
	return this.matrixTransform(matrix, X, Y)
}

/*
 * Skews along the Y axis at specified angle. The angle is specified in
 * degrees.
 */
func (this *SVGPath) skewYTransform(angle float64, X, Y Coord) (Coord, Coord) {
	angle = angle * (math.Pi / 180.0)
	matrix := Matrix{{1, 0, 0}, {Coord(math.Tan(angle)), 1, 0}, {0, 0, 1}}
	return this.matrixTransform(matrix, X, Y)
}

/*
 * Apply a transformation to a point p.
 */
func (this *SVGPath) applyTransformToPoint(txf string, p Point, args []Coord) (Coord, Coord) {
	switch txf {
	case `translate`:
		return this.translateTransform(args[0], args[1], p.X, p.Y)

	case `scale`:
		return this.scaleTransform(args[0], args[1], p.X, p.Y)

	case `rotate`:
		if len(args) > 1 {
			return this.rotateTransform(float64(args[0]), p.X, p.Y, args[1], args[2])
		} else {
			return this.rotateTransform(float64(args[0]), p.X, p.Y, 0, 0)
		}

	case `skewX`:
		return this.skewXTransform(float64(args[0]), p.X, p.Y)

	case `skewY`:
		return this.skewYTransform(float64(args[0]), p.X, p.Y)
	}
	panic("transform not implemented: " + txf)
}

/*
 * Apply the transformation function txf to all coordinates on path p
 * providing args as arguments to the transformation function.
 */
func (this *SVGPath) applyTransformToPath(txf string, p map[string][][]string, args []Coord) {
	pathCmds := len(p[`path`])
	curPoint := NewPoint(0, 0)
	var prevType, curType string

	for i := 0; i < pathCmds; i++ {
		cmd := p[`path`][i]

		prevType = curType
		curType = cmd[0]

		switch curType {
		/* Can't transform those */
		case `z`:
		case `Z`:

		case `m`:
			if prevType != `` {
				curPoint.X += atof(cmd[1])
				curPoint.Y += atof(cmd[2])

				X, Y := this.applyTransformToPoint(txf, curPoint, args)
				curPoint.X = X
				curPoint.Y = Y

				cmd[1] = fmt.Sprint(X)
				cmd[2] = fmt.Sprint(Y)
			} else {
				curPoint.X = atof(cmd[1])
				curPoint.Y = atof(cmd[2])

				X, Y := this.applyTransformToPoint(txf, curPoint, args)
				curPoint.X = X
				curPoint.Y = Y

				cmd[1] = fmt.Sprint(X)
				cmd[2] = fmt.Sprint(Y)
				curType = `l`
			}

			break

		case `M`:
			curPoint.X = atof(cmd[1])
			curPoint.Y = atof(cmd[2])

			X, Y := this.applyTransformToPoint(txf, curPoint, args)
			curPoint.X = X
			curPoint.Y = Y

			cmd[1] = fmt.Sprint(X)
			cmd[2] = fmt.Sprint(Y)

			if prevType == `` {
				curType = `L`
			}
			break

		case `l`:
			curPoint.X += atof(cmd[1])
			curPoint.Y += atof(cmd[2])

			X, Y := this.applyTransformToPoint(txf, curPoint, args)
			curPoint.X = X
			curPoint.Y = Y

			cmd[1] = fmt.Sprint(X)
			cmd[2] = fmt.Sprint(Y)

			break

		case `L`:
			curPoint.X = atof(cmd[1])
			curPoint.Y = atof(cmd[2])

			X, Y := this.applyTransformToPoint(txf, curPoint, args)
			curPoint.X = X
			curPoint.Y = Y

			cmd[1] = fmt.Sprint(X)
			cmd[2] = fmt.Sprint(Y)

			break

		case `v`:
			curPoint.Y += atof(cmd[1])
			curPoint.X += 0

			X, Y := this.applyTransformToPoint(txf, curPoint, args)
			curPoint.X = X
			curPoint.Y = Y

			cmd[1] = fmt.Sprint(Y)

			break

		case `V`:
			curPoint.Y = atof(cmd[1])

			X, Y := this.applyTransformToPoint(txf, curPoint, args)
			curPoint.X = X
			curPoint.Y = Y

			cmd[1] = fmt.Sprint(Y)

			break

		case `h`:
			curPoint.X += atof(cmd[1])

			X, Y := this.applyTransformToPoint(txf, curPoint, args)
			curPoint.X = X
			curPoint.Y = Y

			cmd[1] = fmt.Sprint(X)

			break

		case `H`:
			curPoint.X = atof(cmd[1])

			X, Y := this.applyTransformToPoint(txf, curPoint, args)
			curPoint.X = X
			curPoint.Y = Y

			cmd[1] = fmt.Sprint(X)

			break

		case `c`:
			tP := NewPoint(0, 0)
			tP.X = curPoint.X + atof(cmd[1])
			tP.Y = curPoint.Y + atof(cmd[2])
			X, Y := this.applyTransformToPoint(txf, tP, args)
			cmd[1] = fmt.Sprint(X)
			cmd[2] = fmt.Sprint(Y)

			tP.X = curPoint.X + atof(cmd[3])
			tP.Y = curPoint.Y + atof(cmd[4])
			X, Y = this.applyTransformToPoint(txf, tP, args)
			cmd[3] = fmt.Sprint(X)
			cmd[4] = fmt.Sprint(Y)

			curPoint.X += atof(cmd[5])
			curPoint.Y += atof(cmd[6])
			X, Y = this.applyTransformToPoint(txf, curPoint, args)

			curPoint.X = X
			curPoint.Y = Y
			cmd[5] = fmt.Sprint(X)
			cmd[6] = fmt.Sprint(Y)

			break
		case `C`:
			curPoint.X = atof(cmd[1])
			curPoint.Y = atof(cmd[2])
			X, Y := this.applyTransformToPoint(txf, curPoint, args)
			cmd[1] = fmt.Sprint(X)
			cmd[2] = fmt.Sprint(Y)

			curPoint.X = atof(cmd[3])
			curPoint.Y = atof(cmd[4])
			X, Y = this.applyTransformToPoint(txf, curPoint, args)
			cmd[3] = fmt.Sprint(X)
			cmd[4] = fmt.Sprint(Y)

			curPoint.X = atof(cmd[5])
			curPoint.Y = atof(cmd[6])
			X, Y = this.applyTransformToPoint(txf, curPoint, args)

			curPoint.X = X
			curPoint.Y = Y
			cmd[5] = fmt.Sprint(X)
			cmd[6] = fmt.Sprint(Y)

			break

		case `s`:
		case `S`:

		case `q`:
		case `Q`:

		case `t`:
		case `T`:

		case `a`:
			break

		case `A`:
			/*
			 * This radius is relative to the start and end points, so it makes
			 * sense to scale, rotate, or skew it, but not translate it.
			 */
			if txf != `translate` {
				curPoint.X = atof(cmd[1])
				curPoint.Y = atof(cmd[2])
				X, Y := this.applyTransformToPoint(txf, curPoint, args)
				cmd[1] = fmt.Sprint(X)
				cmd[2] = fmt.Sprint(Y)
			}

			curPoint.X = atof(cmd[6])
			curPoint.Y = atof(cmd[7])
			X, Y := this.applyTransformToPoint(txf, curPoint, args)
			curPoint.X = X
			curPoint.Y = Y
			cmd[6] = fmt.Sprint(X)
			cmd[7] = fmt.Sprint(Y)

			break
		}
	}
}

func (this *SVGPath) Render() string {
	// FIXME(akavel): handle empty this.points
	startPoint := this.points[0]
	this.points = this.points[1:]
	endPoint := this.points[len(this.points)-1]

	out := "<g id=\"group" + this.name + "\">\n"

	/*
	 * If someone has specified one of our special object types, we are going
	 * to want to completely override any of the pathing that we would have
	 * done otherwise, but we defer until here to do anything about it because
	 * we need information about the object we're replacing.
	 */
	// if `` != (this.options[`a2s:type`]) &&
	// // isset(CustomObjects_Objects[this.options[`a2s:type`]]) {
	// struct{}{} != (CustomObjects_Objects[this.options[`a2s:type`]]) {
	// object := CustomObjects_Objects[this.options[`a2s:type`]]

	// /* Again, if no fill was specified, specify one. */
	// if `` == (this.options[`fill`]) {
	// 	this.options[`fill`] = `#fff`
	// }

	// /*
	//  * We don't care so much about the area, but we do care about the width
	//  * and height of the object. All of our "custom" objects are implemented
	//  * in 100x100 space, which makes the transformation marginally easier.
	//  */
	// minX := startPoint.X
	// maxX := minX
	// minY := startPoint.Y
	// maxY := minY
	// for _, p := range this.points {
	// 	if p.X < minX {
	// 		minX = p.X
	// 	} else if p.X > maxX {
	// 		maxX = p.X
	// 	}
	// 	if p.Y < minY {
	// 		minY = p.Y
	// 	} else if p.Y > maxY {
	// 		maxY = p.Y
	// 	}
	// }

	// objW := maxX - minX
	// objH := maxY - minY

	// i := 0
	// for _, o := range object {
	// 	id = SVGPath_id()
	// 	out += "\t<path id=\"path" + this.name + "\" d=\""

	// 	oW = o[`width`]
	// 	oH = o[`height`]

	// 	this.applyTransformToPath(`scale`, o, array(objW/oW, objH/oH))
	// 	this.applyTransformToPath(`translate`, o, array(minX, minY))

	// 	for _, cmd := range o[`path`] {
	// 		out += join(` `, cmd) + ` `
	// 	}
	// 	out += `" `

	// 	/* Don't add options to sub-paths */
	// 	if i < 1 {
	// 		for opt, val := range this.options {
	// 			if strpos(opt, `a2s:`, 0) == 0 {
	// 				continue
	// 			}
	// 			out += "opt=\"val\" "
	// 		}
	// 	}
	// 	i++

	// 	out += " />\n"
	// }

	// if count(this.text) > 0 {
	// 	for _, text := range this.text {
	// 		out += "\t" + text.Render() + "\n"
	// 	}
	// }
	// out += "</g>\n"

	// /* Bazinga. */
	// return out
	// }

	/*
	 * Nothing fancy here -- this is just rendering for our standard
	 * polygons.
	 *
	 * Our start point is represented by a single moveto command (unless the
	 * start point is curved) as the shape will be closed with the Z command
	 * automatically if it is a closed shape. If we have a control point, we
	 * have to go ahead and draw the curve.
	 */
	var path string
	if startPoint.Flags&Point_CONTROL != 0 {
		cX := startPoint.X
		cY := startPoint.Y
		sX := startPoint.X
		sY := startPoint.Y + 10
		eX := startPoint.X + 10
		eY := startPoint.Y

		path = "M " + fmt.Sprint(sX) + " " + fmt.Sprint(sY) + " Q " + fmt.Sprint(cX) + " " + fmt.Sprint(cY) + " " + fmt.Sprint(eX) + " " + fmt.Sprint(eY) + " "
	} else {
		path = "M " + fmt.Sprint(startPoint.X) + " " + fmt.Sprint(startPoint.Y) + " "
	}

	prevP := startPoint
	bound := len(this.points)
	for i := 0; i < bound; i++ {
		p := this.points[i]

		/*
		 * Handle quadratic Bezier curves. NOTE: This algorithm for drawing
		 * the curves only works if the shapes are drawn in a clockwise
		 * manner.
		 */
		if p.Flags&Point_CONTROL != 0 {
			/* Our control point is always the original corner */
			cX := p.X
			cY := p.Y

			/* Need next point to determine which way to turn */
			var nP Point
			if i == len(this.points)-1 {
				nP = startPoint
			} else {
				nP = this.points[i+1]
			}

			var sX, sY, eX, eY Coord
			if prevP.X == p.X {
				/*
				 * If we are on the same vertical axis, our starting X coordinate
				 * is the same as the control point coordinate.
				 */
				sX = p.X

				/* Offset start point from control point in the proper direction */
				if prevP.Y < p.Y {
					sY = p.Y - 10
				} else {
					sY = p.Y + 10
				}

				eY = p.Y
				/* Offset end point from control point in the proper direction */
				if nP.X < p.X {
					eX = p.X - 10
				} else {
					eX = p.X + 10
				}
			} else if prevP.Y == p.Y {
				/* Horizontal decisions mirror vertical's above */
				sY = p.Y
				if prevP.X < p.X {
					sX = p.X - 10
				} else {
					sX = p.X + 10
				}

				eX = p.X
				if nP.Y <= p.Y {
					eY = p.Y - 10
				} else {
					eY = p.Y + 10
				}
			}

			path += "L " + fmt.Sprint(sX) + " " + fmt.Sprint(sY) + " Q " + fmt.Sprint(cX) + " " + fmt.Sprint(cY) + " " + fmt.Sprint(eX) + " " + fmt.Sprint(eY) + " "
		} else {
			/* The excruciating difficulty of drawing a straight line */
			path += "L " + fmt.Sprint(p.X) + " " + fmt.Sprint(p.Y) + " "
		}

		prevP = p
	}

	if this.IsClosed() {
		path += `Z`
	}

	// FIXME(akavel): id = SVGPath_id()

	/* Add markers if necessary. */
	if startPoint.Flags&Point_SMARKER != 0 {
		this.options["marker-start"] = "url(#Pointer)"
	} else if startPoint.Flags&Point_IMARKER != 0 {
		this.options["marker-start"] = "url(#iPointer)"
	}

	if endPoint.Flags&Point_SMARKER != 0 {
		this.options["marker-end"] = "url(#Pointer)"
	} else if endPoint.Flags&Point_IMARKER != 0 {
		this.options["marker-end"] = "url(#iPointer)"
	}

	/*
	 * SVG objects without a fill will be transparent, and this looks so
	 * terrible with the drop-shadow effect. Any objects that aren't filled
	 * automatically get a white fill.
	 */
	if this.IsClosed() && `` == (this.options[`fill`]) {
		this.options[`fill`] = `#fff`
	}

	out += "\t<path id=\"path" + this.name + "\" "
	for opt, val := range this.options {
		if strpos(opt, `a2s:`, 0) == 0 {
			continue
		}
		out += opt + "=\"" + val + "\" "
	}
	out += "d=\"" + path + "\" />\n"

	if len(this.text) > 0 {
		for _, text := range this.text {
			text.SetID(this.name)
			out += "\t" + text.Render() + "\n"
		}
	}

	bound = len(this.ticks)
	for i := 0; i < bound; i++ {
		t := this.ticks[i]
		if t.Flags&Point_DOT != 0 {
			out += "<circle cx=\"" + fmt.Sprint(t.X) + "\" cy=\"" + fmt.Sprint(t.Y) + "\" r=\"3\" fill=\"black\" />"
		} else if t.Flags&Point_TICK != 0 {
			x1 := t.X - 4
			y1 := t.Y - 4
			x2 := t.X + 4
			y2 := t.Y + 4
			out += fmt.Sprintf("<line x1=\"%v\" y1=\"%v\" x2=\"%v\" y2=\"%v\" stroke-width=\"1\" />", x1, y1, x2, y2)

			x1 = t.X + 4
			y1 = t.Y - 4
			x2 = t.X - 4
			y2 = t.Y + 4
			out += fmt.Sprintf("<line x1=\"%v\" y1=\"%v\" x2=\"%v\" y2=\"%v\" stroke-width=\"1\" />", x1, y1, x2, y2)
		}
	}

	out += "</g>\n"
	return out
}

var _SVGText_id = 0

func SVGText_id() string {
	s := fmt.Sprint(_SVGText_id)
	_SVGText_id++
	return s
}

/*
 * Nothing really special here. Container for representing text bits.
 */
type SVGText struct {
	options map[string]string
	string_ string
	point   Point
	name    string
}

func NewSVGText(X, Y Coord) *SVGText {
	this := &SVGText{}
	this.point = NewPoint(X, Y)
	this.name = SVGText_id()
	this.options = map[string]string{}
	return this
}

func (this *SVGText) SetOption(opt, val string) {
	this.options[opt] = val
}

func (this *SVGText) SetOptions(opt map[string]string) {
	for k, v := range opt {
		this.options[k] = v
	}
}

func (this *SVGText) SetID(id string) {
	this.name = str_replace(` `, `_`, str_replace(`"`, `_`, id))
}

func (this *SVGText) GetID() string {
	return this.name
}

func (this *SVGText) GetPoint() Point {
	return this.point
}

func (this *SVGText) SetString(string_ string) {
	this.string_ = string_
}

func (this *SVGText) Render() string {
	out := "<text x=\"" + fmt.Sprint(this.point.X) + "\" y=\"" + fmt.Sprint(this.point.Y) + "\" id=\"text" + this.name + "\" "
	for opt, val := range this.options {
		if strpos(opt, `a2s:`, 0) == 0 {
			continue
		}
		out += opt + "=\"" + val + "\" "
	}
	out += ">"
	// FIXME(akavel): out += htmlentities(this.string_)
	out += html.EscapeString(this.string_)
	out += "</text>\n"
	return out
}

/*
 * Main class for parsing ASCII and constructing the SVG output based on the
 * above classes.
 */
type ASCIIToSVG struct {
	DisableBlurDropShadow bool
	FontFamily            string

	rawData []byte
	grid    [][]rune

	svgObjects   *SVGGroup
	clearCorners [][2]int

	commands map[string]map[string]string
}

/* Directions for traversing lines in our grid */
type ASCIIToSVG_DIR int

const ASCIIToSVG_DIR_UNDEFINED ASCIIToSVG_DIR = 0
const ASCIIToSVG_DIR_UP ASCIIToSVG_DIR = 0x1
const ASCIIToSVG_DIR_DOWN ASCIIToSVG_DIR = 0x2
const ASCIIToSVG_DIR_LEFT ASCIIToSVG_DIR = 0x4
const ASCIIToSVG_DIR_RIGHT ASCIIToSVG_DIR = 0x8
const ASCIIToSVG_DIR_NE ASCIIToSVG_DIR = 0x10
const ASCIIToSVG_DIR_SE ASCIIToSVG_DIR = 0x20

func NewASCIIToSVG(data []byte) *ASCIIToSVG {
	this := &ASCIIToSVG{}
	/* For debugging purposes */
	this.rawData = data
	this.FontFamily = "Consolas,Monaco,Anonymous Pro,Anonymous,Bitstream Sans Mono,monospace"

	// FIXME(akavel): CustomObjects_LoadObjects()

	this.clearCorners = [][2]int{}

	/*
	 * Parse out any command references. These need to be at the bottom of the
	 * diagram due to the way they're removed. Format is:
	 * [identifier] optional-colon optional-spaces ({json-blob})\n
	 *
	 * The JSON blob may not contain objects as values or the regex will break.
	 */
	this.commands = map[string]map[string]string{}
	matches := regexp.MustCompile(`(?ms)`+`^\[`+`([^\]]+)`+`\]`+`:?`+`\s+`+`({[^}]+?})`).FindAllSubmatch(data, -1)
	for _, match := range matches {
		this.commands[string(match[1])] = json_decode(match[2], true)
	}
	data = regexp.MustCompile(`(?ms)`+`^\[`+`([^\]]+)`+`\]`+`(:?)`+`\s+`+`.*`).ReplaceAll(data, nil)

	/*
	 * Treat our ASCII field as a grid and store each character as a point in
	 * that grid. The (0, 0) coordinate on this grid is top-left, just as it
	 * is in images.
	 */
	for _, line := range bytes.Split(data, []byte("\n")) {
		this.grid = append(this.grid, bytes.Runes(line))
	}

	this.svgObjects = NewSVGGroup()
	return this
}

/*
 * This is kind of a stupid and hacky way to do this, but this allows setting
 * the default scale of one grid space on the X and Y axes.
 */
func (this *ASCIIToSVG) SetDimensionScale(X, Y Coord) {
	o := Scale_GetInstance()
	o.SetScale(X, Y)
}

// func (this *ASCIIToSVG) Dump() {
// 	var_export(this)
// }

/* Render out what we've done!  */
func (this *ASCIIToSVG) Render() string {
	o := Scale_GetInstance()

	/* Figure out how wide we need to make the canvas */
	canvasWidth := Coord(0)
	for _, line := range this.grid {
		if Coord(len(line)) > canvasWidth {
			canvasWidth = Coord(len(line))
		}
	}

	canvasWidth = Coord(canvasWidth)*o.XScale + 10
	canvasHeight := Coord(len(this.grid)) * o.YScale

	/*
	 * Boilerplate header with definitions that we might be using for markers
	 * and drop shadows.
	 */
	out := `<?xml version="1.0" standalone="no"?>
<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN" 
  "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd">
<!-- Created with ASCIIToSVG (http://9vx.org/~dho/a2s/) -->
<svg width="` + fmt.Sprint(canvasWidth) + `px" height="` + fmt.Sprint(canvasHeight) + `px" version="1.1"
  xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">
  <defs>
    <filter id="dsFilterNoBlur" width="150%" height="150%">
      <feOffset result="offOut" in="SourceGraphic" dx="3" dy="3"/>
      <feColorMatrix result="matrixOut" in="offOut" type="matrix" values="0.2 0 0 0 0 0 0.2 0 0 0 0 0 0.2 0 0 0 0 0 1 0"/>
      <feBlend in="SourceGraphic" in2="matrixOut" mode="normal"/>
    </filter>
    <filter id="dsFilter" width="150%" height="150%">
      <feOffset result="offOut" in="SourceGraphic" dx="3" dy="3"/>
      <feColorMatrix result="matrixOut" in="offOut" type="matrix" values="0.2 0 0 0 0 0 0.2 0 0 0 0 0 0.2 0 0 0 0 0 1 0"/>
      <feGaussianBlur result="blurOut" in="matrixOut" stdDeviation="3"/>
      <feBlend in="SourceGraphic" in2="blurOut" mode="normal"/>
    </filter>
    <marker id="iPointer"
      viewBox="0 0 10 10" refX="5" refY="5" 
      markerUnits="strokeWidth"
      markerWidth="8" markerHeight="7"
      orient="auto">
      <path d="M 10 0 L 10 10 L 0 5 z" />
    </marker>
    <marker id="Pointer"
      viewBox="0 0 10 10" refX="5" refY="5" 
      markerUnits="strokeWidth"
      markerWidth="8" markerHeight="7"
      orient="auto">
      <path d="M 0 0 L 10 5 L 0 10 z" />
    </marker>
  </defs>
`

	/* Render the group, everything lives in there */
	out += this.svgObjects.Render()

	out += "</svg>\n"

	return out
}

/*
 * Parsing the grid is a multi-step process. We parse out boxes first, as
 * this makes it easier to then parse lines. By parse out, I do mean we
 * parse them and then remove them. This does mean that a complete line
 * will not travel along the edge of a box, but you probably won't notice
 * unless the box is curved anyway. While edges are removed, points are
 * not. This means that you can cleanly allow lines to intersect boxes
 * (as long as they do not bisect!
 *
 * After parsing boxes and lines, we remove the corners from the grid. At
 * this point, all we have left should be text, which we can pick up and
 * place.
 */
func (this *ASCIIToSVG) ParseGrid() {
	this.parseBoxes()
	this.parseLines()

	for _, corner := range this.clearCorners {
		this.grid[corner[0]][corner[1]] = ' '
	}

	this.parseText()

	this.injectCommands()
}

/*
 * Ahh, good ol' box parsing. We do this by scanning each row for points and
 * attempting to close the shape. Since the approach is first horizontal,
 * then vertical, we complete the shape in a clockwise order (which is
 * important for the Bezier curve generation.
 */
func (this *ASCIIToSVG) parseBoxes() {
	/* Set up our box group  */
	this.svgObjects.PushGroup(`boxes`)
	this.svgObjects.SetOption(`stroke`, `black`)
	this.svgObjects.SetOption(`stroke-width`, `2`)
	this.svgObjects.SetOption(`fill`, `none`)

	/* Scan the grid for corners */
	for row, line := range this.grid {
		for col, char := range line {
			if this.isCorner(char) {
				path := NewSVGPath()

				if char == '.' || char == '\'' {
					path.AddPoint(Coord(col), Coord(row), Point_CONTROL)
				} else {
					path.AddPoint(Coord(col), Coord(row), Point_POINT)
				}

				/*
				 * The wall follower is a left-turning, marking follower. See that
				 * function for more information on how it works.
				 */
				this.wallFollow(path, row, col+1, ASCIIToSVG_DIR_RIGHT, nil, 0)

				/* We only care about closed polygons */
				if path.IsClosed() {
					path.OrderPoints()

					skip := false
					/*
					 * The walking code can find the same box from a different edge:
					 *
					 * +---+   +---+
					 * |   |   |   |
					 * |   +---+   |
					 * +-----------+
					 *
					 * so ignore adding a box that we've already added.
					 */
					for _, box := range this.svgObjects.GetGroup(`boxes`) {
						bP := box.(*SVGPath).GetPoints()
						pP := path.GetPoints()
						pPoints := len(pP)
						shared := 0

						/*
						 * If the boxes don't have the same number of edges, they
						 * obviously cannot be the same box.
						 */
						if len(bP) != pPoints {
							continue
						}

						/* Traverse the vertices of this Newbox... */
						for i := 0; i < pPoints; i++ {
							/* ...and find them in this existing box. */
							for j := 0; j < pPoints; j++ {
								if pP[i].X == bP[j].X && pP[i].Y == bP[j].Y {
									shared++
								}
							}
						}

						/* If all the edges are in common, it's the same shape. */
						if shared == len(bP) {
							skip = true
							break
						}
					}

					if skip == false {
						/* Search for any references for styling this polygon; add it */
						if this.DisableBlurDropShadow {
							path.SetOption(`filter`, `url(#dsFilterNoBlur)`)
						} else {
							path.SetOption(`filter`, `url(#dsFilter)`)
						}

						name := this.findCommands(path)

						if name != `` {
							path.SetID(name)
						}

						this.svgObjects.AddObject(path)
					}
				}
			}
		}
	}

	/*
	 * Once we've found all the boxes, we want to remove them from the grid so
	 * that they don't confuse the line parser. However, we don't remove any
	 * corner characters because these might be shared by lines.
	 */
	for _, box := range this.svgObjects.GetGroup(`boxes`) {
		this.clearObject(box.(*SVGPath))
	}

	/* Anything after this is not a subgroup */
	this.svgObjects.PopGroup()
}

/*
 * Our line parser operates differently than the polygon parser. This is
 * because lines are not intrinsically marked with starting points (markers
 * are optional) -- they just sort of begin. Additionally, so that markers
 * will work, we can't just construct a line from some random point: we need
 * to start at the correct edge.
 *
 * Thus, the line parser traverses vertically first, then horizontally. Once
 * a line is found, it is cleared immediately (but leaving any control points
 * in case there were any intersections.
 */
func (this *ASCIIToSVG) parseLines() {
	/* Set standard line options */
	this.svgObjects.PushGroup(`lines`)
	this.svgObjects.SetOption(`stroke`, `black`)
	this.svgObjects.SetOption(`stroke-width`, `2`)
	this.svgObjects.SetOption(`fill`, `none`)

	/* The grid is not uniform, so we need to determine the longest row. */
	maxCols := 0
	bound := len(this.grid)
	for r := 0; r < bound; r++ {
		maxCols = max(maxCols, len(this.grid[r]))
	}

	for c := 0; c < maxCols; c++ {
		for r := 0; r < bound; r++ {
			/* This gets set if we find a line-start here. */
			dir := ASCIIToSVG_DIR_UNDEFINED

			line := NewSVGPath()

			/*
			 * Since the column count isn't uniform, don't attempt to handle any
			 * rows that don't extend out this far.
			 */
			if r >= len(this.grid) || c >= len(this.grid[r]) {
				continue
			}

			char := this.getChar(r, c)
			switch char {
			/*
			 * Do marker characters first. These are the easiest because they are
			 * basically guaranteed to represent the start of the line.
			 */
			case '<':
				e := this.getChar(r, c+1)
				if this.isEdge(e, ASCIIToSVG_DIR_RIGHT) || this.isCorner(e) {
					line.AddMarker(Coord(c), Coord(r), Point_IMARKER)
					dir = ASCIIToSVG_DIR_RIGHT
				} else {
					se := this.getChar(r+1, c+1)
					ne := this.getChar(r-1, c+1)
					if se == '\\' {
						line.AddMarker(Coord(c), Coord(r), Point_IMARKER)
						dir = ASCIIToSVG_DIR_SE
					} else if ne == '/' {
						line.AddMarker(Coord(c), Coord(r), Point_IMARKER)
						dir = ASCIIToSVG_DIR_NE
					}
				}
				break
			case '^':
				s := this.getChar(r+1, c)
				if this.isEdge(s, ASCIIToSVG_DIR_DOWN) || this.isCorner(s) {
					line.AddMarker(Coord(c), Coord(r), Point_IMARKER)
					dir = ASCIIToSVG_DIR_DOWN
				} else if this.getChar(r+1, c+1) == '\\' {
					/* Don't need to check west for diagonals. */
					line.AddMarker(Coord(c), Coord(r), Point_IMARKER)
					dir = ASCIIToSVG_DIR_SE
				}
				break
			case '>':
				w := this.getChar(r, c-1)
				if this.isEdge(w, ASCIIToSVG_DIR_LEFT) || this.isCorner(w) {
					line.AddMarker(Coord(c), Coord(r), Point_IMARKER)
					dir = ASCIIToSVG_DIR_LEFT
				}
				/* All diagonals come from west, so we don't need to check */
				break
			case 'v':
				n := this.getChar(r-1, c)
				if this.isEdge(n, ASCIIToSVG_DIR_UP) || this.isCorner(n) {
					line.AddMarker(Coord(c), Coord(r), Point_IMARKER)
					dir = ASCIIToSVG_DIR_UP
				} else if this.getChar(r-1, c+1) == '/' {
					line.AddMarker(Coord(c), Coord(r), Point_IMARKER)
					dir = ASCIIToSVG_DIR_NE
				}
				break

			/*
			 * Edges are handled specially. We have to look at the context of the
			 * edge to determine whether it's the start of a line. A vertical edge
			 * can appear as the start of a line in the following circumstances:
			 *
			 * +-------------      +--------------     +----    | (s)
			 * |                   |                   |        |
			 * |      | (s)        +-------+           |(s)     |
			 * +------+                    | (s)
			 *
			 * From this we can extrapolate that we are a starting edge if our
			 * southern neighbor is a vertical edge or corner, but we have no line
			 * material to our north (and vice versa). This logic does allow for
			 * the southern / northern neighbor to be part of a separate
			 * horizontal line.
			 */
			case ':':
				line.SetOption(`stroke-dasharray`, `5 5`)
				fallthrough
			case '|':
				n := this.getChar(r-1, c)
				s := this.getChar(r+1, c)
				if (s == '|' || s == ':' || this.isCorner(s)) &&
					n != '|' && n != ':' && !this.isCorner(n) &&
					n != '^' {
					dir = ASCIIToSVG_DIR_DOWN
				} else if (n == '|' || n == ':' || this.isCorner(n)) &&
					s != '|' && s != ':' && !this.isCorner(s) &&
					s != 'v' {
					dir = ASCIIToSVG_DIR_UP
				}
				break

			/*
			 * Horizontal edges have the same properties for search as vertical
			 * edges, except we need to look east / west. The diagrams for the
			 * vertical case are still accurate to visualize this case; just
			 * mentally turn them 90 degrees clockwise.
			 */
			case '=':
				line.SetOption(`stroke-dasharray`, `5 5`)
				fallthrough
			case '-':
				w := this.getChar(r, c-1)
				e := this.getChar(r, c+1)
				if (w == '-' || w == '=' || this.isCorner(w)) &&
					e != '=' && e != '-' && !this.isCorner(e) &&
					e != '>' {
					dir = ASCIIToSVG_DIR_LEFT
				} else if (e == '-' || e == '=' || this.isCorner(e)) &&
					w != '=' && w != '-' && !this.isCorner(w) &&
					w != '<' {
					dir = ASCIIToSVG_DIR_RIGHT
				}
				break

			/*
			 * We can only find diagonals going north or south and east. This is
			 * simplified due to the fact that they have no corners. We are
			 * guaranteed to run into their westernmost point or their relevant
			 * marker.
			 */
			case '/':
				ne := this.getChar(r-1, c+1)
				if ne == '/' || ne == '^' || ne == '>' {
					dir = ASCIIToSVG_DIR_NE
				}
				break

			case '\\':
				se := this.getChar(r+1, c+1)
				if se == '\\' || se == 'v' || se == '>' {
					dir = ASCIIToSVG_DIR_SE
				}
				break

			/*
			 * The corner case must consider all four directions. Though a
			 * reasonable person wouldn't use slant corners for this, they are
			 * considered corners, so it kind of makes sense to handle them the
			 * same way. For this case, envision the starting point being a corner
			 * character in both the horizontal and vertical case. And then
			 * mentally overlay them and consider that :).
			 */
			case '+', '#':
				ne := this.getChar(r-1, c+1)
				se := this.getChar(r+1, c+1)
				if ne == '/' || ne == '^' || ne == '>' {
					dir = ASCIIToSVG_DIR_NE
				} else if se == '\\' || se == 'v' || se == '>' {
					dir = ASCIIToSVG_DIR_SE
				}
				fallthrough

			case '.', '\'':
				n := this.getChar(r-1, c)
				w := this.getChar(r, c-1)
				s := this.getChar(r+1, c)
				e := this.getChar(r, c+1)
				if (w == '=' || w == '-') && n != '|' && n != ':' && w != '-' &&
					e != '=' && e != '|' && s != ':' {
					dir = ASCIIToSVG_DIR_LEFT
				} else if (e == '=' || e == '-') && n != '|' && n != ':' &&
					w != '-' && w != '=' && s != '|' && s != ':' {
					dir = ASCIIToSVG_DIR_RIGHT
				} else if (s == '|' || s == ':') && n != '|' && n != ':' &&
					w != '-' && w != '=' && e != '-' && e != '=' &&
					((char != '.' && char != '\'') ||
						(char == '.' && s != '.') ||
						(char == '\'' && s != '\'')) {
					dir = ASCIIToSVG_DIR_DOWN
				} else if (n == '|' || n == ':') && s != '|' && s != ':' &&
					w != '-' && w != '=' && e != '-' && e != '=' &&
					((char != '.' && char != '\'') ||
						(char == '.' && s != '.') ||
						(char == '\'' && s != '\'')) {
					dir = ASCIIToSVG_DIR_UP
				}
				break
			}

			/* It does actually save lines! */
			if dir != ASCIIToSVG_DIR_UNDEFINED {
				rInc := 0
				cInc := 0
				if !this.isMarker(char) {
					line.AddPoint(Coord(c), Coord(r), Point_POINT)
				}

				/*
				 * The walk routine may attempt to add the point again, so skip it.
				 * If we don't, we can miss the line or end up with just a point.
				 */
				if dir == ASCIIToSVG_DIR_UP {
					rInc = -1
					cInc = 0
				} else if dir == ASCIIToSVG_DIR_DOWN {
					rInc = 1
					cInc = 0
				} else if dir == ASCIIToSVG_DIR_RIGHT {
					rInc = 0
					cInc = 1
				} else if dir == ASCIIToSVG_DIR_LEFT {
					rInc = 0
					cInc = -1
				} else if dir == ASCIIToSVG_DIR_NE {
					rInc = -1
					cInc = 1
				} else if dir == ASCIIToSVG_DIR_SE {
					rInc = 1
					cInc = 1
				}

				/*
				 * Walk the points of this line. Note we don't use wallFollow; we are
				 * operating under the assumption that lines do not meander. (And, in
				 * any event, that algorithm is intended to find a closed object.)
				 */
				this.walk(line, r+rInc, c+cInc, dir, 0)

				/*
				 * Remove it so that we don't confuse any other lines. This leaves
				 * corners in tact, still.
				 */
				this.clearObject(line)
				this.svgObjects.AddObject(line)

				/* We may be able to find more lines starting from this same point */
				if this.isCorner(char) {
					r--
				}
			}
		}
	}

	this.svgObjects.PopGroup()
}

/*
 * Look for text in a file. If the text appears in a box that has a dark
 * fill, we want to give it a light fill (and vice versa). This means we
 * have to figure out what box it lives in (if any) and do all sorts of
 * color calculation magic.
 */
func (this *ASCIIToSVG) parseText() {
	o := Scale_GetInstance()

	/*
	 * The style options deserve some comments. The monospace and font-size
	 * choices are not accidental. This gives the best sort of estimation
	 * for font size to scale that I could come up with empirically.
	 *
	 * N.B. This might change with different scales. I kind of feel like this
	 * is a bug waiting to be filed, but whatever.
	 */
	fSize := 0.95 * o.YScale
	this.svgObjects.PushGroup(`text`)
	this.svgObjects.SetOption(`fill`, `black`)
	this.svgObjects.SetOption(`style`,
		"font-family:"+this.FontFamily+";font-size:"+fmt.Sprint(fSize)+"px")

	/*
	 * Text gets the same scanning treatment as boxes. We do left-to-right
	 * scanning, which should probably be configurable in case someone wants
	 * to use this with e.g. Arabic or some other right-to-left language.
	 * Either way, this isn't UTF-8 safe (thanks, PHP!!!), so that'll require
	 * thought regardless.
	 */
	boxes := this.svgObjects.GetGroup(`boxes`)
	bound := len(boxes)

	for row, line := range this.grid {
		cols := len(line)
		for i := 0; i < cols; i++ {
			if this.getChar(row, i) != ' ' {
				/* More magic numbers that probably need research. */
				t := NewSVGText(Coord(i)-.6, Coord(row)+0.3)

				/* Time to figure out which (if any) box we live inside */
				tP := t.GetPoint()

				maxPoint := NewPoint(-1, -1)
				boxQueue := []*SVGPath{}

				for j := 0; j < bound; j++ {
					if boxes[j].(*SVGPath).HasPoint(Coord(tP.GridX), Coord(tP.GridY)) {
						boxPoints := boxes[j].(*SVGPath).GetPoints()
						boxTL := boxPoints[0]

						/*
						 * This text is in this box, but it may still be in a more
						 * specific nested box. Find the box with the highest top
						 * left X,Y coordinate. Keep a queue of boxes in case the top
						 * most box doesn't have a fill.
						 */
						if boxTL.Y > maxPoint.Y && boxTL.X > maxPoint.X {
							maxPoint.X = boxTL.X
							maxPoint.Y = boxTL.Y
							boxQueue = append(boxQueue, boxes[j].(*SVGPath))
						}
					}
				}

				if len(boxQueue) > 0 {
					/*
					 * Work backwards through the boxes to find the box with the most
					 * specific fill.
					 */
					j := len(boxQueue) - 1
					for ; j >= 0; j-- {
						fill := boxQueue[j].GetOption(`fill`)

						if fill == `none` || fill == `` {
							continue
						}

						if fill[0] != '#' {
							if A2S_colors[strtolower(fill)] == `` {
								continue
							} else {
								fill = A2S_colors[strtolower(fill)]
							}
						} else {
							if len(fill) != 4 && len(fill) != 7 {
								continue
							}
						}

						// FIXME(akavel): if fill {
						if fill != `` {
							/* Attempt to parse the fill color */
							var cR, cG, cB int
							if len(fill) == 4 {
								cR = hexdec(str_repeat(fill[1:2], 2))
								cG = hexdec(str_repeat(fill[2:3], 2))
								cB = hexdec(str_repeat(fill[3:4], 2))
							} else if len(fill) == 7 {
								cR = hexdec(fill[1:3])
								cG = hexdec(fill[3:5])
								cB = hexdec(fill[5:7])
							}

							/*
							 * This magic is gleaned from the working group paper on
							 * accessibility at http://www.w3.org/TR/AERT. The recommended
							 * contrast is a brightness difference of at least 125 and a
							 * color difference of at least 500. Since our default color
							 * is black, that makes the color difference easier.
							 */
							bFill := ((cR * 299) + (cG * 587) + (cB * 114)) / 1000
							bDiff := cR + cG + cB
							bText := 0

							if bFill-bText < 125 || bDiff < 500 {
								/* If black is too dark, white will work */
								t.SetOption(`fill`, `#fff`)
							} else {
								t.SetOption(`fill`, `#000`)
							}

							break
						}
					}

					if j < 0 {
						t.SetOption(`fill`, `#000`)
					}
				} else {
					/* This text isn't inside a box; make it black */
					t.SetOption(`fill`, `#000`)
				}

				/* We found a stringy character, eat it and the rest. */
				str := string(this.getChar(row, i))
				i++
				for i < len(line) && this.getChar(row, i) != ' ' {
					str += string(this.getChar(row, i))
					i++
					/* Eat up to 1 space */
					if this.getChar(row, i) == ' ' {
						str += " "
						i++
					}
				}

				if str == `` {
					continue
				}

				t.SetString(str)

				/*
				 * If we were in a box, group with the box. Otherwise it gets its
				 * own group.
				 */
				if len(boxQueue) > 0 {
					t.SetOption(`stroke`, `none`)
					t.SetOption(`style`,
						"font-family:"+this.FontFamily+";font-size:"+fmt.Sprint(fSize)+"px")
					boxQueue[len(boxQueue)-1].AddText(t)
				} else {
					this.svgObjects.AddObject(t)
				}
			}
		}
	}
}

/*
 * Allow specifying references that target an object starting at grid point
 * (ROW,COL). This allows styling of lines, boxes, or any text object.
 */
func (this *ASCIIToSVG) injectCommands() {
	boxes := this.svgObjects.GetGroup(`boxes`)
	lines := this.svgObjects.GetGroup(`lines`)
	text := this.svgObjects.GetGroup(`text`)

	for _, obj := range boxes {
		objPoints := obj.(*SVGPath).GetPoints()
		pointCmd := "" + fmt.Sprint(objPoints[0].GridY) + "," + fmt.Sprint(objPoints[0].GridX) + ""

		if (this.commands[pointCmd]) != nil {
			obj.(*SVGPath).SetOptions(this.commands[pointCmd])
		}

		for _, text := range obj.(*SVGPath).GetText() {
			textPoint := text.GetPoint()
			pointCmd = "" + fmt.Sprint(textPoint.GridY) + "," + fmt.Sprint(textPoint.GridX) + ""

			if (this.commands[pointCmd]) != nil {
				text.SetOptions(this.commands[pointCmd])
			}
		}
	}

	for _, obj := range lines {
		objPoints := obj.(*SVGPath).GetPoints()
		pointCmd := "" + fmt.Sprint(objPoints[0].GridY) + "," + fmt.Sprint(objPoints[0].GridX) + ""

		if (this.commands[pointCmd]) != nil {
			obj.(*SVGPath).SetOptions(this.commands[pointCmd])
		}
	}

	for _, obj := range text {
		objPoint := obj.(*SVGText).GetPoint()
		pointCmd := "" + fmt.Sprint(objPoint.GridY) + "," + fmt.Sprint(objPoint.GridX) + ""

		if (this.commands[pointCmd]) != nil {
			obj.(*SVGText).SetOptions(this.commands[pointCmd])
		}
	}
}

/*
 * A generic, recursive line walker. This walker makes the assumption that
 * lines want to go in the direction that they are already heading. I'm
 * sure that there are ways to formulate lines to screw this walker up,
 * but it does a good enough job right now.
 */
// FIXME(akavel): func (this *ASCIIToSVG) walk(path, row, col, dir, d = 0) {
func (this *ASCIIToSVG) walk(path *SVGPath, row, col int, dir ASCIIToSVG_DIR, d int) {
	d++
	r := row
	c := col
	var cInc, rInc int

	if dir == ASCIIToSVG_DIR_RIGHT || dir == ASCIIToSVG_DIR_LEFT {
		cInc = -1
		if dir == ASCIIToSVG_DIR_RIGHT {
			cInc = 1
		}
		rInc = 0
	} else if dir == ASCIIToSVG_DIR_DOWN || dir == ASCIIToSVG_DIR_UP {
		cInc = 0
		rInc = -1
		if dir == ASCIIToSVG_DIR_DOWN {
			rInc = 1
		}
	} else if dir == ASCIIToSVG_DIR_SE || dir == ASCIIToSVG_DIR_NE {
		cInc = 1
		rInc = -1
		if dir == ASCIIToSVG_DIR_SE {
			rInc = 1
		}
	}

	/* Follow the edge for as long as we can */
	cur := this.getChar(r, c)
	for this.isEdge(cur, dir) {
		if cur == ':' || cur == '=' {
			path.SetOption(`stroke-dasharray`, `5 5`)
		}

		if this.isTick(cur) {
			if cur == 'o' {
				path.AddTick(Coord(c), Coord(r), Point_DOT)
			} else {
				path.AddTick(Coord(c), Coord(r), Point_TICK)
			}
			path.AddPoint(Coord(c), Coord(r), Point_POINT)
		}

		c += cInc
		r += rInc
		cur = this.getChar(r, c)
	}

	if this.isCorner(cur) {
		if cur == '.' || cur == '\'' {
			path.AddPoint(Coord(c), Coord(r), Point_CONTROL)
		} else {
			path.AddPoint(Coord(c), Coord(r), Point_POINT)
		}

		if path.IsClosed() {
			path.PopPoint()
			return
		}

		/*
		 * Attempt first to continue in the current direction. If we can't,
		 * try to go in any direction other than the one opposite of where
		 * we just came from -- no backtracking.
		 */
		n := this.getChar(r-1, c)
		s := this.getChar(r+1, c)
		e := this.getChar(r, c+1)
		w := this.getChar(r, c-1)
		next := this.getChar(r+rInc, c+cInc)

		se := this.getChar(r+1, c+1)
		ne := this.getChar(r-1, c+1)

		if this.isCorner(next) || this.isEdge(next, dir) {
			this.walk(path, r+rInc, c+cInc, dir, d)
			return
		} else if dir != ASCIIToSVG_DIR_DOWN &&
			(this.isCorner(n) || this.isEdge(n, ASCIIToSVG_DIR_UP)) {
			/* Can't turn up into bottom corner */
			if (cur != '.' && cur != '\'') || (cur == '.' && n != '.') ||
				(cur == '\'' && n != '\'') {
				this.walk(path, r-1, c, ASCIIToSVG_DIR_UP, d)
				return
			}
		} else if dir != ASCIIToSVG_DIR_UP &&
			(this.isCorner(s) || this.isEdge(s, ASCIIToSVG_DIR_DOWN)) {
			/* Can't turn down into top corner */
			if (cur != '.' && cur != '\'') || (cur == '.' && s != '.') ||
				(cur == '\'' && s != '\'') {
				this.walk(path, r+1, c, ASCIIToSVG_DIR_DOWN, d)
				return
			}
		} else if dir != ASCIIToSVG_DIR_LEFT &&
			(this.isCorner(e) || this.isEdge(e, ASCIIToSVG_DIR_RIGHT)) {
			this.walk(path, r, c+1, ASCIIToSVG_DIR_RIGHT, d)
			return
		} else if dir != ASCIIToSVG_DIR_RIGHT &&
			(this.isCorner(w) || this.isEdge(w, ASCIIToSVG_DIR_LEFT)) {
			this.walk(path, r, c-1, ASCIIToSVG_DIR_LEFT, d)
			return
		} else if dir == ASCIIToSVG_DIR_SE &&
			(this.isCorner(ne) || this.isEdge(ne, ASCIIToSVG_DIR_NE)) {
			this.walk(path, r-1, c+1, ASCIIToSVG_DIR_NE, d)
			return
		} else if dir == ASCIIToSVG_DIR_NE &&
			(this.isCorner(se) || this.isEdge(se, ASCIIToSVG_DIR_SE)) {
			this.walk(path, r+1, c+1, ASCIIToSVG_DIR_SE, d)
			return
		}
	} else if this.isMarker(cur) {
		/* We found a marker! Add it. */
		path.AddMarker(Coord(c), Coord(r), Point_SMARKER)
		return
	} else {
		/*
		 * Not a corner, not a marker, and we already ate edges. Whatever this
		 * is, it is not part of the line.
		 */
		path.AddPoint(Coord(c-cInc), Coord(r-rInc), Point_POINT)
		return
	}
}

/*
 * This function attempts to follow a line and complete it into a closed
 * polygon. It assumes that we have been called from a top point, and in any
 * case that the polygon can be found by moving clockwise along its edges.
 * Any time this algorithm finds a corner, it attempts to turn right. If it
 * cannot turn right, it goes in any direction other than the one it came
 * from. If it cannot complete the polygon by continuing in any direction
 * from a point, that point is removed from the path, and we continue on
 * from the previous point (since this is a recursive function).
 *
 * Because the function assumes that it is starting from the top left,
 * if its first turn cannot be a right turn to moving down, the object
 * cannot be a valid polygon. It also maintains an internal list of points
 * it has already visited, and refuses to visit any point twice.
 */
// FIXME(akavel): func (this *ASCIIToSVG) wallFollow(path, r, c, dir, bucket = array(), d = 0) {
func (this *ASCIIToSVG) wallFollow(path *SVGPath, r, c int, dir ASCIIToSVG_DIR, bucket map[string]ASCIIToSVG_DIR, d int) {
	if bucket == nil {
		bucket = map[string]ASCIIToSVG_DIR{}
	}
	d++

	var cInc, rInc int
	if dir == ASCIIToSVG_DIR_RIGHT || dir == ASCIIToSVG_DIR_LEFT {
		cInc = -1
		if dir == ASCIIToSVG_DIR_RIGHT {
			cInc = 1
		}
		rInc = 0
	} else if dir == ASCIIToSVG_DIR_DOWN || dir == ASCIIToSVG_DIR_UP {
		cInc = 0
		rInc = -1
		if dir == ASCIIToSVG_DIR_DOWN {
			rInc = 1
		}
	}

	/* Traverse the edge in whatever direction we are going. */
	cur := this.getChar(r, c)
	for this.isBoxEdge(cur, dir) {
		r += rInc
		c += cInc
		cur = this.getChar(r, c)
	}

	/* We 'key' our location by catting r and c together */
	key := fmt.Sprint(r) + fmt.Sprint(c)
	if _, set := (bucket[key]); set {
		return
	}

	/*
	 * When we run into a corner, we have to make a somewhat complicated
	 * decision about which direction to turn.
	 */
	if this.isBoxCorner(cur) {
		if _, set := (bucket[key]); !set {
			bucket[key] = 0
		}

		pointExists := false
		switch cur {
		case '.', '\'':
			pointExists = path.AddPoint(Coord(c), Coord(r), Point_CONTROL)
			break

		case '#':
			pointExists = path.AddPoint(Coord(c), Coord(r), Point_POINT)
			break
		}

		if path.IsClosed() || pointExists {
			return
		}

		/*
		 * Special case: if we're looking for our first turn and we can't make it
		 * due to incompatible corners, keep looking, but don't adjust our call
		 * depth so that we can continue to make progress.
		 */
		if d == 1 && cur == '.' && this.getChar(r+1, c) == '.' {
			this.wallFollow(path, r, c+1, dir, bucket, 0)
			return
		}

		/*
		 * We need to make a decision here on where to turn. We may have multiple
		 * directions we can choose, and all of them might generate a closed
		 * object. Always try turning right first.
		 */
		newDir := ASCIIToSVG_DIR_UNDEFINED
		n := this.getChar(r-1, c)
		s := this.getChar(r+1, c)
		e := this.getChar(r, c+1)
		w := this.getChar(r, c-1)

		if dir == ASCIIToSVG_DIR_RIGHT {
			if 0 == (bucket[key]&ASCIIToSVG_DIR_DOWN) &&
				(this.isBoxEdge(s, ASCIIToSVG_DIR_DOWN) || this.isBoxCorner(s)) {
				/* We can't turn into another top edge. */
				if (cur != '.' && cur != '\'') || (cur == '.' && s != '.') ||
					(cur == '\'' && s != '\'') {
					newDir = ASCIIToSVG_DIR_DOWN
				}
			} else {
				/* There is no right hand turn for us; this isn't a valid start */
				if d == 1 {
					return
				}
			}
		} else if dir == ASCIIToSVG_DIR_DOWN {
			if 0 == (bucket[key]&ASCIIToSVG_DIR_LEFT) &&
				(this.isBoxEdge(w, ASCIIToSVG_DIR_LEFT) || this.isBoxCorner(w)) {
				// FIXME(akavel): bugfixed below?
				newDir = ASCIIToSVG_DIR_LEFT
			}
		} else if dir == ASCIIToSVG_DIR_LEFT {
			if 0 == (bucket[key]&ASCIIToSVG_DIR_UP) &&
				(this.isBoxEdge(n, ASCIIToSVG_DIR_UP) || this.isBoxCorner(n)) {
				/* We can't turn into another bottom edge. */
				if (cur != '.' && cur != '\'') || (cur == '.' && n != '.') ||
					(cur == '\'' && n != '\'') {
					newDir = ASCIIToSVG_DIR_UP
				}
			}
		} else if dir == ASCIIToSVG_DIR_UP {
			if 0 == (bucket[key]&ASCIIToSVG_DIR_RIGHT) &&
				(this.isBoxEdge(e, ASCIIToSVG_DIR_RIGHT) || this.isBoxCorner(e)) {
				newDir = ASCIIToSVG_DIR_RIGHT
			}
		}

		var cMod, rMod int
		if newDir != ASCIIToSVG_DIR_UNDEFINED {
			if newDir == ASCIIToSVG_DIR_RIGHT || newDir == ASCIIToSVG_DIR_LEFT {
				cMod = -1
				if newDir == ASCIIToSVG_DIR_RIGHT {
					cMod = 1
				}
				rMod = 0
			} else if newDir == ASCIIToSVG_DIR_DOWN || newDir == ASCIIToSVG_DIR_UP {
				cMod = 0
				rMod = -1
				if newDir == ASCIIToSVG_DIR_DOWN {
					rMod = 1
				}
			}

			bucket[key] |= newDir
			this.wallFollow(path, r+rMod, c+cMod, newDir, bucket, d)
			if path.IsClosed() {
				return
			}
		}

		/*
		 * Unfortunately, we couldn't complete the search by turning right,
		 * so we need to pick a different direction. Note that this will also
		 * eventually cause us to continue in the direction we were already
		 * going. We make sure that we don't go in the direction opposite of
		 * the one in which we're already headed, or an any direction we've
		 * already travelled for this point (we may have hit it from an
		 * earlier branch). We accept the first closing polygon as the
		 * "correct" one for this object.
		 */
		if dir != ASCIIToSVG_DIR_RIGHT && 0 == (bucket[key]&ASCIIToSVG_DIR_LEFT) &&
			(this.isBoxEdge(w, ASCIIToSVG_DIR_LEFT) || this.isBoxCorner(w)) {
			bucket[key] |= ASCIIToSVG_DIR_LEFT
			this.wallFollow(path, r, c-1, ASCIIToSVG_DIR_LEFT, bucket, d)
			if path.IsClosed() {
				return
			}
		}
		if dir != ASCIIToSVG_DIR_LEFT && 0 == (bucket[key]&ASCIIToSVG_DIR_RIGHT) &&
			(this.isBoxEdge(e, ASCIIToSVG_DIR_RIGHT) || this.isBoxCorner(e)) {
			bucket[key] |= ASCIIToSVG_DIR_RIGHT
			this.wallFollow(path, r, c+1, ASCIIToSVG_DIR_RIGHT, bucket, d)
			if path.IsClosed() {
				return
			}
		}
		if dir != ASCIIToSVG_DIR_DOWN && 0 == (bucket[key]&ASCIIToSVG_DIR_UP) &&
			(this.isBoxEdge(n, ASCIIToSVG_DIR_UP) || this.isBoxCorner(n)) {
			if (cur != '.' && cur != '\'') || (cur == '.' && n != '.') ||
				(cur == '\'' && n != '\'') {
				/* We can't turn into another bottom edge. */
				bucket[key] |= ASCIIToSVG_DIR_UP
				this.wallFollow(path, r-1, c, ASCIIToSVG_DIR_UP, bucket, d)
				if path.IsClosed() {
					return
				}
			}
		}
		if dir != ASCIIToSVG_DIR_UP && 0 == (bucket[key]&ASCIIToSVG_DIR_DOWN) &&
			(this.isBoxEdge(s, ASCIIToSVG_DIR_DOWN) || this.isBoxCorner(s)) {
			if (cur != '.' && cur != '\'') || (cur == '.' && s != '.') ||
				(cur == '\'' && s != '\'') {
				/* We can't turn into another top edge. */
				bucket[key] |= ASCIIToSVG_DIR_DOWN
				this.wallFollow(path, r+1, c, ASCIIToSVG_DIR_DOWN, bucket, d)
				if path.IsClosed() {
					return
				}
			}
		}

		/*
		 * If we get here, the path doesn't close in any direction from this
		 * point (it's probably a line extension). Get rid of the point from our
		 * path and go back to the last one.
		 */
		path.PopPoint()
		return
	} else if this.isMarker(this.getChar(r, c)) {
		/* Marker is part of a line, not a wall to close. */
		return
	} else {
		/* We landed on some whitespace or something; this isn't a closed path */
		return
	}
}

/*
 * Clears an object from the grid, erasing all edge and marker points. This
 * function retains corners in "clearCorners" to be cleaned up before we do
 * text parsing.
 */
func (this *ASCIIToSVG) clearObject(obj *SVGPath) {
	points := obj.GetPoints()
	closed := obj.IsClosed()

	bound := len(points)
	for i := 0; i < bound; i++ {
		p := points[i]

		var nP *Point
		if i == len(points)-1 {
			/* This keeps us from handling end of line to start of line */
			if closed {
				nP = &points[0]
			} else {
				nP = nil
			}
		} else {
			nP = &points[i+1]
		}

		/* If we're on the same vertical axis as our next point... */
		if nP != nil && p.GridX == nP.GridX {
			/* ...traverse the vertical line from the minimum to maximum points */
			maxY := max(p.GridY, nP.GridY)
			for j := min(p.GridY, nP.GridY); j <= maxY; j++ {
				char := this.getChar(j, p.GridX)

				if !this.isTick(char) && this.isEdge(char, ASCIIToSVG_DIR_UNDEFINED) || this.isMarker(char) {
					this.grid[j][p.GridX] = ' '
				} else if this.isCorner(char) {
					this.clearCorners = append(this.clearCorners, [2]int{j, p.GridX})
				} else if this.isTick(char) {
					this.grid[j][p.GridX] = '+'
				}
			}
		} else if nP != nil && p.GridY == nP.GridY {
			/* Same horizontal plane; traverse from min to max point */
			maxX := max(p.GridX, nP.GridX)
			for j := min(p.GridX, nP.GridX); j <= maxX; j++ {
				char := this.getChar(p.GridY, j)

				if !this.isTick(char) && this.isEdge(char, ASCIIToSVG_DIR_UNDEFINED) || this.isMarker(char) {
					this.grid[p.GridY][j] = ' '
				} else if this.isCorner(char) {
					this.clearCorners = append(this.clearCorners, [2]int{p.GridY, j})
				} else if this.isTick(char) {
					this.grid[p.GridY][j] = '+'
				}
			}
		} else if nP != nil && closed == false && p.GridX != nP.GridX &&
			p.GridY != nP.GridY {
			/*
			 * This is a diagonal line starting from the westernmost point. It
			 * must contain max(p.GridY, nP.GridY) - min(p.GridY, nP.GridY)
			 * segments, and we can tell whether to go north or south depending
			 * on which side of zero p.GridY - nP.GridY lies. There are no
			 * corners in diagonals, so we don't have to keep those around.
			 */
			c := p.GridX
			r := p.GridY
			rInc := 1
			if p.GridY > nP.GridY {
				rInc = -1
			}
			bound := max(p.GridY, nP.GridY) - min(p.GridY, nP.GridY)

			/*
			 * This looks like an off-by-one, but it is not. This clears the
			 * corner, if one exists.
			 */
			for j := 0; j <= bound; j++ {
				char := this.getChar(r, c)
				if char == '/' || char == '\\' || this.isMarker(char) {
					this.grid[r][c] = ' '
					c++
				} else if this.isCorner(char) {
					this.clearCorners = append(this.clearCorners, [2]int{r, c})
					c++
				} else if this.isTick(char) {
					this.grid[r][c] = '+'
				}
				r += rInc
			}

			this.grid[p.GridY][p.GridX] = ' '
			break
		}
	}
}

/*
 * Find style information for this polygon. This information is required to
 * exist on the first line after the top, touching the left wall. It's kind
 * of a pain requirement, but there's not a much better way to do it:
 * ditaa's handling requires too much text flung everywhere and this way
 * gives you a good method for specifying *tons* of information about the
 * object.
 */
func (this *ASCIIToSVG) findCommands(box *SVGPath) string {
	points := box.GetPoints()
	sX := points[0].GridX + 1
	sY := points[0].GridY + 1
	ref := ``
	if this.getChar(sY, sX) == '[' {
		sX++
		char := this.getChar(sY, sX)
		sX++
		for char != ']' {
			ref += string(char)
			char = this.getChar(sY, sX)
			sX++
		}

		if char == ']' {
			sX = points[0].GridX + 1
			sY = points[0].GridY + 1

			if `` == (this.commands[ref][`a2s:delref`]) &&
				`` == (this.commands[ref][`a2s:label`]) {
				this.grid[sY][sX] = ' '
				this.grid[sY][sX+len(ref)+1] = ' '
			} else {
				label := ``
				if `` != (this.commands[ref][`a2s:label`]) {
					label = this.commands[ref][`a2s:label`]
				}

				length := len(ref) + 2
				runes := []rune(label)
				for i := 0; i < length; i++ {
					if len(runes) > i {
						this.grid[sY][sX+i] = runes[i]
					} else {
						this.grid[sY][sX+i] = ' '
					}
				}
			}

			if nil != (this.commands[ref]) {
				box.SetOptions(this.commands[ref])
			}
		}
	}

	return ref
}

/*
 * Extremely useful debugging information to figure out what has been
 * parsed, especially when used in conjunction with clearObject.
 */
func (this *ASCIIToSVG) dumpGrid() {
	for _, line := range this.grid {
		fmt.Println(string(line))
	}
}

func (this *ASCIIToSVG) getChar(row, col int) rune {
	if row < 0 || col < 0 || row >= len(this.grid) || col >= len(this.grid[row]) {
		return 0
	}
	return this.grid[row][col]
}

// FIXME(akavel): func (this *ASCIIToSVG) isBoxEdge(char, dir = nil) {
func (this *ASCIIToSVG) isBoxEdge(char rune, dir ASCIIToSVG_DIR) bool {
	if dir == ASCIIToSVG_DIR_UNDEFINED {
		return char == '-' || char == '|' || char == ':' || char == '=' || char == '*' || char == '+'
	} else if dir == ASCIIToSVG_DIR_UP || dir == ASCIIToSVG_DIR_DOWN {
		return char == '|' || char == ':' || char == '*' || char == '+'
	} else if dir == ASCIIToSVG_DIR_LEFT || dir == ASCIIToSVG_DIR_RIGHT {
		return char == '-' || char == '=' || char == '*' || char == '+'
	}
	return false // FIXME(akavel): ok?
}

// FIXME(akavel): func (this *ASCIIToSVG) isEdge(char, dir = nil) {
func (this *ASCIIToSVG) isEdge(char rune, dir ASCIIToSVG_DIR) bool {
	if char == 'o' || char == 'X' {
		return true
	}

	if dir == ASCIIToSVG_DIR_UNDEFINED {
		return char == '-' || char == '|' || char == ':' || char == '=' || char == '*' || char == '/' || char == '\\'
	} else if dir == ASCIIToSVG_DIR_UP || dir == ASCIIToSVG_DIR_DOWN {
		return char == '|' || char == ':' || char == '*'
	} else if dir == ASCIIToSVG_DIR_LEFT || dir == ASCIIToSVG_DIR_RIGHT {
		return char == '-' || char == '=' || char == '*'
	} else if dir == ASCIIToSVG_DIR_NE {
		return char == '/'
	} else if dir == ASCIIToSVG_DIR_SE {
		return char == '\\'
	}
	return false
}

func (this *ASCIIToSVG) isBoxCorner(char rune) bool {
	return char == '.' || char == '\'' || char == '#'
}

func (this *ASCIIToSVG) isCorner(char rune) bool {
	return char == '.' || char == '\'' || char == '#' || char == '+'
}

func (this *ASCIIToSVG) isMarker(char rune) bool {
	return char == 'v' || char == '^' || char == '<' || char == '>'
}

func (this *ASCIIToSVG) isTick(char rune) bool {
	return char == 'o' || char == 'X'
}

/* vim:ts=2:sw=2:et:
 *  * */
