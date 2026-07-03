package main

import (
	"math"

	"cubie/cube"
)

const (
	rotMinAngle = 55.0
	rotWobble   = 30.0
	rotSnapTol  = 30.0
	rotSettleMs = 120
	rotBackDeg  = 30.0
)

func rotationMagnitude(e Euler) float64 {
	return math.Sqrt(e.Pitch*e.Pitch + e.Roll*e.Roll + e.Yaw*e.Yaw)
}

func classifyRotation(e Euler) (string, bool) {
	ax, ay, az := math.Abs(e.Pitch), math.Abs(e.Roll), math.Abs(e.Yaw)
	var axis string
	var dv, o1, o2 float64
	switch {
	case ax >= ay && ax >= az:
		axis, dv, o1, o2 = "x", e.Pitch, ay, az
	case ay >= az:
		axis, dv, o1, o2 = "y", e.Roll, ax, az
	default:
		axis, dv, o1, o2 = "z", e.Yaw, ax, ay
	}
	if math.Max(o1, o2) > rotWobble {
		return "", false
	}
	adv := math.Abs(dv)
	if adv < rotMinAngle {
		return "", false
	}
	k := int(math.Round(adv / 90))
	if k < 1 {
		return "", false
	}
	if k > 2 {
		k = 2
	}
	if math.Abs(adv-float64(k)*90) > rotSnapTol {
		return "", false
	}
	if k == 2 {
		return axis + "2", true
	}
	if dv >= 0 {
		return axis, true
	}
	return axis + "'", true
}

type rotationDetector struct {
	ref       cube.Quaternion
	haveRef   bool
	cand      string
	candStart int64
}

func (d *rotationDetector) feed(t int64, q cube.Quaternion) (string, bool) {
	if q == (cube.Quaternion{}) {
		return "", false
	}
	if !d.haveRef {
		d.ref, d.haveRef = q, true
		return "", false
	}
	e := relativeEuler(d.ref, q)
	label, ok := classifyRotation(e)
	if !ok {
		if rotationMagnitude(e) < rotBackDeg {
			d.ref = q
		}
		d.cand = ""
		return "", false
	}
	if d.cand != label {
		d.cand = label
		d.candStart = t
		return "", false
	}
	if t-d.candStart < rotSettleMs {
		return "", false
	}
	d.ref = q
	d.cand = ""
	return label, true
}
