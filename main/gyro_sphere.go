package main

import (
	"image"
	"image/color"
	"math"
	"sync"

	"cubie/cube"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

type GyroSphere struct {
	widget.BaseWidget
	mu       sync.Mutex
	q        cube.Quaternion
	minSize  fyne.Size
	renderPx int
}

func NewGyroSphere() *GyroSphere {
	g := &GyroSphere{q: cube.Quaternion{W: 1}, minSize: fyne.NewSize(170, 170), renderPx: 190}
	g.ExtendBaseWidget(g)
	return g
}

func (g *GyroSphere) SetMinSize(s fyne.Size) {
	g.mu.Lock()
	g.minSize = s
	g.mu.Unlock()
	g.Refresh()
}

func (g *GyroSphere) SetQuaternion(q cube.Quaternion) {
	g.mu.Lock()
	g.q = q
	g.mu.Unlock()
	g.Refresh()
}

func (g *GyroSphere) CreateRenderer() fyne.WidgetRenderer {
	img := canvas.NewImageFromImage(g.render())
	img.FillMode = canvas.ImageFillContain
	img.ScaleMode = canvas.ImageScaleFastest
	return &gyroSphereRenderer{g: g, img: img}
}

type gyroSphereRenderer struct {
	g   *GyroSphere
	img *canvas.Image
}

func (r *gyroSphereRenderer) Layout(s fyne.Size) {
	r.img.Resize(s)
	px := int(s.Width)
	if h := int(s.Height); h < px {
		px = h
	}
	if px < 16 {
		px = 16
	}
	r.g.mu.Lock()
	changed := r.g.renderPx != px
	r.g.renderPx = px
	r.g.mu.Unlock()
	if changed {
		r.img.Image = r.g.render()
		canvas.Refresh(r.img)
	}
}

func (r *gyroSphereRenderer) MinSize() fyne.Size {
	r.g.mu.Lock()
	defer r.g.mu.Unlock()
	return r.g.minSize
}

func (r *gyroSphereRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.img} }
func (r *gyroSphereRenderer) Destroy()                     {}

func (r *gyroSphereRenderer) Refresh() {
	r.img.Image = r.g.render()
	canvas.Refresh(r.img)
}

func quatNlerp(a, b cube.Quaternion, t float64) cube.Quaternion {
	dot := a.W*b.W + a.X*b.X + a.Y*b.Y + a.Z*b.Z
	if dot < 0 {
		b = cube.Quaternion{W: -b.W, X: -b.X, Y: -b.Y, Z: -b.Z}
	}
	r := cube.Quaternion{
		W: a.W + (b.W-a.W)*t,
		X: a.X + (b.X-a.X)*t,
		Y: a.Y + (b.Y-a.Y)*t,
		Z: a.Z + (b.Z-a.Z)*t,
	}
	n := math.Sqrt(r.W*r.W + r.X*r.X + r.Y*r.Y + r.Z*r.Z)
	if n < 1e-6 {
		return cube.Quaternion{W: 1}
	}
	return cube.Quaternion{W: r.W / n, X: r.X / n, Y: r.Y / n, Z: r.Z / n}
}

func rotateByQuat(q cube.Quaternion, vx, vy, vz float64) (float64, float64, float64) {
	tx := 2 * (q.Y*vz - q.Z*vy)
	ty := 2 * (q.Z*vx - q.X*vz)
	tz := 2 * (q.X*vy - q.Y*vx)
	rx := vx + q.W*tx + (q.Y*tz - q.Z*ty)
	ry := vy + q.W*ty + (q.Z*tx - q.X*tz)
	rz := vz + q.W*tz + (q.X*ty - q.Y*tx)
	return rx, ry, rz
}

func (g *GyroSphere) render() image.Image {
	g.mu.Lock()
	q := g.q
	size := g.renderPx
	g.mu.Unlock()
	if size < 16 {
		size = 16
	}

	n := math.Sqrt(q.W*q.W + q.X*q.X + q.Y*q.Y + q.Z*q.Z)
	if n < 1e-6 {
		q = cube.Quaternion{W: 1}
	} else {
		q = cube.Quaternion{W: q.W / n, X: q.X / n, Y: q.Y / n, Z: q.Z / n}
	}

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for x := 0; x < size; x++ {
		for y := 0; y < size; y++ {
			img.SetRGBA(x, y, color.RGBA{16, 17, 24, 255})
		}
	}

	cx := float64(size) / 2
	cy := float64(size) / 2
	radius := float64(size)*0.5 - 6

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) + 0.5 - cx
			dy := float64(y) + 0.5 - cy
			d := math.Sqrt(dx*dx + dy*dy)
			if d <= radius {
				shade := 1 - d/radius*0.7
				img.SetRGBA(x, y, color.RGBA{
					uint8(30 * shade),
					uint8(34 * shade),
					uint8(52 * shade),
					255,
				})
			}
		}
	}

	plot := func(px, py float64, z float64, base color.RGBA) {
		if z < 0 {
			base = color.RGBA{base.R / 3, base.G / 3, base.B / 3, 255}
		}
		ix, iy := int(px), int(py)
		for oy := 0; oy <= 1; oy++ {
			for ox := 0; ox <= 1; ox++ {
				x, y := ix+ox, iy+oy
				if x >= 0 && y >= 0 && x < size && y < size {
					img.SetRGBA(x, y, base)
				}
			}
		}
	}

	grid := color.RGBA{90, 150, 240, 255}
	for latDeg := -75; latDeg <= 75; latDeg += 15 {
		lat := float64(latDeg) * math.Pi / 180
		for lonDeg := 0; lonDeg < 360; lonDeg += 8 {
			lon := float64(lonDeg) * math.Pi / 180
			vx := math.Cos(lat) * math.Cos(lon)
			vy := math.Sin(lat)
			vz := math.Cos(lat) * math.Sin(lon)
			rx, ry, rz := rotateByQuat(q, vx, vy, vz)
			plot(cx+radius*rx, cy-radius*ry, rz, grid)
		}
	}
	for lonDeg := 0; lonDeg < 360; lonDeg += 30 {
		lon := float64(lonDeg) * math.Pi / 180
		for latDeg := -85; latDeg <= 85; latDeg += 6 {
			lat := float64(latDeg) * math.Pi / 180
			vx := math.Cos(lat) * math.Cos(lon)
			vy := math.Sin(lat)
			vz := math.Cos(lat) * math.Sin(lon)
			rx, ry, rz := rotateByQuat(q, vx, vy, vz)
			plot(cx+radius*rx, cy-radius*ry, rz, grid)
		}
	}

	axes := []struct {
		vx, vy, vz float64
		col        color.RGBA
	}{
		{1, 0, 0, color.RGBA{230, 70, 70, 255}},
		{0, 1, 0, color.RGBA{70, 210, 110, 255}},
		{0, 0, 1, color.RGBA{80, 130, 240, 255}},
	}
	for _, ax := range axes {
		rx, ry, rz := rotateByQuat(q, ax.vx, ax.vy, ax.vz)
		ex := cx + radius*rx
		ey := cy - radius*ry
		col := ax.col
		if rz < 0 {
			col = color.RGBA{col.R / 3, col.G / 3, col.B / 3, 255}
		}
		drawLine(img, int(cx), int(cy), int(ex), int(ey), col, size)
		for oy := -2; oy <= 2; oy++ {
			for ox := -2; ox <= 2; ox++ {
				if ox*ox+oy*oy > 6 {
					continue
				}
				x, y := int(ex)+ox, int(ey)+oy
				if x >= 0 && y >= 0 && x < size && y < size {
					img.SetRGBA(x, y, col)
				}
			}
		}
	}

	return img
}
