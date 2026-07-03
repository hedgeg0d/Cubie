package main

import (
	"image"
	"image/color"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

var (
	timerFontOnce sync.Once
	timerFont     *opentype.Font
)

func loadTimerFont() {
	timerFontOnce.Do(func() {
		res := theme.DefaultTheme().Font(fyne.TextStyle{Bold: true})
		if f, err := opentype.Parse(res.Content()); err == nil {
			timerFont = f
		}
	})
}

const rollDuration = 220 * time.Millisecond

type digitCell struct {
	cur, prev rune
	start     time.Time
	animating bool
}

type RollingTimer struct {
	widget.BaseWidget
	mu       sync.Mutex
	cells    []digitCell
	col       color.RGBA
	w, h      int
	minSize   fyne.Size
	atlas     map[rune]*image.RGBA
	atlasSize int
	atlasCol  color.RGBA
	wake      chan struct{}
	stopCh    chan struct{}
	stopOnce  sync.Once
}

const glyphSet = "0123456789:."

func (r *RollingTimer) ensureAtlas(px int, col color.RGBA) {
	if r.atlas != nil && r.atlasSize == px && r.atlasCol == col {
		return
	}
	if timerFont == nil || px < 1 {
		return
	}
	face, err := opentype.NewFace(timerFont, &opentype.FaceOptions{Size: float64(px), DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return
	}
	m := face.Metrics()
	ascent := m.Ascent.Ceil()
	h := ascent + m.Descent.Ceil()
	atlas := make(map[rune]*image.RGBA, len(glyphSet))
	for _, ch := range glyphSet {
		adv := font.MeasureString(face, string(ch)).Ceil()
		if adv < 1 {
			adv = 1
		}
		gi := image.NewRGBA(image.Rect(0, 0, adv, h))
		d := &font.Drawer{Dst: gi, Src: image.NewUniform(col), Face: face, Dot: fixed.P(0, ascent)}
		d.DrawString(string(ch))
		atlas[ch] = gi
	}
	r.atlas = atlas
	r.atlasSize = px
	r.atlasCol = col
}

func NewRollingTimer() *RollingTimer {
	loadTimerFont()
	r := &RollingTimer{
		col:     color.RGBA{0xEC, 0xEC, 0xF2, 0xFF},
		minSize: fyne.NewSize(360, 150),
		w:       360,
		h:       150,
		wake:    make(chan struct{}, 1),
		stopCh:  make(chan struct{}),
	}
	r.setCells("0.00")
	r.ExtendBaseWidget(r)
	go r.animator()
	return r
}

func (r *RollingTimer) setCells(s string) {
	r.cells = make([]digitCell, len(s))
	for i, ch := range s {
		r.cells[i] = digitCell{cur: ch, prev: ch}
	}
}

func (r *RollingTimer) SetText(s string) {
	r.mu.Lock()
	if len(s) != len(r.cells) {
		old := len(r.cells)
		r.setCells(s)
		if len(s) > old {
			now := time.Now()
			for i := 0; i < len(s)-old; i++ {
				r.cells[i].prev = ' '
				r.cells[i].animating = true
				r.cells[i].start = now
			}
		}
	} else {
		now := time.Now()
		for i, ch := range s {
			if ch != r.cells[i].cur {
				r.cells[i].prev = r.cells[i].cur
				r.cells[i].cur = ch
				r.cells[i].start = now
				r.cells[i].animating = true
			}
		}
	}
	r.mu.Unlock()
	select {
	case r.wake <- struct{}{}:
	default:
	}
	r.Refresh()
}

func (r *RollingTimer) SetColor(c color.RGBA) {
	r.mu.Lock()
	r.col = c
	r.mu.Unlock()
	r.Refresh()
}

func (r *RollingTimer) stop() { r.stopOnce.Do(func() { close(r.stopCh) }) }

func (r *RollingTimer) animator() {
	for {
		select {
		case <-r.stopCh:
			return
		case <-r.wake:
		}
		for {
			r.mu.Lock()
			any := false
			now := time.Now()
			for i := range r.cells {
				if r.cells[i].animating {
					if now.Sub(r.cells[i].start) >= rollDuration {
						r.cells[i].animating = false
					} else {
						any = true
					}
				}
			}
			r.mu.Unlock()
			r.Refresh()
			if !any {
				break
			}
			select {
			case <-r.stopCh:
				return
			case <-time.After(16 * time.Millisecond):
			}
		}
	}
}

func (r *RollingTimer) CreateRenderer() fyne.WidgetRenderer {
	img := canvas.NewImageFromImage(r.render())
	img.FillMode = canvas.ImageFillStretch
	img.ScaleMode = canvas.ImageScaleSmooth
	return &rollingRenderer{r: r, img: img}
}

type rollingRenderer struct {
	r   *RollingTimer
	img *canvas.Image
}

func (rr *rollingRenderer) Layout(size fyne.Size) {
	rr.img.Resize(size)
	w, h := int(size.Width), int(size.Height)
	if w < 8 {
		w = 8
	}
	if h < 8 {
		h = 8
	}
	rr.r.mu.Lock()
	changed := rr.r.w != w || rr.r.h != h
	rr.r.w, rr.r.h = w, h
	rr.r.mu.Unlock()
	if changed {
		rr.img.Image = rr.r.render()
		canvas.Refresh(rr.img)
	}
}

func (rr *rollingRenderer) MinSize() fyne.Size {
	rr.r.mu.Lock()
	defer rr.r.mu.Unlock()
	return rr.r.minSize
}

func (rr *rollingRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{rr.img} }
func (rr *rollingRenderer) Destroy()                     { rr.r.stop() }
func (rr *rollingRenderer) Refresh() {
	rr.img.Image = rr.r.render()
	canvas.Refresh(rr.img)
}

func (r *RollingTimer) render() image.Image {
	const ss = 3
	r.mu.Lock()
	cells := make([]digitCell, len(r.cells))
	copy(cells, r.cells)
	col := r.col
	W, H := r.w*ss, r.h*ss

	cellH := float64(H) * 0.9
	digitW := cellH * 0.64
	sepW := cellH * 0.32
	gap := digitW * 0.16

	total := 0.0
	for _, c := range cells {
		if c.cur == '.' || c.cur == ':' {
			total += sepW + gap
		} else {
			total += digitW + gap
		}
	}
	total -= gap
	if avail := float64(W) * 0.96; total > avail {
		s := avail / total
		cellH *= s
		digitW *= s
		sepW *= s
		gap *= s
		total *= s
	}

	r.ensureAtlas(int(cellH*0.82), col)
	atlas := r.atlas
	r.mu.Unlock()

	img := image.NewRGBA(image.Rect(0, 0, W, H))
	x := (float64(W) - total) / 2
	y0 := (float64(H) - cellH) / 2

	chipBg := color.RGBA{0x24, 0x27, 0x36, 0xFF}
	now := time.Now()

	for _, c := range cells {
		if c.cur == '.' || c.cur == ':' {
			cell := image.Rect(int(x), int(y0), int(x+sepW), int(y0+cellH))
			blitGlyph(img, atlas[c.cur], cell, 0)
			x += sepW + gap
			continue
		}
		cellRect := image.Rect(int(x), int(y0), int(x+digitW), int(y0+cellH))
		fillRoundRect(img, cellRect, int(cellH*0.16), chipBg)

		if c.animating {
			p := float64(now.Sub(c.start)) / float64(rollDuration)
			if p < 0 {
				p = 0
			}
			if p > 1 {
				p = 1
			}
			e := p * p * (3 - 2*p)
			travel := cellH
			blitGlyph(img, atlas[c.prev], cellRect, -travel*e)
			blitGlyph(img, atlas[c.cur], cellRect, travel*(1-e))
		} else {
			blitGlyph(img, atlas[c.cur], cellRect, 0)
		}
		x += digitW + gap
	}
	return img
}

func blitGlyph(img *image.RGBA, gi *image.RGBA, cell image.Rectangle, yOffset float64) {
	if gi == nil {
		return
	}
	gw, gh := gi.Bounds().Dx(), gi.Bounds().Dy()
	cw, ch := cell.Dx(), cell.Dy()
	if cw <= 0 || ch <= 0 {
		return
	}
	originX := cell.Min.X + (cw-gw)/2
	originY := cell.Min.Y + (ch-gh)/2 + int(yOffset)
	iw, ih := img.Bounds().Dx(), img.Bounds().Dy()
	for gy := 0; gy < gh; gy++ {
		dy := originY + gy
		if dy < cell.Min.Y || dy >= cell.Max.Y || dy < 0 || dy >= ih {
			continue
		}
		for gx := 0; gx < gw; gx++ {
			dx := originX + gx
			if dx < cell.Min.X || dx >= cell.Max.X || dx < 0 || dx >= iw {
				continue
			}
			c := gi.RGBAAt(gx, gy)
			if c.A == 0 {
				continue
			}
			if c.A == 0xFF {
				img.SetRGBA(dx, dy, c)
				continue
			}
			bg := img.RGBAAt(dx, dy)
			inv := 1 - float64(c.A)/255
			img.SetRGBA(dx, dy, color.RGBA{
				R: uint8(float64(c.R) + float64(bg.R)*inv),
				G: uint8(float64(c.G) + float64(bg.G)*inv),
				B: uint8(float64(c.B) + float64(bg.B)*inv),
				A: 0xFF,
			})
		}
	}
}

func fillRoundRect(img *image.RGBA, rect image.Rectangle, radius int, col color.RGBA) {
	rad := radius
	if rad*2 > rect.Dx() {
		rad = rect.Dx() / 2
	}
	if rad*2 > rect.Dy() {
		rad = rect.Dy() / 2
	}
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			dx := 0
			dy := 0
			if x < rect.Min.X+rad {
				dx = rect.Min.X + rad - x
			} else if x >= rect.Max.X-rad {
				dx = x - (rect.Max.X - rad - 1)
			}
			if y < rect.Min.Y+rad {
				dy = rect.Min.Y + rad - y
			} else if y >= rect.Max.Y-rad {
				dy = y - (rect.Max.Y - rad - 1)
			}
			if dx > 0 && dy > 0 && dx*dx+dy*dy > rad*rad {
				continue
			}
			if x >= 0 && y >= 0 && x < img.Bounds().Dx() && y < img.Bounds().Dy() {
				img.SetRGBA(x, y, col)
			}
		}
	}
}
