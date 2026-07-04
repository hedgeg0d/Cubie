package cubestate

import "testing"

var solvedState = [18]byte{0, 0, 0, 36, 146, 73, 73, 36, 146, 109, 182, 219, 146, 73, 36, 182, 219, 109}

func colorByPos(m *Model) map[Vec3]int {
	out := make(map[Vec3]int, len(m.Stickers))
	for _, s := range m.Stickers {
		out[s.Pos] = s.Color
	}
	return out
}

func sameLayout(a, b *Model) bool {
	ma, mb := colorByPos(a), colorByPos(b)
	if len(ma) != len(mb) {
		return false
	}
	for p, c := range ma {
		if mb[p] != c {
			return false
		}
	}
	return true
}

func applied(moves ...string) *Model {
	m := NewSolved()
	for _, mv := range moves {
		m.Apply(mv)
	}
	return m
}

func TestModelFromSolvedState(t *testing.T) {
	got := ModelFromState(solvedState)
	if len(got.Stickers) != 54 {
		t.Fatalf("expected 54 stickers, got %d", len(got.Stickers))
	}
	if !got.IsSolved() {
		t.Error("reconstructed solved state not solved")
	}
	if !sameLayout(got, NewSolved()) {
		t.Error("ModelFromState(solved) layout != NewSolved()")
	}
}

func TestDecodeFaceValues(t *testing.T) {
	got := decodeFaceValues([]byte{36, 146, 73})
	for k, v := range got {
		if v != 1 {
			t.Fatalf("sticker %d = %d, want 1", k, v)
		}
	}
}

func TestModelFromScrambledStates(t *testing.T) {
	cases := []struct {
		state [18]byte
		moves []string
	}{
		{[18]byte{1, 134, 3, 68, 162, 137, 72, 32, 144, 108, 178, 217, 146, 73, 36, 182, 219, 109}, []string{"R"}},
		{[18]byte{182, 134, 3, 146, 34, 137, 73, 36, 0, 108, 178, 217, 1, 201, 36, 68, 219, 109}, []string{"R", "U"}},
		{[18]byte{2, 138, 221, 146, 34, 137, 73, 37, 35, 181, 50, 217, 1, 199, 33, 4, 138, 45}, []string{"R", "U", "F"}},
		{[18]byte{130, 196, 162, 134, 186, 219, 20, 162, 41, 44, 200, 0, 69, 37, 13, 109, 139, 5}, []string{"U'", "B", "F'", "R'", "D"}},
	}
	for _, c := range cases {
		if !sameLayout(ModelFromState(c.state), applied(c.moves...)) {
			t.Errorf("ModelFromState mismatch for moves %v", c.moves)
		}
	}
}
