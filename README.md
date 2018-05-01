# ASCIIToSVG

    .-------------------------.
    |                         |
    | .---.-. .-----. .-----. |
    | | .-. | +-->  | |  <--| |
    | | '-' | |  <--| +-->  | |
    | '---'-' '-----' '-----' |
    |  ascii     2      svg   |
    |                         |
    '-------------------------'

   https://github.com/asciitosvg

Create beautiful SVG diagrams from ASCII art.

## Introduction

### License

ASCIIToSVG is copyright © 2012-2018 The ASCIIToSVG contributors, and is
distributed under an MIT license. All code without explicit copyright retains
this license. Any code not adhering to this license is explicit in its own
copyright.

### What does it do?

ASCIIToSVG is a pretty simple Go library (with an accompanying CLI tool) that
parses ASCII art diagrams, attempting to convert them to an aesthetically
pleasing SVG output. Someone snidely remarked in a HN thread that if you make
a thing that generates visual output, show some examples of that output. Ask
GitHub to support insertion of SVG into their Markdown. In the meantime, you
can easily get example output by running `a2s -L`.

### What about that other project?

The original code at https://github.com/dhobsd/asciitosvg suffers from a
number of issues. It isn't particularly efficient in what's effectively a graph
search problem, and its implementation in PHP leaves a bit to be desired. Some
side-effects of this are that things one would really expect (like using `+`
for angled polygon corners) are not supported. This project subsumes the other,
which will eventually become unmaintained. Another advantage to the Go version
is that it supports UTF-8 inputs natively. Other features, like tab expansion,
have also been introduced.

Some other features, like custom objects, have not yet been implemented.

### So... why did you do this?

There are a few reasons:

 * We feel that Markdown is a great format for authoring documents.
 * When authoring technical documents, it's common to want to embed diagrams.
 * People say pictures are worth a thousand words.

Markdown support for inline images isn't particularly great, because you
cannot view the image in text mode. An ASCII diagram is usually doable for
many illustrations, but is often rendered as text in the HTML output. We'd
like the best of both worlds.

### Aren't there already things that do this?

Well, yes. There is a project called [ditaa][1] that has some of this
functionality. When dhobsd initially implemented A2S, ditaa left a couple key
points to be desired:

 * Its annotation format was too verbose, distracting from the visual content
 in text mode.
 * Its output was a rasterized format, so it didn't scale well.

Later, ditaa became unusable. The [software it was integrated with][2] was
written in PHP, and had to shell out to run ditaa. Because ditaa was written in
Java, new JVMs were spawned often to provide real-time feedback on diagram
changes. The functionality was re-implemented in PHP to output SVG, but this
ultimately ran into the issues described above. Hence this project.

## Compilation and Usage

To get the CLI tool, make sure `$GOPATH/bin` is in your `$PATH`. Run:

    $ go get github.com/asciitosvg/asciitosvg/cmd/a2s
    $ a2s -h
     .-------------------------.
     |                         |
     | .---.-. .-----. .-----. |
     | | .-. | +-->  | |  <--| |
     | | '-' | |  <--| +-->  | |
     | '---'-' '-----' '-----' |
     |  ascii     2      svg   |
     |                         |
     '-------------------------'

    https://github.com/asciitosvg

    [1,0]: {"fill":"#88d","a2s:delref":1}

    Usage of go/bin/a2s:
      -L	Generate SVG of the a2s logo.
      -b	Disable drop-shadow blur.
      -f string
            Font family to use. (default "Consolas,Monaco,Anonymous Pro,Anonymous,Bitstream Sans Mono,monospace")
      -i string
            Path to input text file. If set to "-" (hyphen), stdin is used. (default "-")
      -o string
            Path to output SVG file. If set to "-" (hyphen), stdout is used. (default "-")
      -t int
            Tab width. (default 8)
      -x int
            X grid scale in pixels. (default 9)
      -y int
            Y grid scale in pixels. (default 16)


To play with the library:

    $ go get github.com/asciitosvg/asciitosvg

Documentation on the API is available through your local `godoc` server.

## Drawing diagrams

Enough yammering about the impetus, code, and functionality. I bet you want
to draw something. ASCIIToSVG supports a few different ways to do that. 

### Basics: polygons and line segments

ASCIIToSVG supports concave and convex polygons with rounded or
degree-appropriate corners. Horizontal, vertical, and diagonal lines are all
supported. ASCIIToSVG has nearly complete support for output from
[App::Asciiio][3]. Edges of polygons and line segments can be drawn using the
following characters:

 * `-` or `=`: Horizontal lines, solid or dashed (respectively).
 * `|` or `:`: Vertical lines, solid or dashed (respectively).
 * `\` or `/`: Diagonal lines.
 * `+`: Edge of a line segment, or an angled corner.

Ticks and dots can be added into the middle of a line segment using `x` and
`o`, respectively. Note that these characters cannot be inserted into diagonal
lines, and they cannot begin a line.

To draw a polygon or turn a line, corners are necessary. The following
characters are valid corner characters:

 * `'` and `.`: Quadratic Bézier corners
 * `+`: Angled corners.

The `+` token is a control point for lines. It denotes an area where a line
intersects another line or traverses a box boundary.

A simple box with 3 rounded corners and a line pointing at it:

    +----------.
    |          | <---------
    '----------'

Diagonals may be used to form a closed polygon, but this is rarely a good idea.

### Basics: markers

Markers can be attached at the end of a line to give it a nice arrow by
using one of the following characters:

 * `<`: Left marker
 * `>`: Right marker
 * `^`: Up marker
 * `v`: Down marker

### Basics: text

Text can be inserted at almost any point in the image. Text is rendered in
a monospaced font. There aren't many restrictions, but obviously anything
you type that looks like it's actually a line or a box is going to be a good
candidate for turning into some pretty SVG path. Here is a box with some
plain black text in it:

    .-------------------------------------.
    | Hello here and there and everywhere |
    '-------------------------------------'

### Basics: formatting

It's possible to change the format of any boxes / polygons you create. This
is done (true to markdown form) by providing a reference on the top left
edge of the object, and defining that reference at the bottom of the input.
References *must* appear in the input *below* the objects they are associated
with, and preferably at the bottom of the diagram.

An example:

    .-------------.  .--------------.
    |[Red Box]    |  |[Blue Box]    |
    '-------------'  '--------------'

    [Red Box]: {"fill":"#aa4444"}
    [Blue Box]: {"fill":"#ccccff"}

Text appearing within a stylized box automatically tries to fix the color
contrast if the black text would be too dark on the background. The
reference commands can take any valid SVG properties / settings for a
[path element][4]. The commands are specified in JSON form, one per line.
Reference commands do not accept nested JSON objects -- don't try to
place additional curly braces inside! (Indeed, the current Go implementation
currently requires all JSON values other than `a2s:delref` to be strings.)

By default, the text of a reference is rendered inside the polygon, and the
reference is left in-tact in the output. You can remove the reference text
using the `a2s:delref` option; if it is set to any valid JSON value, it will
remove the reference text. You can use the `a2s:label` to replace the text with
any value, or to an empty string to remove it entirely.

The `a2s:link` option will wrap the target object with a clickable link to the
URL specified in the value.

#### Special references

It is possible to reference an object for formatting using its X and Y
coordinates. The coordinate system for ASCIIToSVG starts at (0, 0) in the top
left of the diagram.

Such references should only be made for stable diagrams, and only if you
*really* need to style text or a line in some particular way. These references
are marked by beginning the line with `[X,Y]` where `X` is the numeric row and 
`Y` is the numeric column of the object's top-left-most point.

## Unsupported features

The Go implementation does not yet support all the features of the PHP version.
Features that are currently unimplemented include:

 * Custom objects, including the `a2s:type` format specifier, are not yet
 implemented.
 * No support is planned for angled corners using `#`.
 * No support is planned for undirected lines using `*`.

## External Resources

There are some interesting sites that you might be interested in; these are
mildly related to the goals of ASCIIToSVG:

 * [ditaa][1] (previously mentioned) is a Java utility with a very similar
   syntax that translates ASCII diagrams into PNG files.
 * [App::Asciio][3] (previously mentioned) allows you to programmatically
   draw ASCII diagrams.
 * [Asciiflow][5] is a web front-end to designing ASCII flowcharts.

If you have something really cool that is related to ASCIIToSVG, and I have
failed to list it here, do please let me know.



[1]: http://ditaa.sourceforge.net/ "ditaa - DIagrams Through ASCII Art"
[2]: http://mtrack.wezfurlong.org/ "mtrack project management software"
[3]: http://search.cpan.org/dist/App-Asciio/lib/App/Asciio.pm "App::Asciio"
[4]: http://www.w3.org/TR/SVG/paths.html "SVG Paths"
[5]: http://www.asciiflow.com/ "Asciiflow"
