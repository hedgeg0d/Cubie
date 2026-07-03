package main

import (
	"math"
	"testing"

	"cubie/cube"
)

func quatAxis(axis string, deg float64) cube.Quaternion {
	r := deg * math.Pi / 180
	s, c := math.Sin(r/2), math.Cos(r/2)
	switch axis {
	case "x":
		return cube.Quaternion{W: c, X: s}
	case "y":
		return cube.Quaternion{W: c, Y: s}
	default:
		return cube.Quaternion{W: c, Z: s}
	}
}

func TestClassifyRotation(t *testing.T) {
	cases := []struct {
		e    Euler
		want string
		ok   bool
	}{
		{Euler{Roll: 90}, "y", true},
		{Euler{Roll: -88}, "y'", true},
		{Euler{Roll: 180}, "y2", true},
		{Euler{Pitch: 92}, "x", true},
		{Euler{Yaw: -90}, "z'", true},
		{Euler{Roll: 40}, "", false},
		{Euler{Pitch: 60, Roll: 60}, "", false},
		{Euler{Roll: 50}, "", false},
		{Euler{Roll: 60}, "y", true},
	}
	for _, c := range cases {
		got, ok := classifyRotation(c.e)
		if ok != c.ok || got != c.want {
			t.Fatalf("classify %+v -> (%q,%v), want (%q,%v)", c.e, got, ok, c.want, c.ok)
		}
	}
}

func TestDetectorHold(t *testing.T) {
	var d rotationDetector
	d.feed(0, quatAxis("y", 0))
	if _, ok := d.feed(50, quatAxis("y", 90)); ok {
		t.Fatal("should not emit before settle")
	}
	if _, ok := d.feed(100, quatAxis("y", 90)); ok {
		t.Fatal("should not emit before settle dwell")
	}
	label, ok := d.feed(200, quatAxis("y", 90))
	if !ok || label != "y" {
		t.Fatalf("held 90 should emit y, got (%q,%v)", label, ok)
	}
	if _, ok := d.feed(250, quatAxis("y", 90)); ok {
		t.Fatal("should not re-emit after commit (ref rebased)")
	}
}

func TestDetectorTransient(t *testing.T) {
	var d rotationDetector
	d.feed(0, quatAxis("y", 0))
	d.feed(50, quatAxis("y", 90))
	if _, ok := d.feed(80, quatAxis("y", 0)); ok {
		t.Fatal("transient tilt returning home must not emit")
	}
	if _, ok := d.feed(110, quatAxis("y", 90)); ok {
		t.Fatal("candidate should restart, no emit yet")
	}
	if _, ok := d.feed(140, quatAxis("y", 90)); ok {
		t.Fatal("still within dwell window, no emit")
	}
}

func TestDetectorDouble(t *testing.T) {
	var d rotationDetector
	d.feed(0, quatAxis("y", 0))
	d.feed(50, quatAxis("y", 180))
	d.feed(100, quatAxis("y", 180))
	label, ok := d.feed(200, quatAxis("y", 180))
	if !ok || label != "y2" {
		t.Fatalf("held 180 should emit y2, got (%q,%v)", label, ok)
	}
}
