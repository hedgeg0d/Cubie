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

func quatToEuler(q cube.Quaternion) Euler {
	const rad2deg = 180 / math.Pi
	roll := math.Atan2(2*(q.W*q.X+q.Y*q.Z), 1-2*(q.X*q.X+q.Y*q.Y))
	sinp := 2 * (q.W*q.Y - q.Z*q.X)
	var pitch float64
	if math.Abs(sinp) >= 1 {
		pitch = math.Copysign(math.Pi/2, sinp)
	} else {
		pitch = math.Asin(sinp)
	}
	yaw := math.Atan2(2*(q.W*q.Z+q.X*q.Y), 1-2*(q.Y*q.Y+q.Z*q.Z))
	return Euler{Pitch: pitch * rad2deg, Roll: roll * rad2deg, Yaw: yaw * rad2deg}
}

func relativeEuler(neutral, current cube.Quaternion) Euler {
	rel := quatNormalize(quatMul(quatConj(neutral), current))
	return quatToEuler(rel)
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
