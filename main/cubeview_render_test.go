package main

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"cubie/cubestate"
)

func writePNG(t *testing.T, path string, v *CubeView) {
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, v.render()); err != nil {
		t.Fatal(err)
	}
}

func TestRenderPNG(t *testing.T) {
	dir := t.TempDir()
	solved := NewCubeView(cubestate.NewSolved())
	writePNG(t, filepath.Join(dir, "cube_solved.png"), solved)

	m := cubestate.NewSolved()
	for _, mv := range []string{"R", "U", "R'", "U'", "F", "R", "F'"} {
		m.Apply(mv)
	}
	scr := NewCubeView(m)
	writePNG(t, filepath.Join(dir, "cube_scrambled.png"), scr)
}
