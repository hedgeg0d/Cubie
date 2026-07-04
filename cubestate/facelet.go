package cubestate

var groupToFace = [6]string{"F", "B", "U", "D", "L", "R"}

var colorByValue [6]int

var faceStickerPos = map[string][8]Vec3{
	"F": {{-1, 1, 1.5}, {0, 1, 1.5}, {1, 1, 1.5}, {-1, 0, 1.5}, {1, 0, 1.5}, {-1, -1, 1.5}, {0, -1, 1.5}, {1, -1, 1.5}},
	"B": {{1, 1, -1.5}, {0, 1, -1.5}, {-1, 1, -1.5}, {1, 0, -1.5}, {-1, 0, -1.5}, {1, -1, -1.5}, {0, -1, -1.5}, {-1, -1, -1.5}},
	"U": {{-1, 1.5, -1}, {0, 1.5, -1}, {1, 1.5, -1}, {-1, 1.5, 0}, {1, 1.5, 0}, {-1, 1.5, 1}, {0, 1.5, 1}, {1, 1.5, 1}},
	"D": {{-1, -1.5, 1}, {0, -1.5, 1}, {1, -1.5, 1}, {-1, -1.5, 0}, {1, -1.5, 0}, {-1, -1.5, -1}, {0, -1.5, -1}, {1, -1.5, -1}},
	"L": {{-1.5, 1, -1}, {-1.5, 1, 0}, {-1.5, 1, 1}, {-1.5, 0, -1}, {-1.5, 0, 1}, {-1.5, -1, -1}, {-1.5, -1, 0}, {-1.5, -1, 1}},
	"R": {{1.5, 1, 1}, {1.5, 1, 0}, {1.5, 1, -1}, {1.5, 0, 1}, {1.5, 0, -1}, {1.5, -1, 1}, {1.5, -1, 0}, {1.5, -1, -1}},
}

func init() {
	for g, name := range groupToFace {
		colorByValue[g] = faceSpecs[name].color
	}
}

func decodeFaceValues(b []byte) [8]int {
	bits := uint32(b[0])<<16 | uint32(b[1])<<8 | uint32(b[2])
	var out [8]int
	for k := range 8 {
		out[k] = int((bits >> uint(21-3*k)) & 0x7)
	}
	return out
}

func ModelFromState(state [18]byte) *Model {
	m := &Model{}
	for _, name := range faceOrder {
		f := faceSpecs[name]
		var c Vec3
		switch f.axis {
		case 0:
			c = Vec3{1.5 * f.sign, 0, 0}
		case 1:
			c = Vec3{0, 1.5 * f.sign, 0}
		case 2:
			c = Vec3{0, 0, 1.5 * f.sign}
		}
		m.Stickers = append(m.Stickers, Sticker{Pos: c, Color: f.color})
	}
	for g, name := range groupToFace {
		vals := decodeFaceValues(state[g*3 : g*3+3])
		for k, p := range faceStickerPos[name] {
			m.Stickers = append(m.Stickers, Sticker{Pos: p, Color: colorByValue[vals[k]]})
		}
	}
	return m
}
