package cubestate

import (
	"reflect"
	"testing"
)

func TestGenerateScrambleLength(t *testing.T) {
	for _, n := range []int{5, 20, 25} {
		s := GenerateScramble(n)
		if len(s) != n {
			t.Errorf("GenerateScramble(%d) len = %d", n, len(s))
		}
	}
}

func TestGenerateScrambleNoConsecutiveSameFace(t *testing.T) {
	for range 200 {
		s := GenerateScramble(25)
		for i := 1; i < len(s); i++ {
			if s[i][:1] == s[i-1][:1] {
				t.Fatalf("consecutive same face at %d: %v", i, s)
			}
		}
	}
}

func TestInvertMove(t *testing.T) {
	cases := map[string]string{"R": "R'", "R'": "R", "F2": "F2", "U": "U'"}
	for in, want := range cases {
		if got := InvertMove(in); got != want {
			t.Errorf("InvertMove(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestInvertScramble(t *testing.T) {
	in := []string{"R", "U'", "F2"}
	want := []string{"F2", "U", "R'"}
	if got := InvertScramble(in); !reflect.DeepEqual(got, want) {
		t.Errorf("InvertScramble(%v) = %v, want %v", in, got, want)
	}
	if got := InvertScramble(InvertScramble(in)); !reflect.DeepEqual(got, in) {
		t.Errorf("double invert = %v, want %v", got, in)
	}
}

func TestOrientationIdentity(t *testing.T) {
	o := NewOrientation()
	for _, m := range []string{"R", "U'", "F", "D2"} {
		if got := o.Remap(m); got != m {
			t.Fatalf("identity remap %s -> %s", m, got)
		}
	}
}

func TestOrientationYRemap(t *testing.T) {
	o := NewOrientation()
	o.Apply("y")
	if got := o.Remap("F"); got != "L" {
		t.Fatalf("after y, absolute F should map to L, got %s", got)
	}
	if got := o.Remap("R'"); got != "F'" {
		t.Fatalf("after y, absolute R' should map to F', got %s", got)
	}
	if got := o.Remap("U"); got != "U" {
		t.Fatalf("after y, U axis unchanged, got %s", got)
	}
}

func TestOrientationRoundTrip(t *testing.T) {
	o := NewOrientation()
	o.Apply("y")
	o.Apply("y'")
	for _, m := range []string{"R", "U'", "F", "L2", "B", "D"} {
		if got := o.Remap(m); got != m {
			t.Fatalf("y then y' should be identity, %s -> %s", m, got)
		}
	}
}

func TestOrientationYDouble(t *testing.T) {
	o := NewOrientation()
	o.Apply("y2")
	if got := o.Remap("F"); got != "B" {
		t.Fatalf("after y2, F should map to B, got %s", got)
	}
}
