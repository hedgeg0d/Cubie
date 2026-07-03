package main

import (
	"image"
	"image/color"
	"math"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

const padGlow = 260 * time.Millisecond

type GamepadView struct {
	widget.BaseWidget
	mu       sync.Mutex
	held     map[string]bool
	flash    map[string]time.Time
	axes     map[string]int32
	w, h     int
	minSize  fyne.Size
	wake     chan struct{}
	stopCh   chan struct{}
	stopOnce sync.Once
}

func NewGamepadView() *GamepadView {
	g := &GamepadView{
		held:    map[string]bool{},
		flash:   map[string]time.Time{},
		axes:    map[string]int32{},
		w:       420,
		h:       200,
		minSize: fyne.NewSize(380, 190),
		wake:    make(chan struct{}, 1),
		stopCh:  make(chan struct{}),
	}
	g.ExtendBaseWidget(g)
	go g.animator()
	return g
}

func (g *GamepadView) kick() {
	select {
	case g.wake <- struct{}{}:
	default:
	}
	g.Refresh()
}

func (g *GamepadView) SetButton(action string, down bool) {
	g.mu.Lock()
	g.held[action] = down
	g.flash[action] = time.Now()
	g.mu.Unlock()
	g.kick()
}

func (g *GamepadView) Flash(action string) {
	g.mu.Lock()
	g.flash[action] = time.Now()
	g.mu.Unlock()
	g.kick()
}

func (g *GamepadView) SetAxis(name string, value int32) {
	g.mu.Lock()
	g.axes[name] = value
	g.mu.Unlock()
}

func (g *GamepadView) Poke() { g.kick() }

func (g *GamepadView) stop() { g.stopOnce.Do(func() { close(g.stopCh) }) }

func (g *GamepadView) busy() bool {
	now := time.Now()
	for a, t := range g.flash {
		if !g.held[a] && now.Sub(t) < padGlow {
			return true
		}
	}
	return false
}

func (g *GamepadView) animator() {
	for {
		select {
		case <-g.stopCh:
			return
		case <-g.wake:
		}
		for {
			g.Refresh()
			g.mu.Lock()
			b := g.busy()
			g.mu.Unlock()
			if !b {
				break
			}
			select {
			case <-g.stopCh:
				return
			case <-time.After(33 * time.Millisecond):
			}
		}
	}
}

func (g *GamepadView) intensity(action string, now time.Time) float64 {
	if g.held[action] {
		return 1
	}
	t, ok := g.flash[action]
	if !ok {
		return 0
	}
	d := now.Sub(t)
	if d >= padGlow {
		return 0
	}
	return 1 - float64(d)/float64(padGlow)
}

func (g *GamepadView) CreateRenderer() fyne.WidgetRenderer {
	img := canvas.NewImageFromImage(g.render())
	img.FillMode = canvas.ImageFillContain
	img.ScaleMode = canvas.ImageScaleSmooth
	return &gamepadRenderer{g: g, img: img}
}

type gamepadRenderer struct {
	g   *GamepadView
	img *canvas.Image
}

func (r *gamepadRenderer) Layout(size fyne.Size) { r.img.Resize(size) }
func (r *gamepadRenderer) MinSize() fyne.Size    { return r.g.minSize }
func (r *gamepadRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.img}
}
func (r *gamepadRenderer) Destroy() { r.g.stop() }
func (r *gamepadRenderer) Refresh() {
	r.img.Image = r.g.render()
	canvas.Refresh(r.img)
}

var (
	padBase   = color.RGBA{0x2A, 0x2E, 0x3C, 0xFF}
	padBg     = color.RGBA{0x14, 0x16, 0x1F, 0xFF}
	padA      = color.RGBA{0x34, 0xD3, 0x99, 0xFF}
	padB      = color.RGBA{0xE7, 0x4C, 0x5B, 0xFF}
	padX      = color.RGBA{0x4F, 0x8C, 0xFF, 0xFF}
	padY      = color.RGBA{0xF5, 0xC8, 0x3C, 0xFF}
	padAccent = color.RGBA{0x7C, 0x5C, 0xFF, 0xFF}
	padKnob   = color.RGBA{0x22, 0xD3, 0xEE, 0xFF}
)

func (g *GamepadView) render() image.Image {
	const ss = 2
	const vw, vh = 420.0, 200.0
	g.mu.Lock()
	W, H := g.w*ss, g.h*ss
	now := time.Now()
	inten := func(a string) float64 { return g.intensity(a, now) }
	axis := func(n string) float64 { return float64(g.axes[n]) / 32767 }
	g.mu.Unlock()

	img := image.NewRGBA(image.Rect(0, 0, W, H))
	for i := range img.Pix {
		img.Pix[i] = 0
	}
	sx := float64(W) / vw
	sy := float64(H) / vh
	px := func(x float64) float64 { return x * sx }
	py := func(y float64) float64 { return y * sy }

	lit := func(a string, on color.RGBA) color.RGBA { return mixCol(padBase, on, inten(a)) }

	btn := func(a string, cx, cy, r float64, on color.RGBA) {
		drawDisc(img, px(cx), py(cy), r*sx, lit(a, on))
	}
	rrect := func(a string, x0, y0, x1, y1 float64, on color.RGBA) {
		fillRoundRect(img, image.Rect(int(px(x0)), int(py(y0)), int(px(x1)), int(py(y1))), int(4*sx), lit(a, on))
	}

	rrect("LT", 36, 10, 96, 24, padAccent)
	rrect("RT", 324, 10, 384, 24, padAccent)
	rrect("LB", 36, 30, 96, 46, padAccent)
	rrect("RB", 324, 30, 384, 46, padAccent)

	drawStick(img, px(78), py(96), 30*sx, axis("Left X"), axis("Left Y"))
	drawStick(img, px(258), py(150), 30*sx, axis("Right X"), axis("Right Y"))

	dcx, dcy := 128.0, 150.0
	rrect("DPad Up", dcx-8, dcy-28, dcx+8, dcy-8, padAccent)
	rrect("DPad Down", dcx-8, dcy+8, dcx+8, dcy+28, padAccent)
	rrect("DPad Left", dcx-28, dcy-8, dcx-8, dcy+8, padAccent)
	rrect("DPad Right", dcx+8, dcy-8, dcx+28, dcy+8, padAccent)
	fillRoundRect(img, image.Rect(int(px(dcx-8)), int(py(dcy-8)), int(px(dcx+8)), int(py(dcy+8))), 0, padBase)

	fcx, fcy := 342.0, 96.0
	btn("A", fcx, fcy+28, 15, padA)
	btn("B", fcx+28, fcy, 15, padB)
	btn("X", fcx-28, fcy, 15, padX)
	btn("Y", fcx, fcy-28, 15, padY)

	btn("Select", 186, 96, 8, padAccent)
	btn("Start", 234, 96, 8, padAccent)

	return img
}

func drawStick(img *image.RGBA, cx, cy, r, ax, ay float64) {
	drawDisc(img, cx, cy, r, padBg)
	drawRing(img, cx, cy, r, padBase)
	off := r * 0.5
	kx := cx + ax*off
	ky := cy + ay*off
	drawDisc(img, kx, ky, r*0.55, padKnob)
}

func drawDisc(img *image.RGBA, cx, cy, r float64, col color.RGBA) {
	if r < 0.5 {
		return
	}
	minx := int(math.Floor(cx - r - 1))
	maxx := int(math.Ceil(cx + r + 1))
	miny := int(math.Floor(cy - r - 1))
	maxy := int(math.Ceil(cy + r + 1))
	for y := miny; y <= maxy; y++ {
		for x := minx; x <= maxx; x++ {
			d := math.Hypot(float64(x)-cx, float64(y)-cy)
			cov := r + 0.5 - d
			if cov <= 0 {
				continue
			}
			if cov > 1 {
				cov = 1
			}
			blendPix(img, x, y, col, cov)
		}
	}
}

func drawRing(img *image.RGBA, cx, cy, r float64, col color.RGBA) {
	inner := r - 2
	minx := int(math.Floor(cx - r - 1))
	maxx := int(math.Ceil(cx + r + 1))
	miny := int(math.Floor(cy - r - 1))
	maxy := int(math.Ceil(cy + r + 1))
	for y := miny; y <= maxy; y++ {
		for x := minx; x <= maxx; x++ {
			d := math.Hypot(float64(x)-cx, float64(y)-cy)
			cov := math.Min(r+0.5-d, d-inner+0.5)
			if cov <= 0 {
				continue
			}
			if cov > 1 {
				cov = 1
			}
			blendPix(img, x, y, col, cov)
		}
	}
}

func blendPix(img *image.RGBA, x, y int, col color.RGBA, alpha float64) {
	if x < 0 || y < 0 || x >= img.Bounds().Dx() || y >= img.Bounds().Dy() {
		return
	}
	bg := img.RGBAAt(x, y)
	inv := 1 - alpha
	img.SetRGBA(x, y, color.RGBA{
		R: uint8(float64(col.R)*alpha + float64(bg.R)*inv),
		G: uint8(float64(col.G)*alpha + float64(bg.G)*inv),
		B: uint8(float64(col.B)*alpha + float64(bg.B)*inv),
		A: 0xFF,
	})
}

func mixCol(a, b color.RGBA, t float64) color.RGBA {
	if t <= 0 {
		return a
	}
	if t >= 1 {
		return b
	}
	return color.RGBA{
		R: uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		G: uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		B: uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t),
		A: 0xFF,
	}
}
