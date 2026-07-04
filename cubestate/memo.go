package cubestate

import "math"

type Facelet struct {
	Face string
	Idx  int
}

type MemoResult struct {
	Corners []Facelet
	Edges   []Facelet
}

type mFacelet struct {
	pos  Vec3
	home Vec3
}

func roundUnit(v float64) int {
	if v >= 0.5 {
		return 1
	}
	if v <= -0.5 {
		return -1
	}
	return 0
}

func cubieOf(p Vec3) [3]int {
	return [3]int{roundUnit(p.X), roundUnit(p.Y), roundUnit(p.Z)}
}

func nonZero(c [3]int) int {
	n := 0
	for _, v := range c {
		if v != 0 {
			n++
		}
	}
	return n
}

func pkey(p Vec3) [3]int {
	return [3]int{int(math.Round(p.X * 2)), int(math.Round(p.Y * 2)), int(math.Round(p.Z * 2))}
}

func stickerFaceIdx(p Vec3) Facelet {
	m := func(v float64) int { return roundUnit(v) + 1 }
	switch {
	case p.X >= 1.4:
		return Facelet{"R", m(-p.Y)*3 + m(-p.Z)}
	case p.X <= -1.4:
		return Facelet{"L", m(-p.Y)*3 + m(p.Z)}
	case p.Y >= 1.4:
		return Facelet{"U", m(p.Z)*3 + m(p.X)}
	case p.Y <= -1.4:
		return Facelet{"D", m(-p.Z)*3 + m(p.X)}
	case p.Z >= 1.4:
		return Facelet{"F", m(-p.Y)*3 + m(p.X)}
	default:
		return Facelet{"B", m(-p.Y)*3 + m(-p.X)}
	}
}

func buildFacelets() []mFacelet {
	inner := []float64{-1, 0, 1}
	var fs []mFacelet
	for _, name := range faceOrder {
		f := faceSpecs[name]
		for _, a := range inner {
			for _, b := range inner {
				var p Vec3
				switch f.axis {
				case 0:
					p = Vec3{1.5 * f.sign, a, b}
				case 1:
					p = Vec3{a, 1.5 * f.sign, b}
				case 2:
					p = Vec3{a, b, 1.5 * f.sign}
				}
				if nonZero(cubieOf(p)) == 1 {
					continue
				}
				fs = append(fs, mFacelet{pos: p, home: p})
			}
		}
	}
	return fs
}

func applyMemoMove(fs []mFacelet, move string) {
	axis, sign, dir, quarters, ok := MoveInfo(move)
	if !ok {
		return
	}
	for range quarters {
		for i := range fs {
			if onLayer(fs[i].pos, axis, sign) {
				fs[i].pos = rot90(fs[i].pos, axis, dir)
			}
		}
	}
}

func traceCycles(slots []Vec3, home map[[3]int]Vec3, buffer Vec3) []Facelet {
	var out []Facelet
	visited := map[[3]int]bool{}
	const guard = 60

	homeOf := func(p Vec3) Vec3 { return home[pkey(p)] }

	slotsOf := map[[3]int][]Vec3{}
	for _, s := range slots {
		cb := cubieOf(s)
		slotsOf[cb] = append(slotsOf[cb], s)
	}
	solved := func(cb [3]int) bool {
		for _, s := range slotsOf[cb] {
			if pkey(homeOf(s)) != pkey(s) {
				return false
			}
		}
		return true
	}

	traceFrom := func(start Vec3, emitStart bool) {
		cur := start
		if emitStart {
			out = append(out, stickerFaceIdx(start))
		}
		visited[cubieOf(start)] = true
		for i := 0; i < guard; i++ {
			h := homeOf(cur)
			if cubieOf(h) == cubieOf(start) {
				return
			}
			out = append(out, stickerFaceIdx(h))
			visited[cubieOf(h)] = true
			cur = h
		}
	}

	traceFrom(buffer, false)

	for _, s := range slots {
		cb := cubieOf(s)
		if visited[cb] || solved(cb) {
			visited[cb] = true
			continue
		}
		traceFrom(s, true)
	}
	return out
}

func Memo(scramble []string) MemoResult {
	fs := buildFacelets()
	for _, mv := range scramble {
		applyMemoMove(fs, mv)
	}
	home := map[[3]int]Vec3{}
	for _, f := range fs {
		home[pkey(f.pos)] = f.home
	}

	var cornerSlots, edgeSlots []Vec3
	for _, f := range fs {
		switch nonZero(cubieOf(f.home)) {
		case 3:
			cornerSlots = append(cornerSlots, f.home)
		case 2:
			edgeSlots = append(edgeSlots, f.home)
		}
	}

	cornerBuffer := Vec3{-1, 1.5, -1}
	edgeBuffer := Vec3{0, 1.5, 1}

	return MemoResult{
		Corners: traceCycles(cornerSlots, home, cornerBuffer),
		Edges:   traceCycles(edgeSlots, home, edgeBuffer),
	}
}
