package main

import (
	"image"
	"image/color"
	"math"
	"sort"
	"sync"

	"cubie/cube"
	"cubie/cubestate"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

var stickerColors = map[int]color.RGBA{
	cubestate.ColorU: {245, 245, 245, 255},
	cubestate.ColorD: {255, 213, 0, 255},
	cubestate.ColorR: {200, 0, 0, 255},
	cubestate.ColorL: {255, 120, 0, 255},
	cubestate.ColorF: {0, 160, 60, 255},
	cubestate.ColorB: {0, 80, 210, 255},
}

type CubeView struct {
	widget.BaseWidget
	mu          sync.Mutex
	model       *cubestate.Model
	yaw         float64
	pitch       float64
	interactive bool
	minSize     fyne.Size
	renderPx    int
	followGyro  bool
	gyroQuat    cube.Quaternion
}

func NewCubeView(model *cubestate.Model) *CubeView {
	c := &CubeView{model: model, yaw: 0.7, pitch: 0.5, interactive: true, minSize: fyne.NewSize(300, 300), renderPx: 400}
	c.ExtendBaseWidget(c)
	return c
}

func (c *CubeView) SetMinSize(size fyne.Size) {
	c.mu.Lock()
	c.minSize = size
	c.mu.Unlock()
	c.Refresh()
}

func (c *CubeView) DisableInteraction() {
	c.mu.Lock()
	c.interactive = false
	c.mu.Unlock()
}

func (c *CubeView) SetModel(model *cubestate.Model) {
	c.mu.Lock()
	c.model = model
	c.mu.Unlock()
	c.Refresh()
}

func (c *CubeView) SetFollowGyro(b bool) {
	c.mu.Lock()
	c.followGyro = b
	c.mu.Unlock()
	c.Refresh()
}

func (c *CubeView) SetGyro(q cube.Quaternion) {
	c.mu.Lock()
	c.gyroQuat = q
	follow := c.followGyro
	c.mu.Unlock()
	if follow {
		c.Refresh()
	}
}

func (c *CubeView) ApplyMove(move string) {
	c.mu.Lock()
	if c.model != nil {
		c.model.Apply(move)
	}
	c.mu.Unlock()
	c.Refresh()
}

func (c *CubeView) Dragged(e *fyne.DragEvent) {
	c.mu.Lock()
	if !c.interactive {
		c.mu.Unlock()
		return
	}
	c.yaw += float64(e.Dragged.DX) * 0.01
	c.pitch += float64(e.Dragged.DY) * 0.01
	if c.pitch > 1.4 {
		c.pitch = 1.4
	}
	if c.pitch < -1.4 {
		c.pitch = -1.4
	}
	c.mu.Unlock()
	c.Refresh()
}

func (c *CubeView) DragEnd() {}

func (c *CubeView) CreateRenderer() fyne.WidgetRenderer {
	img := canvas.NewImageFromImage(c.render())
	img.FillMode = canvas.ImageFillContain
	img.ScaleMode = canvas.ImageScaleFastest
	return &cubeViewRenderer{c: c, img: img}
}

type cubeViewRenderer struct {
	c   *CubeView
	img *canvas.Image
}

func (r *cubeViewRenderer) Layout(s fyne.Size) {
	r.img.Resize(s)
	px := int(s.Width)
	if h := int(s.Height); h < px {
		px = h
	}
	if px < 16 {
		px = 16
	}
	r.c.mu.Lock()
	changed := r.c.renderPx != px
	r.c.renderPx = px
	r.c.mu.Unlock()
	if changed {
		r.img.Image = r.c.render()
		canvas.Refresh(r.img)
	}
}
func (r *cubeViewRenderer) MinSize() fyne.Size {
	r.c.mu.Lock()
	size := r.c.minSize
	r.c.mu.Unlock()
	return size
}
func (r *cubeViewRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.img} }
func (r *cubeViewRenderer) Destroy()                     {}

func (r *cubeViewRenderer) Refresh() {
	r.img.Image = r.c.render()
	canvas.Refresh(r.img)
}

func (c *CubeView) render() image.Image {
	c.mu.Lock()
	model := c.model
	yaw, pitch := c.yaw, c.pitch
	size := c.renderPx
	follow := c.followGyro
	gq := c.gyroQuat
	c.mu.Unlock()
	if size < 16 {
		size = 16
	}

	rot := func(v cubestate.Vec3) cubestate.Vec3 {
		if follow {
			x, y, z := rotateByQuat(gq, v.X, v.Y, v.Z)
			v = cubestate.Vec3{X: x, Y: y, Z: z}
		}
		return viewRotate(v, yaw, pitch)
	}

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for x := 0; x < size; x++ {
		for y := 0; y < size; y++ {
			d := float64(x+y) / float64(size*2)
			img.SetRGBA(x, y, color.RGBA{
				uint8(16 + 16*d),
				uint8(19 + 18*d),
				uint8(30 + 28*d),
				255,
			})
		}
	}
	if model == nil {
		return img
	}

	scale := float64(size) * 0.15
	center := float64(size) / 2

	type quad struct {
		pts   [4][2]float64
		depth float64
		col   color.RGBA
	}
	var quads []quad

	for _, st := range model.Stickers {
		n := stickerNormal(st.Pos)
		nv := rot(n)
		if nv.Z <= 0 {
			continue
		}
		t1, t2 := tangents(n)
		const half = 0.44
		corners := [4]cubestate.Vec3{
			addV(st.Pos, addV(scaleV(t1, half), scaleV(t2, half))),
			addV(st.Pos, addV(scaleV(t1, -half), scaleV(t2, half))),
			addV(st.Pos, addV(scaleV(t1, -half), scaleV(t2, -half))),
			addV(st.Pos, addV(scaleV(t1, half), scaleV(t2, -half))),
		}
		var pts [4][2]float64
		var dsum float64
		for k, corner := range corners {
			v := rot(corner)
			pts[k] = [2]float64{center + scale*v.X, center - scale*v.Y}
			dsum += v.Z
		}
		quads = append(quads, quad{pts, dsum / 4, stickerColors[st.Color]})
	}

	sort.Slice(quads, func(i, j int) bool { return quads[i].depth < quads[j].depth })
	for _, q := range quads {
		fillQuad(img, q.pts, q.col, size)
		strokeQuad(img, q.pts, color.RGBA{15, 15, 15, 255}, size)
	}
	return img
}

func viewRotate(p cubestate.Vec3, yaw, pitch float64) cubestate.Vec3 {
	sy, cy := math.Sin(yaw), math.Cos(yaw)
	x := cy*p.X + sy*p.Z
	z := -sy*p.X + cy*p.Z
	y := p.Y
	sp, cp := math.Sin(pitch), math.Cos(pitch)
	y2 := cp*y - sp*z
	z2 := sp*y + cp*z
	return cubestate.Vec3{X: x, Y: y2, Z: z2}
}

func stickerNormal(p cubestate.Vec3) cubestate.Vec3 {
	switch {
	case p.X >= 1.4:
		return cubestate.Vec3{X: 1}
	case p.X <= -1.4:
		return cubestate.Vec3{X: -1}
	case p.Y >= 1.4:
		return cubestate.Vec3{Y: 1}
	case p.Y <= -1.4:
		return cubestate.Vec3{Y: -1}
	case p.Z >= 1.4:
		return cubestate.Vec3{Z: 1}
	default:
		return cubestate.Vec3{Z: -1}
	}
}

func tangents(n cubestate.Vec3) (cubestate.Vec3, cubestate.Vec3) {
	if n.X != 0 {
		return cubestate.Vec3{Y: 1}, cubestate.Vec3{Z: 1}
	}
	if n.Y != 0 {
		return cubestate.Vec3{X: 1}, cubestate.Vec3{Z: 1}
	}
	return cubestate.Vec3{X: 1}, cubestate.Vec3{Y: 1}
}

func addV(a, b cubestate.Vec3) cubestate.Vec3 {
	return cubestate.Vec3{X: a.X + b.X, Y: a.Y + b.Y, Z: a.Z + b.Z}
}

func scaleV(a cubestate.Vec3, s float64) cubestate.Vec3 {
	return cubestate.Vec3{X: a.X * s, Y: a.Y * s, Z: a.Z * s}
}

func fillQuad(img *image.RGBA, pts [4][2]float64, col color.RGBA, size int) {
	minX, minY := pts[0][0], pts[0][1]
	maxX, maxY := pts[0][0], pts[0][1]
	for _, p := range pts {
		minX = math.Min(minX, p[0])
		maxX = math.Max(maxX, p[0])
		minY = math.Min(minY, p[1])
		maxY = math.Max(maxY, p[1])
	}
	x0, x1 := int(math.Floor(minX)), int(math.Ceil(maxX))
	y0, y1 := int(math.Floor(minY)), int(math.Ceil(maxY))
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			if x < 0 || y < 0 || x >= size || y >= size {
				continue
			}
			if pointInQuad(float64(x)+0.5, float64(y)+0.5, pts) {
				img.SetRGBA(x, y, col)
			}
		}
	}
}

func pointInQuad(px, py float64, pts [4][2]float64) bool {
	sign := 0
	for i := 0; i < 4; i++ {
		a := pts[i]
		b := pts[(i+1)%4]
		cross := (b[0]-a[0])*(py-a[1]) - (b[1]-a[1])*(px-a[0])
		if cross > 0 {
			if sign < 0 {
				return false
			}
			sign = 1
		} else if cross < 0 {
			if sign > 0 {
				return false
			}
			sign = -1
		}
	}
	return true
}

func strokeQuad(img *image.RGBA, pts [4][2]float64, col color.RGBA, size int) {
	for i := 0; i < 4; i++ {
		a := pts[i]
		b := pts[(i+1)%4]
		drawLine(img, int(a[0]), int(a[1]), int(b[0]), int(b[1]), col, size)
	}
}

func drawLine(img *image.RGBA, x0, y0, x1, y1 int, col color.RGBA, size int) {
	dx := abs(x1 - x0)
	dy := -abs(y1 - y0)
	sx := 1
	if x0 >= x1 {
		sx = -1
	}
	sy := 1
	if y0 >= y1 {
		sy = -1
	}
	err := dx + dy
	for {
		if x0 >= 0 && y0 >= 0 && x0 < size && y0 < size {
			img.SetRGBA(x0, y0, col)
		}
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
