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
		suffix := suffixes[rand.Intn(len(suffixes))]
		moves = append(moves, face+suffix)
		prevPrev = prev
		prev = face
	}
	return moves
}

func ScrambleString(moves []string) string {
	return strings.Join(moves, " ")
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
