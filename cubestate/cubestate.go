package cubestate

import (
	"math/rand"
	"strings"
)

var faces = []string{"U", "D", "R", "L", "F", "B"}

var oppositeFace = map[string]string{
	"U": "D", "D": "U",
	"R": "L", "L": "R",
	"F": "B", "B": "F",
}

var suffixes = []string{"", "'", "2"}

func GenerateScramble(n int) []string {
	return genScramble(n, suffixes)
}

func GenerateScrambleQuarter(n int) []string {
	return genScramble(n, []string{"", "'"})
}

func genScramble(n int, allowedSuffixes []string) []string {
	moves := make([]string, 0, n)
	var prev, prevPrev string
	for len(moves) < n {
		face := faces[rand.Intn(len(faces))]
		if face == prev {
			continue
		}
		if face == prevPrev && oppositeFace[face] == prev {
			continue
		}
		suffix := allowedSuffixes[rand.Intn(len(allowedSuffixes))]
		moves = append(moves, face+suffix)
		prevPrev = prev
		prev = face
	}
	return moves
}

func ScrambleString(moves []string) string {
	return strings.Join(moves, " ")
}

var rotationAxes = map[string][]string{
	"x": {"F", "U", "B", "D"},
	"y": {"F", "L", "B", "R"},
	"z": {"U", "R", "D", "L"},
}

type Orientation struct {
	perm map[string]string
}

func NewOrientation() *Orientation {
	p := make(map[string]string, len(faces))
	for _, f := range faces {
		p[f] = f
	}
	return &Orientation{perm: p}
}

func (o *Orientation) Apply(rot string) {
	if rot == "" {
		return
	}
	cycle, ok := rotationAxes[rot[:1]]
	if !ok {
		return
	}
	prime := strings.HasSuffix(rot, "'")
	steps := 1
	if strings.HasSuffix(rot, "2") {
		steps = 2
	}
	for s := 0; s < steps; s++ {
		g := make(map[string]string, len(faces))
		for _, f := range faces {
			g[f] = f
		}
		for i, p := range cycle {
			if prime {
				g[p] = cycle[(i+3)%4]
			} else {
				g[p] = cycle[(i+1)%4]
			}
		}
		newPerm := make(map[string]string, len(faces))
		for _, q := range faces {
			for from, to := range g {
				if to == q {
					newPerm[q] = o.perm[from]
					break
				}
			}
		}
		o.perm = newPerm
	}
}

func (o *Orientation) Remap(move string) string {
	if move == "" {
		return move
	}
	face := move[:1]
	for pos, abs := range o.perm {
		if abs == face {
			return pos + move[1:]
		}
	}
	return move
}

func InvertMove(move string) string {
	if move == "" {
		return ""
	}
	face := move[:1]
	suffix := move[1:]
	switch suffix {
	case "":
		return face + "'"
	case "'":
		return face
	case "2":
		return move
	default:
		return move
	}
}

func InvertScramble(moves []string) []string {
	inv := make([]string, len(moves))
	for i, m := range moves {
		inv[len(moves)-1-i] = InvertMove(m)
	}
	return inv
}
