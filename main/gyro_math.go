package main

import (
	"math"

	"cubie/cube"
)

type Euler struct {
	Pitch, Roll, Yaw float64
}

func quatNormalize(q cube.Quaternion) cube.Quaternion {
	n := math.Sqrt(q.W*q.W + q.X*q.X + q.Y*q.Y + q.Z*q.Z)
	if n < 1e-9 {
		return cube.Quaternion{W: 1}
	}
	return cube.Quaternion{W: q.W / n, X: q.X / n, Y: q.Y / n, Z: q.Z / n}
}

func quatConj(q cube.Quaternion) cube.Quaternion {
	return cube.Quaternion{W: q.W, X: -q.X, Y: -q.Y, Z: -q.Z}
}

func quatMul(a, b cube.Quaternion) cube.Quaternion {
	return cube.Quaternion{
		W: a.W*b.W - a.X*b.X - a.Y*b.Y - a.Z*b.Z,
		X: a.W*b.X + a.X*b.W + a.Y*b.Z - a.Z*b.Y,
		Y: a.W*b.Y - a.X*b.Z + a.Y*b.W + a.Z*b.X,
		Z: a.W*b.Z + a.X*b.Y - a.Y*b.X + a.Z*b.W,
	}
}

func rotationVector(q cube.Quaternion) Euler {
	const rad2deg = 180 / math.Pi
	s := math.Sqrt(q.X*q.X + q.Y*q.Y + q.Z*q.Z)
	if s < 1e-9 {
		return Euler{Pitch: 2 * q.X * rad2deg, Roll: 2 * q.Y * rad2deg, Yaw: 2 * q.Z * rad2deg}
	}
	angle := 2 * math.Atan2(s, q.W)
	k := angle / s * rad2deg
	return Euler{Pitch: q.X * k, Roll: q.Y * k, Yaw: q.Z * k}
}

func relativeEuler(neutral, current cube.Quaternion) Euler {
	rel := quatNormalize(quatMul(quatConj(neutral), current))
	if rel.W < 0 {
		rel = cube.Quaternion{W: -rel.W, X: -rel.X, Y: -rel.Y, Z: -rel.Z}
	}
	return rotationVector(rel)
}

func angleToAxis(angle, deadzone, rangeDeg float64, invert bool) int32 {
	mag := math.Abs(angle)
	if mag <= deadzone {
		return 0
	}
	span := rangeDeg - deadzone
	if span < 1e-6 {
		span = 1e-6
	}
	norm := (mag - deadzone) / span
	if norm > 1 {
		norm = 1
	}
	v := math.Copysign(norm, angle)
	if invert {
		v = -v
	}
	return int32(math.Round(v * 32767))
}
