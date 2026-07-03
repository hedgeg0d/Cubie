package cubestate

const (
	ColorU = iota
	ColorD
	ColorR
	ColorL
	ColorF
	ColorB
)

type Vec3 struct {
	X, Y, Z float64
}

type Sticker struct {
	Pos   Vec3
	Color int
}

type Model struct {
	Stickers []Sticker
}

type faceSpec struct {
	axis  int
	sign  float64
	dir   int
	color int
}

var faceSpecs = map[string]faceSpec{
	"R": {0, 1, -1, ColorR},
	"L": {0, -1, 1, ColorL},
	"U": {1, 1, -1, ColorU},
	"D": {1, -1, 1, ColorD},
	"F": {2, 1, -1, ColorF},
	"B": {2, -1, 1, ColorB},
}

var faceOrder = []string{"U", "D", "R", "L", "F", "B"}

func NewSolved() *Model {
	inner := []float64{-1, 0, 1}
	m := &Model{}
	for _, name := range faceOrder {
		f := faceSpecs[name]
		for _, a := range inner {
			for _, b := range inner {
				var p Vec3
				switch f.axis {
				case 0:
					p = Vec3{1.5 * f.sign, a, b}
				case 1:
					p = Vec3{a, 1.5 * f.sign, b}
				case 2:
					p = Vec3{a, b, 1.5 * f.sign}
				}
				m.Stickers = append(m.Stickers, Sticker{Pos: p, Color: f.color})
			}
		}
	}
	return m
}

func (m *Model) Apply(move string) {
	if move == "" {
		return
	}
	f, ok := faceSpecs[move[:1]]
	if !ok {
		return
	}
	quarters := 1
	dir := f.dir
	switch move[1:] {
	case "":
	case "'":
		dir = -f.dir
	case "2":
		quarters = 2
	default:
		return
	}
	for range quarters {
		m.turn(f.axis, f.sign, dir)
	}
}

func (m *Model) turn(axis int, sign float64, dir int) {
	for i := range m.Stickers {
		if onLayer(m.Stickers[i].Pos, axis, sign) {
			m.Stickers[i].Pos = rot90(m.Stickers[i].Pos, axis, dir)
		}
	}
}

func OnLayer(p Vec3, axis int, sign float64) bool {
	return onLayer(p, axis, sign)
}

func MoveInfo(move string) (axis int, sign float64, dir, quarters int, ok bool) {
	if move == "" {
		return 0, 0, 0, 0, false
	}
	f, found := faceSpecs[move[:1]]
	if !found {
		return 0, 0, 0, 0, false
	}
	dir = f.dir
	quarters = 1
	switch move[1:] {
	case "":
	case "'":
		dir = -f.dir
	case "2":
		quarters = 2
	default:
		return 0, 0, 0, 0, false
	}
	return f.axis, f.sign, dir, quarters, true
}

func onLayer(p Vec3, axis int, sign float64) bool {
	var comp float64
	switch axis {
	case 0:
		comp = p.X
	case 1:
		comp = p.Y
	case 2:
		comp = p.Z
	}
	if sign > 0 {
		return comp >= 0.9
	}
	return comp <= -0.9
}

func rot90(p Vec3, axis, dir int) Vec3 {
	d := float64(dir)
	switch axis {
	case 0:
		return Vec3{p.X, -d * p.Z, d * p.Y}
	case 1:
		return Vec3{d * p.Z, p.Y, -d * p.X}
	case 2:
		return Vec3{-d * p.Y, d * p.X, p.Z}
	}
	return p
}

func (m *Model) IsSolved() bool {
	type key struct {
		axis int
		sign int
	}
	faceColor := map[key]int{}
	for _, s := range m.Stickers {
		k, ok := faceOf(s.Pos)
		if !ok {
			continue
		}
		if c, seen := faceColor[k]; seen {
			if c != s.Color {
				return false
			}
		} else {
			faceColor[k] = s.Color
		}
	}
	return true
}

func faceOf(p Vec3) (struct {
	axis int
	sign int
}, bool) {
	type key = struct {
		axis int
		sign int
	}
	switch {
	case p.X >= 1.4:
		return key{0, 1}, true
	case p.X <= -1.4:
		return key{0, -1}, true
	case p.Y >= 1.4:
		return key{1, 1}, true
	case p.Y <= -1.4:
		return key{1, -1}, true
	case p.Z >= 1.4:
		return key{2, 1}, true
	case p.Z <= -1.4:
		return key{2, -1}, true
	}
	return key{}, false
}
