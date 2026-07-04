package cubestate

import "testing"

func countSolved(scramble []string) (int, int) {
	r := Memo(scramble)
	return len(r.Corners), len(r.Edges)
}

func TestMemoSolved(t *testing.T) {
	c, e := countSolved(nil)
	if c != 0 || e != 0 {
		t.Fatalf("solved cube memo should be empty, got %d corners %d edges", c, e)
	}
}

func TestMemoMoveInverse(t *testing.T) {
	c, e := countSolved([]string{"R", "U", "F", "F'", "U'", "R'"})
	if c != 0 || e != 0 {
		t.Fatalf("move+inverse should solve, got %d/%d", c, e)
	}
}

func TestMemoSingleQuarter(t *testing.T) {
	c, e := countSolved([]string{"U"})
	if c != 3 || e != 3 {
		t.Fatalf("single quarter turn should give 3 corner + 3 edge targets, got %d/%d", c, e)
	}
}

func TestMemoStickerFaceIdxBijective(t *testing.T) {
	fs := buildFacelets()
	perFace := map[string]map[int]bool{}
	for _, f := range fs {
		fl := stickerFaceIdx(f.home)
		if perFace[fl.Face] == nil {
			perFace[fl.Face] = map[int]bool{}
		}
		if fl.Idx == 4 {
			t.Fatalf("non-center sticker mapped to center idx: %v", fl)
		}
		if perFace[fl.Face][fl.Idx] {
			t.Fatalf("duplicate idx %d on face %s", fl.Idx, fl.Face)
		}
		perFace[fl.Face][fl.Idx] = true
	}
	for _, face := range letterFacesTest {
		if len(perFace[face]) != 8 {
			t.Fatalf("face %s has %d stickers, want 8", face, len(perFace[face]))
		}
	}
}

var letterFacesTest = []string{"U", "L", "F", "R", "B", "D"}
