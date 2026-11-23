// Copyright 2012 - 2018 The ASCIIToSVG Contributors
// All rights reserved.

package asciitosvg

// Compass direction indices ordered counter-clockwise in screen space (X right, Y down),
// i.e., decreasing atan2(dy, dx). Advancing (dirIdx + 1) % 8 from the incoming direction
// selects the sharpest clockwise (right-hand) turn.
const (
	dirIdxE = iota
	dirIdxNE
	dirIdxN
	dirIdxNW
	dirIdxW
	dirIdxSW
	dirIdxS
	dirIdxSE
)

type graphEdge struct {
	u, v     int
	isBridge bool
	inClosed bool
	inOpen   bool
}

type adjEntry struct {
	to     int
	edgeID int
	dirIdx int
}

type gridGraph struct {
	c       *canvas
	w, h    int
	edges   []graphEdge
	fullAdj [][]adjEntry
}

type candSeam struct {
	u, v         int
	dirUV, dirVU int
}

func newGridGraph(c *canvas) *gridGraph {
	w, h := c.size.X, c.size.Y
	n := w * h
	g := &gridGraph{
		c:       c,
		w:       w,
		h:       h,
		fullAdj: make([][]adjEntry, n),
	}
	if w == 0 || h == 0 {
		return g
	}

	var seams []candSeam

	// Build undirected edges using the 4 canonical forward directions (E, S, SE, NE).
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			u := y*w + x
			if x+1 < w {
				ok, seam := g.canConnectHorizontal(x, y)
				if ok {
					g.addEdge(u, y*w+(x+1), dirIdxE, dirIdxW)
				} else if seam {
					seams = append(seams, candSeam{u: u, v: y*w + (x + 1), dirUV: dirIdxE, dirVU: dirIdxW})
				}
			}
			if y+1 < h {
				ok, seam := g.canConnectVertical(x, y)
				if ok {
					g.addEdge(u, (y+1)*w+x, dirIdxS, dirIdxN)
				} else if seam {
					seams = append(seams, candSeam{u: u, v: (y+1)*w + x, dirUV: dirIdxS, dirVU: dirIdxN})
				}
			}
			if x+1 < w && y+1 < h && g.canConnectDiagSE(x, y) {
				v := (y+1)*w + (x + 1)
				g.addEdge(u, v, dirIdxSE, dirIdxNW)
			}
			if x+1 < w && y-1 >= 0 && g.canConnectDiagNE(x, y) {
				v := (y-1)*w + (x + 1)
				g.addEdge(u, v, dirIdxNE, dirIdxSW)
			}
		}
	}

	// Keep each vertex's adjacency list sorted in dirIdx order (0..7).
	for v := range g.fullAdj {
		sortAdjByDir(g.fullAdj[v])
	}

	// Only keep candidate ++ seams disconnected when both endpoints already belong
	// to distinct closed polygons (adjacent touching boxes). Adjacent line anchors
	// (e.g. ----++---- with || above/below, or parallel spurs anchored on a line/wall)
	// are connected so lines are not broken apart.
	if len(seams) > 0 {
		g.resolveSeams(seams)
	}

	return g
}

func (g *gridGraph) resolveSeams(seams []candSeam) {
	facesOf := g.findBoundedVertexFaces()
	dirty := make([]bool, len(g.fullAdj))
	for _, s := range seams {
		if hasDistinctFaces(facesOf[s.u], facesOf[s.v]) {
			continue
		}
		g.addEdge(s.u, s.v, s.dirUV, s.dirVU)
		dirty[s.u] = true
		dirty[s.v] = true
	}
	for v, d := range dirty {
		if d {
			sortAdjByDir(g.fullAdj[v])
		}
	}
}

func hasDistinctFaces(facesU, facesV []int) bool {
	if len(facesU) == 0 || len(facesV) == 0 {
		return false
	}
	uHasOwn := false
	for _, fu := range facesU {
		if !containsInt(facesV, fu) {
			uHasOwn = true
			break
		}
	}
	if !uHasOwn {
		return false
	}
	for _, fv := range facesV {
		if !containsInt(facesU, fv) {
			return true
		}
	}
	return false
}

func containsInt(s []int, target int) bool {
	for _, v := range s {
		if v == target {
			return true
		}
	}
	return false
}

func (g *gridGraph) addEdge(u, v, dirUV, dirVU int) {
	id := len(g.edges)
	g.edges = append(g.edges, graphEdge{u: u, v: v})
	g.fullAdj[u] = append(g.fullAdj[u], adjEntry{to: v, edgeID: id, dirIdx: dirUV})
	g.fullAdj[v] = append(g.fullAdj[v], adjEntry{to: u, edgeID: id, dirIdx: dirVU})
}

func sortAdjByDir(entries []adjEntry) {
	// len(entries) <= 8; insertion sort avoids slice allocation.
	for i := 1; i < len(entries); i++ {
		cur := entries[i]
		j := i - 1
		for j >= 0 && entries[j].dirIdx > cur.dirIdx {
			entries[j+1] = entries[j]
			j--
		}
		entries[j+1] = cur
	}
}

func (g *gridGraph) at(x, y int) char {
	if x < 0 || x >= g.w || y < 0 || y >= g.h {
		return ' '
	}
	return g.c.grid[y*g.w+x]
}

func (g *gridGraph) toPoint(v int) Point {
	return Point{X: v % g.w, Y: v / g.w}
}

func isHorizBasic(ch char) bool {
	return ch.isHorizontalLine() || ch.isCorner() || ch.isArrowHorizontal()
}

func isVertBasic(ch char) bool {
	return ch.isVerticalLine() || ch.isCorner() || ch.isArrowVertical()
}

func (g *gridGraph) isValidHorizontalTickOrDot(x, y int) bool {
	ch := g.at(x, y)
	if !ch.isTick() && !ch.isDot() {
		return false
	}
	l := g.at(x-1, y)
	r := g.at(x+1, y)
	if !isHorizBasic(l) || !isHorizBasic(r) {
		return false
	}
	// Do not allow inward-facing arrows that terminate against the tick/dot.
	if l == '>' || r == '<' {
		return false
	}
	return true
}

func (g *gridGraph) isValidVerticalTickOrDot(x, y int) bool {
	ch := g.at(x, y)
	if !ch.isTick() && !ch.isDot() {
		return false
	}
	if g.at(x-1, y).isAlnum() || g.at(x+1, y).isAlnum() {
		return false
	}
	u := g.at(x, y-1)
	d := g.at(x, y+1)
	if !isVertBasic(u) || !isVertBasic(d) {
		return false
	}
	if u == '\'' || u == 'v' || d == '.' || d == '^' {
		return false
	}
	return true
}

func (g *gridGraph) isHorizElem(x, y int) bool {
	ch := g.at(x, y)
	return isHorizBasic(ch) || g.isValidHorizontalTickOrDot(x, y)
}

func (g *gridGraph) isVertElem(x, y int) bool {
	ch := g.at(x, y)
	return isVertBasic(ch) || g.isValidVerticalTickOrDot(x, y)
}

func (g *gridGraph) hasHorizCont(x, y int) bool {
	ch := g.at(x, y)
	return isHorizBasic(ch) || ch.isVerticalLine() || g.isValidHorizontalTickOrDot(x, y)
}

func (g *gridGraph) hasVertCont(x, y int) bool {
	ch := g.at(x, y)
	return isVertBasic(ch) || ch.isHorizontalLine() || g.isValidVerticalTickOrDot(x, y)
}

func (g *gridGraph) isVertNeighbor(x, y int, fromTop bool) bool {
	ch := g.at(x, y)
	if ch.isVerticalLine() || ch.isAngledCorner() {
		return true
	}
	if fromTop {
		return ch == '.' || ch == '^'
	}
	return ch == '\'' || ch == 'v'
}

func (g *gridGraph) canConnectHorizontal(x, y int) (bool, bool) {
	cu := g.at(x, y)
	cv := g.at(x+1, y)

	// T-junction from a horizontal line into a vertical wall: |--- or ---|
	if cu.isVerticalLine() {
		return cv.isHorizontalLine() && g.hasHorizCont(x+2, y) &&
			(g.isVertNeighbor(x, y-1, true) || g.isVertNeighbor(x, y+1, false)), false
	}
	if cv.isVerticalLine() {
		return cu.isHorizontalLine() && g.hasHorizCont(x-1, y) &&
			(g.isVertNeighbor(x+1, y-1, true) || g.isVertNeighbor(x+1, y+1, false)), false
	}

	if !g.isHorizElem(x, y) || !g.isHorizElem(x+1, y) {
		return false, false
	}

	// Avoid turning inline prose like "C#-code" or "issue-#1" into tiny paths.
	if (cu == '#' || cv == '#') && g.at(x-1, y).isAlnum() && g.at(x+2, y).isAlnum() &&
		!g.isVertElem(x, y-1) && !g.isVertElem(x, y+1) &&
		!g.isVertElem(x+1, y-1) && !g.isVertElem(x+1, y+1) {
		return false, false
	}

	// Arrowheads stop at corners/opposite arrowheads, while allowing >--- and ---<.
	if cu == '>' && !cv.isHorizontalLine() && !cv.isTick() && !cv.isDot() {
		return false, false
	}
	if cv == '<' && !cu.isHorizontalLine() && !cu.isTick() && !cu.isDot() {
		return false, false
	}
	if cu == '<' && cv == '>' {
		return false, false
	}

	// Prevent adjacent boxes with back-to-back corners (e.g. +-++-+ or .--..--.) from merging,
	// while preserving adjacent '+'/'#' line anchors and parallel crossings (----++----).
	if cu.isCorner() && cv.isCorner() {
		uHasLeft := g.isHorizElem(x-1, y)
		vHasRight := g.isHorizElem(x+2, y)
		if uHasLeft && vHasRight {
			if !cu.isAngledCorner() || !cv.isAngledCorner() {
				return false, false
			}
			bothCrossings := g.at(x, y-1).isVerticalLine() && g.at(x, y+1).isVerticalLine() &&
				g.at(x+1, y-1).isVerticalLine() && g.at(x+1, y+1).isVerticalLine()
			if !bothCrossings {
				return false, true
			}
		}
	}

	return true, false
}

func (g *gridGraph) canConnectVertical(x, y int) (bool, bool) {
	cu := g.at(x, y)
	cv := g.at(x, y+1)

	// Rounded corners are strictly directional vertically: '.' opens downward, '\'' opens upward.
	if cu == '\'' {
		return false, false
	}
	if cv == '.' {
		// Allow a vertical line '|' / ':' to terminate onto a '.' corner from above,
		// but never connect '.' to '.' or '\'' to '.'.
		if !cu.isVerticalLine() || !g.isVertElem(x, y-1) {
			return false, false
		}
		ok := g.isHorizElem(x-1, y+1) || g.isHorizElem(x+1, y+1) ||
			g.at(x-1, y+2) == '/' || g.at(x+1, y+2) == '\\' ||
			g.at(x, y+2).isVerticalLine()
		return ok, false
	}

	// T-junction from a vertical line into a horizontal wall.
	if cu.isHorizontalLine() {
		if !cv.isVerticalLine() {
			return false, false
		}
		if g.at(x-1, y+1).isAlnum() || g.at(x+1, y+1).isAlnum() {
			return false, false
		}
		return g.hasVertCont(x, y+2), false
	}
	if cv.isHorizontalLine() {
		if !cu.isVerticalLine() {
			return false, false
		}
		if g.at(x-1, y).isAlnum() || g.at(x+1, y).isAlnum() {
			return false, false
		}
		return g.hasVertCont(x, y-1), false
	}

	if !g.isVertElem(x, y) || !g.isVertElem(x, y+1) {
		return false, false
	}

	// Arrowheads stop at corners/opposite arrowheads, while allowing v above | and ^ below |.
	if cu == 'v' && !cv.isVerticalLine() && !cv.isTick() && !cv.isDot() {
		return false, false
	}
	if cv == '^' && !cu.isVerticalLine() && !cu.isTick() && !cu.isDot() {
		return false, false
	}
	if cu == '^' && cv == 'v' {
		return false, false
	}

	// Candidate seam for vertically stacked boxes (+---+\n|   |\n+---+\n+---+\n|   |\n+---+);
	// adjacent line anchors (+--- above +--- along a vertical shaft) are reconnected in resolveSeams.
	if cu.isAngledCorner() && cv.isAngledCorner() {
		if g.isVertNeighbor(x, y-1, true) && g.isVertNeighbor(x, y+2, false) {
			uLeft := g.isHorizElem(x-1, y)
			uRight := g.isHorizElem(x+1, y)
			vLeft := g.isHorizElem(x-1, y+1)
			vRight := g.isHorizElem(x+1, y+1)
			if (uLeft && !uRight && vLeft && !vRight) || (!uLeft && uRight && !vLeft && vRight) {
				return false, true
			}
		}
	}

	return true, false
}

func (g *gridGraph) isStraightEndpoint(x, y int) bool {
	ch := g.at(x, y)
	if ch.isHorizontalLine() {
		l := g.hasHorizCont(x-1, y)
		r := g.hasHorizCont(x+1, y)
		return l != r
	}
	if ch.isVerticalLine() {
		u := g.hasVertCont(x, y-1)
		d := g.hasVertCont(x, y+1)
		return u != d
	}
	return false
}

func (g *gridGraph) isInlineDiagonal(x, y int) bool {
	return g.at(x-1, y).isAlnum() && g.at(x+1, y).isAlnum()
}

func (g *gridGraph) canConnectDiag(x, y, dy int, diagCh, uCorner, vCorner, uArrow, vArrow char) bool {
	cu := g.at(x, y)
	cv := g.at(x+1, y+dy)
	if cu != diagCh && cv != diagCh {
		return false
	}
	if (cu == diagCh && g.isInlineDiagonal(x, y)) ||
		(cv == diagCh && g.isInlineDiagonal(x+1, y+dy)) {
		return false
	}
	if cu == diagCh && cv == diagCh {
		return true
	}
	if cu == diagCh {
		if cv.isAngledCorner() || cv == vCorner || cv == vArrow || cv == '>' {
			return true
		}
		if cv.isHorizontalLine() || cv.isVerticalLine() {
			prev := g.at(x-1, y-dy)
			if prev == diagCh || prev.isAngledCorner() || prev == uCorner || prev == uArrow || prev == '<' {
				return true
			}
			return !g.at(x-1, y).isAlnum() && !g.at(x+1, y).isAlnum() && g.isStraightEndpoint(x+1, y+dy)
		}
		return false
	}
	// cv == diagCh
	if cu.isAngledCorner() || cu == uCorner || cu == uArrow || cu == '<' {
		return true
	}
	if cu.isHorizontalLine() || cu.isVerticalLine() {
		next := g.at(x+2, y+2*dy)
		if next == diagCh || next.isAngledCorner() || next == vCorner || next == vArrow || next == '>' {
			return true
		}
		return !g.at(x, y+dy).isAlnum() && !g.at(x+2, y+dy).isAlnum() && g.isStraightEndpoint(x, y)
	}
	return false
}

func (g *gridGraph) canConnectDiagSE(x, y int) bool {
	return g.canConnectDiag(x, y, 1, '\\', '.', '\'', '^', 'v')
}

func (g *gridGraph) canConnectDiagNE(x, y int) bool {
	return g.canConnectDiag(x, y, -1, '/', '\'', '.', 'v', '^')
}

// markBridges runs iterative Tarjan bridge detection so cut-edges and spurs
// are excluded from closed polygon face extraction.
func (g *gridGraph) markBridges() {
	for i := range g.edges {
		g.edges[i].isBridge = false
	}
	n := len(g.fullAdj)
	disc := make([]int, n)
	low := make([]int, n)
	timer := 0

	type frame struct {
		v          int
		parentEdge int
		adjIdx     int
	}
	var stack []frame

	for root := 0; root < n; root++ {
		if disc[root] != 0 || len(g.fullAdj[root]) == 0 {
			continue
		}
		timer++
		disc[root] = timer
		low[root] = timer
		stack = append(stack, frame{v: root, parentEdge: -1, adjIdx: 0})

		for len(stack) > 0 {
			top := &stack[len(stack)-1]
			v := top.v
			adj := g.fullAdj[v]

			if top.adjIdx < len(adj) {
				entry := adj[top.adjIdx]
				top.adjIdx++
				if entry.edgeID == top.parentEdge {
					continue
				}
				w := entry.to
				if disc[w] == 0 {
					timer++
					disc[w] = timer
					low[w] = timer
					stack = append(stack, frame{v: w, parentEdge: entry.edgeID, adjIdx: 0})
				} else if disc[w] < low[v] {
					low[v] = disc[w]
				}
			} else {
				pe := top.parentEdge
				stack = stack[:len(stack)-1]
				if pe != -1 && len(stack) > 0 {
					p := stack[len(stack)-1].v
					if low[v] < low[p] {
						low[p] = low[v]
					}
					if low[v] > disc[p] {
						g.edges[pe].isBridge = true
					}
				}
			}
		}
	}
}

func (g *gridGraph) findBoundedVertexFaces() [][]int {
	faces := g.walkBoundedFaces()
	facesOf := make([][]int, len(g.fullAdj))
	for faceID, f := range faces {
		for _, v := range f.verts {
			facesOf[v] = append(facesOf[v], faceID)
		}
	}
	return facesOf
}

type faceCycle struct {
	verts []int
	edges []int
	pts   []Point
}

func (g *gridGraph) walkBoundedFaces() []faceCycle {
	g.markBridges()

	n := len(g.fullAdj)
	cycAdj := make([][]adjEntry, n)
	for v := 0; v < n; v++ {
		for _, entry := range g.fullAdj[v] {
			if !g.edges[entry.edgeID].isBridge {
				cycAdj[v] = append(cycAdj[v], entry)
			}
		}
	}

	usedFwd := make([]bool, len(g.edges))
	usedRev := make([]bool, len(g.edges))
	isUsedHalf := func(edgeID, from int) bool {
		if g.edges[edgeID].u == from {
			return usedFwd[edgeID]
		}
		return usedRev[edgeID]
	}
	markUsedHalf := func(edgeID, from int) {
		if g.edges[edgeID].u == from {
			usedFwd[edgeID] = true
		} else {
			usedRev[edgeID] = true
		}
	}

	nextFaceEdge := func(v, incomingEdgeID int) (adjEntry, bool) {
		adj := cycAdj[v]
		for i, entry := range adj {
			if entry.edgeID == incomingEdgeID {
				return adj[(i+1)%len(adj)], true
			}
		}
		return adjEntry{}, false
	}

	var faces []faceCycle
	for u := 0; u < n; u++ {
		for _, startEntry := range cycAdj[u] {
			if isUsedHalf(startEntry.edgeID, u) {
				continue
			}

			var cycleVerts []int
			var cycleEdges []int
			curU, curV, curE := u, startEntry.to, startEntry.edgeID
			closedCycle := false
			for {
				if isUsedHalf(curE, curU) {
					break
				}
				markUsedHalf(curE, curU)
				cycleVerts = append(cycleVerts, curU)
				cycleEdges = append(cycleEdges, curE)

				nextEntry, ok := nextFaceEdge(curV, curE)
				if !ok {
					break
				}
				curU, curV, curE = curV, nextEntry.to, nextEntry.edgeID
				if curU == u && curE == startEntry.edgeID {
					closedCycle = true
					break
				}
			}

			if !closedCycle || len(cycleVerts) < 3 {
				continue
			}

			pts := make([]Point, len(cycleVerts))
			for i, v := range cycleVerts {
				pts[i] = g.toPoint(v)
			}
			// Bounded clockwise faces have strictly positive signed area in screen space;
			// the outer unbounded counter-clockwise face of each component has negative area.
			if signedArea2(pts) <= 0 {
				continue
			}

			faces = append(faces, faceCycle{
				verts: cycleVerts,
				edges: cycleEdges,
				pts:   pts,
			})
		}
	}
	return g.filterBridgeFaces(faces)
}

func (g *gridGraph) hasDir(x, y, dir int) bool {
	if x < 0 || x >= g.w || y < 0 || y >= g.h {
		return false
	}
	for _, e := range g.fullAdj[y*g.w+x] {
		if e.dirIdx == dir {
			return true
		}
	}
	return false
}

// filterBridgeFaces removes false closed cycles formed in the gap between two parallel
// open lines bridging two separate closed polygons (or crossing another pair of parallel lines).
func (g *gridGraph) filterBridgeFaces(faces []faceCycle) []faceCycle {
	if len(faces) == 0 {
		return faces
	}
	edgeFaceCount := make([]int, len(g.edges))
	for _, f := range faces {
		for _, eID := range f.edges {
			edgeFaceCount[eID]++
		}
	}

	kept := make([]faceCycle, 0, len(faces))
	for _, f := range faces {
		if g.isDoubleBridgeFace(f, edgeFaceCount) {
			continue
		}
		kept = append(kept, f)
	}
	return kept
}

func (g *gridGraph) isDoubleBridgeFace(f faceCycle, edgeFaceCount []int) bool {
	pts := rotateToTopLeft(append([]Point(nil), f.pts...))
	corners := pointsToCornersWithClosed(pts, true)
	if len(corners) != 4 {
		return false
	}
	xMin, yMin := corners[0].X, corners[0].Y
	xMax, yMax := corners[2].X, corners[2].Y
	if xMax <= xMin || yMax <= yMin {
		return false
	}
	if corners[1] != (Point{X: xMax, Y: yMin}) || corners[3] != (Point{X: xMin, Y: yMax}) {
		return false
	}

	topUnshared, botUnshared := true, true
	leftUnshared, rightUnshared := true, true
	topShared, botShared := true, true
	leftShared, rightShared := true, true

	for _, eID := range f.edges {
		pu := g.toPoint(g.edges[eID].u)
		pv := g.toPoint(g.edges[eID].v)
		shared := edgeFaceCount[eID] >= 2
		switch {
		case pu.Y == yMin && pv.Y == yMin:
			if shared {
				topUnshared = false
			} else {
				topShared = false
			}
		case pu.Y == yMax && pv.Y == yMax:
			if shared {
				botUnshared = false
			} else {
				botShared = false
			}
		case pu.X == xMin && pv.X == xMin:
			if shared {
				leftUnshared = false
			} else {
				leftShared = false
			}
		case pu.X == xMax && pv.X == xMax:
			if shared {
				rightUnshared = false
			} else {
				rightShared = false
			}
		default:
			return false
		}
	}

	// Horizontal double bridge between two vertical box walls: left and right sides belong
	// to other faces, top and bottom spans are exposed, and the horizontal bridges terminate
	// against the outer walls rather than continuing through them.
	if leftShared && rightShared && topUnshared && botUnshared {
		if !g.hasDir(xMin, yMin, dirIdxW) && !g.hasDir(xMin, yMax, dirIdxW) &&
			!g.hasDir(xMax, yMin, dirIdxE) && !g.hasDir(xMax, yMax, dirIdxE) {
			return true
		}
	}

	// Vertical double bridge between two horizontal box walls.
	if topShared && botShared && leftUnshared && rightUnshared {
		if !g.hasDir(xMin, yMin, dirIdxN) && !g.hasDir(xMax, yMin, dirIdxN) &&
			!g.hasDir(xMin, yMax, dirIdxS) && !g.hasDir(xMax, yMax, dirIdxS) {
			return true
		}
	}

	// Tight 1x1 cycle formed by two adjacent parallel lines crossing two perpendicular lines.
	if xMax-xMin == 1 && yMax-yMin == 1 && topUnshared && botUnshared && leftUnshared && rightUnshared {
		vertThrough := g.hasDir(xMin, yMin, dirIdxN) && g.hasDir(xMin, yMax, dirIdxS) &&
			g.hasDir(xMax, yMin, dirIdxN) && g.hasDir(xMax, yMax, dirIdxS)
		horizThrough := g.hasDir(xMin, yMin, dirIdxW) && g.hasDir(xMax, yMin, dirIdxE) &&
			g.hasDir(xMin, yMax, dirIdxW) && g.hasDir(xMax, yMax, dirIdxE)
		if vertThrough && horizThrough {
			return true
		}
	}

	return false
}

func (g *gridGraph) extractObjects() objects {
	onClosed := make([]bool, len(g.fullAdj))
	objs := g.extractClosedFaces(onClosed)
	return append(objs, g.extractOpenPaths(onClosed)...)
}

func (g *gridGraph) extractClosedFaces(onClosed []bool) objects {
	faces := g.walkBoundedFaces()
	objs := make(objects, 0, len(faces))
	for _, f := range faces {
		pts := rotateToTopLeft(f.pts)
		for _, eID := range f.edges {
			g.edges[eID].inClosed = true
		}
		for _, v := range f.verts {
			onClosed[v] = true
		}

		o := &object{points: pts}
		o.sealPath(g.c, true)
		g.markMidEdgeCusps(o)
		objs = append(objs, o)
	}
	return objs
}

// markMidEdgeCusps enables stylistic inward-curving cusps for '.' and '\” characters
// placed midway along a straight horizontal run (such as the 'a' in the ASCIIToSVG logo),
// provided the vertex is completely unambiguous (degree 2 between horizontal line chars,
// with no adjacent path touching its exterior side).
func (g *gridGraph) markMidEdgeCusps(o *object) {
	n := len(o.points)
	if n < 3 {
		return
	}
	for i, p := range o.points {
		if !o.isClosed && (i == 0 || i == n-1) {
			continue
		}
		if o.points[i].Hint != None {
			continue
		}
		ch := g.at(p.X, p.Y)
		if !ch.isRoundedCorner() {
			continue
		}
		v := p.Y*g.w + p.X
		if len(g.fullAdj[v]) != 2 {
			continue
		}
		prev := o.points[(i-1+n)%n]
		next := o.points[(i+1)%n]
		if prev.Y != p.Y || next.Y != p.Y || (p.X-prev.X)*(next.X-p.X) <= 0 {
			continue
		}
		if !g.at(prev.X, prev.Y).isHorizontalLine() || !g.at(next.X, next.Y).isHorizontalLine() {
			continue
		}

		if ch == '.' {
			if p.Y > 0 && len(g.fullAdj[(p.Y-1)*g.w+p.X]) > 0 {
				continue
			}
			if !o.isClosed && p.Y+1 < g.h && len(g.fullAdj[(p.Y+1)*g.w+p.X]) > 0 {
				continue
			}
			if g.at(p.X, p.Y+1).isHorizontalLine() {
				continue
			}
			o.points[i].Hint = RoundedCuspDown
		} else if ch == '\'' {
			if p.Y+1 < g.h && len(g.fullAdj[(p.Y+1)*g.w+p.X]) > 0 {
				continue
			}
			if !o.isClosed && p.Y > 0 && len(g.fullAdj[(p.Y-1)*g.w+p.X]) > 0 {
				continue
			}
			if g.at(p.X, p.Y-1).isHorizontalLine() {
				continue
			}
			o.points[i].Hint = RoundedCuspUp
		}
	}
}

func rotateToTopLeft(pts []Point) []Point {
	best := 0
	for i := 1; i < len(pts); i++ {
		if pts[i].Y < pts[best].Y || (pts[i].Y == pts[best].Y && pts[i].X < pts[best].X) {
			best = i
		}
	}
	if best == 0 {
		return pts
	}
	rotated := make([]Point, len(pts))
	copy(rotated, pts[best:])
	copy(rotated[len(pts)-best:], pts[:best])
	return rotated
}

func (g *gridGraph) extractOpenPaths(onClosed []bool) objects {
	n := len(g.fullAdj)
	openAdj := make([][]adjEntry, n)
	for v := 0; v < n; v++ {
		for _, entry := range g.fullAdj[v] {
			if !g.edges[entry.edgeID].inClosed {
				openAdj[v] = append(openAdj[v], entry)
			}
		}
	}

	remDeg := func(v int) int {
		cnt := 0
		for _, e := range openAdj[v] {
			if !g.edges[e.edgeID].inOpen {
				cnt++
			}
		}
		return cnt
	}

	hasEdgeInDir := func(adj []adjEntry, dirIdx int) bool {
		for _, e := range adj {
			if e.dirIdx == dirIdx {
				return true
			}
		}
		return false
	}

	var objs objects
	walkOpenPath := func(startV int, first adjEntry) {
		g.edges[first.edgeID].inOpen = true
		pts := []Point{g.toPoint(startV), g.toPoint(first.to)}
		curV := first.to
		inDir := first.dirIdx

		for {
			// Prefer continuing straight across crossings and T-junctions.
			var straight *adjEntry
			for i := range openAdj[curV] {
				e := &openAdj[curV][i]
				if !g.edges[e.edgeID].inOpen && e.dirIdx == inDir {
					straight = e
					break
				}
			}
			if straight != nil {
				if g.c.at(g.toPoint(curV)).isArrow() {
					break
				}
				g.edges[straight.edgeID].inOpen = true
				pts = append(pts, g.toPoint(straight.to))
				curV = straight.to
				continue
			}

			// Stop at dead ends, box boundaries, arrows, or multi-way junctions.
			if remDeg(curV) == 0 || onClosed[curV] || len(g.fullAdj[curV]) > 2 ||
				hasEdgeInDir(g.fullAdj[curV], inDir) || g.c.at(g.toPoint(curV)).isArrow() {
				break
			}

			// Pure degree-2 corner: turn along the single remaining edge.
			var nextEdge *adjEntry
			for i := range openAdj[curV] {
				e := &openAdj[curV][i]
				if !g.edges[e.edgeID].inOpen {
					nextEdge = e
					break
				}
			}
			if nextEdge == nil {
				break
			}
			g.edges[nextEdge.edgeID].inOpen = true
			pts = append(pts, g.toPoint(nextEdge.to))
			curV = nextEdge.to
			inDir = nextEdge.dirIdx
		}

		// Orient open paths consistently so points[0] is top-most, left-most when possible.
		last := pts[len(pts)-1]
		if last.Y < pts[0].Y || (last.Y == pts[0].Y && last.X < pts[0].X) {
			for i, j := 0, len(pts)-1; i < j; i, j = i+1, j-1 {
				pts[i], pts[j] = pts[j], pts[i]
			}
		}

		o := &object{points: pts}
		o.sealPath(g.c, false)
		g.markMidEdgeCusps(o)
		objs = append(objs, o)
	}

	// Pass 2a: Start from natural endpoints (open degree 1 or attached to a closed polygon).
	for u := 0; u < n; u++ {
		if len(openAdj[u]) == 0 {
			continue
		}
		if len(openAdj[u]) == 1 || onClosed[u] {
			for _, e := range openAdj[u] {
				if !g.edges[e.edgeID].inOpen {
					walkOpenPath(u, e)
				}
			}
		}
	}

	// Pass 2b: Pick up any remaining edges (e.g., junctions with no degree-1 leaves).
	for u := 0; u < n; u++ {
		for _, e := range openAdj[u] {
			if !g.edges[e.edgeID].inOpen {
				walkOpenPath(u, e)
			}
		}
	}

	return objs
}
