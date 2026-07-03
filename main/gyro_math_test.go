package main

import (
	"math"
	"testing"

	"cubie/cube"
)

func TestAngleToAxis(t *testing.T) {
	if v := angleToAxis(3, 5, 45, false); v != 0 {
		t.Fatalf("inside deadzone want 0 got %d", v)
	}
	if v := angleToAxis(45, 5, 45, false); v != 32767 {
		t.Fatalf("at range want 32767 got %d", v)
	}
	if v := angleToAxis(90, 5, 45, false); v != 32767 {
		t.Fatalf("beyond range clamps want 32767 got %d", v)
	}
	if v := angleToAxis(-45, 5, 45, false); v != -32767 {
		t.Fatalf("negative want -32767 got %d", v)
	}
	if v := angleToAxis(45, 5, 45, true); v != -32767 {
		t.Fatalf("invert want -32767 got %d", v)
	}
}

func TestRelativeEulerIdentity(t *testing.T) {
	q := quatNormalize(cube.Quaternion{W: 0.9, X: 0.1, Y: -0.2, Z: 0.3})
	e := relativeEuler(q, q)
	if math.Abs(e.Pitch) > 1e-6 || math.Abs(e.Roll) > 1e-6 || math.Abs(e.Yaw) > 1e-6 {
		t.Fatalf("same orientation should be zero euler, got %+v", e)
	}
}

func TestRelativeEulerYaw(t *testing.T) {
	neutral := cube.Quaternion{W: 1}
	half := 20 * math.Pi / 180
	current := cube.Quaternion{W: math.Cos(half), Z: math.Sin(half)}
	e := relativeEuler(neutral, current)
	if math.Abs(e.Yaw-40) > 0.5 {
		t.Fatalf("yaw want ~40 got %f", e.Yaw)
	}
	if math.Abs(e.Pitch) > 0.5 || math.Abs(e.Roll) > 0.5 {
		t.Fatalf("pitch/roll should be ~0, got %+v", e)
	}
}
