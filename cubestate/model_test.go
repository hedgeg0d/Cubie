package cubestate

import "testing"

func TestSolvedModel(t *testing.T) {
	m := NewSolved()
	if len(m.Stickers) != 54 {
		t.Fatalf("expected 54 stickers, got %d", len(m.Stickers))
	}
	if !m.IsSolved() {
		t.Error("fresh model not solved")
	}
}

func TestSingleMoveUnsolves(t *testing.T) {
	for _, mv := range []string{"R", "U", "F", "L", "D", "B"} {
		m := NewSolved()
		m.Apply(mv)
		if m.IsSolved() {
			t.Errorf("model still solved after %s", mv)
		}
	}
}

func TestMoveInverse(t *testing.T) {
	for _, mv := range []string{"R", "U", "F", "L", "D", "B"} {
		m := NewSolved()
		m.Apply(mv)
		m.Apply(InvertMove(mv))
		if !m.IsSolved() {
			t.Errorf("%s then %s did not restore solved", mv, InvertMove(mv))
		}
	}
}

func TestQuadrupleMoveIdentity(t *testing.T) {
	for _, mv := range []string{"R", "U", "F", "L", "D", "B"} {
		m := NewSolved()
		for range 4 {
			m.Apply(mv)
		}
		if !m.IsSolved() {
			t.Errorf("%s x4 did not restore solved", mv)
		}
	}
}

func TestDoubleMove(t *testing.T) {
	m := NewSolved()
	m.Apply("R2")
	m2 := NewSolved()
	m2.Apply("R")
	m2.Apply("R")
	for i := range m.Stickers {
		if m.Stickers[i] != m2.Stickers[i] {
			t.Fatalf("R2 != R R at sticker %d", i)
		}
	}
}

func TestScrambleThenInverseSolves(t *testing.T) {
	scramble := GenerateScramble(25)
	m := NewSolved()
	for _, mv := range scramble {
		m.Apply(mv)
	}
	for _, mv := range InvertScramble(scramble) {
		m.Apply(mv)
	}
	if !m.IsSolved() {
		t.Error("scramble followed by its inverse did not solve")
	}
}
