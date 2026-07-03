package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

func heading(text string, size float32) *canvas.Text {
	t := canvas.NewText(text, fgColor)
	t.TextSize = size
	t.TextStyle = fyne.TextStyle{Bold: true}
	return t
}

func caption(text string) *canvas.Text {
	t := canvas.NewText(text, mutedColor)
	t.TextSize = 13
	return t
}

func card(content fyne.CanvasObject) fyne.CanvasObject {
	bg := canvas.NewRectangle(surfaceColor)
	bg.CornerRadius = 18
	bg.StrokeColor = borderColor
	bg.StrokeWidth = 1
	return container.NewStack(bg, container.NewPadded(container.NewPadded(content)))
}

func newMovesStrip() (fyne.CanvasObject, func([]string)) {
	const n = 5
	texts := make([]*canvas.Text, n)
	bgs := make([]*canvas.Rectangle, n)
	cells := make([]fyne.CanvasObject, n)
	for i := range texts {
		bg := canvas.NewRectangle(color.RGBA{0, 0, 0, 0})
		bg.CornerRadius = 8
		bgs[i] = bg
		txt := canvas.NewText("", mutedColor)
		txt.TextSize = 24
		txt.TextStyle = fyne.TextStyle{Bold: true}
		txt.Alignment = fyne.TextAlignCenter
		texts[i] = txt
		cells[i] = container.NewGridWrap(fyne.NewSize(50, 46), container.NewStack(bg, container.NewCenter(txt)))
	}
	strip := container.NewHBox(cells...)
	update := func(list []string) {
		for i := 0; i < n; i++ {
			val := ""
			if i < len(list) {
				val = list[i]
			}
			texts[i].Text = val
			switch {
			case val == "":
				bgs[i].FillColor = color.RGBA{0, 0, 0, 0}
			case i == n-1:
				bgs[i].FillColor = surfaceHover
				texts[i].Color = accentColor
			default:
				bgs[i].FillColor = surfaceHover
				texts[i].Color = fgColor
			}
			bgs[i].Refresh()
			texts[i].Refresh()
		}
	}
	return strip, update
}

func pill(text *canvas.Text, accent color.Color) fyne.CanvasObject {
	bg := canvas.NewRectangle(surfaceColor)
	bg.CornerRadius = 14
	bg.StrokeColor = accent
	bg.StrokeWidth = 1.5
	return container.NewStack(bg, container.NewPadded(text))
}

type ModeTile struct {
	widget.BaseWidget
	title    string
	subtitle string
	accent   color.Color
	onTap    func()
	hovered  bool
}

func NewModeTile(title, subtitle string, accent color.Color, onTap func()) *ModeTile {
	t := &ModeTile{title: title, subtitle: subtitle, accent: accent, onTap: onTap}
	t.ExtendBaseWidget(t)
	return t
}

func (t *ModeTile) Tapped(_ *fyne.PointEvent) {
	if t.onTap != nil {
		t.onTap()
	}
}

func (t *ModeTile) MouseIn(_ *desktop.MouseEvent) {
	t.hovered = true
	t.Refresh()
}

func (t *ModeTile) MouseMoved(_ *desktop.MouseEvent) {}

func (t *ModeTile) MouseOut() {
	t.hovered = false
	t.Refresh()
}

func (t *ModeTile) Cursor() desktop.Cursor { return desktop.PointerCursor }

func (t *ModeTile) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(surfaceColor)
	bg.CornerRadius = 16
	bg.StrokeColor = borderColor
	bg.StrokeWidth = 1

	dot := canvas.NewCircle(t.accent)

	title := canvas.NewText(t.title, fgColor)
	title.TextSize = 19
	title.TextStyle = fyne.TextStyle{Bold: true}

	sub := canvas.NewText(t.subtitle, mutedColor)
	sub.TextSize = 13

	r := &modeTileRenderer{t: t, bg: bg, dot: dot, title: title, sub: sub}
	r.objects = []fyne.CanvasObject{bg, dot, title, sub}
	return r
}

type modeTileRenderer struct {
	t       *ModeTile
	bg      *canvas.Rectangle
	dot     *canvas.Circle
	title   *canvas.Text
	sub     *canvas.Text
	objects []fyne.CanvasObject
}

func (r *modeTileRenderer) Layout(s fyne.Size) {
	r.bg.Resize(s)
	r.bg.Move(fyne.NewPos(0, 0))
	r.dot.Resize(fyne.NewSize(14, 14))
	r.dot.Move(fyne.NewPos(24, 28))
	r.title.Move(fyne.NewPos(48, 20))
	r.sub.Move(fyne.NewPos(24, 54))
}

func (r *modeTileRenderer) MinSize() fyne.Size { return fyne.NewSize(240, 96) }

func (r *modeTileRenderer) Objects() []fyne.CanvasObject { return r.objects }

func (r *modeTileRenderer) Destroy() {}

func (r *modeTileRenderer) Refresh() {
	if r.t.hovered {
		r.bg.FillColor = surfaceHover
		r.bg.StrokeColor = r.t.accent
		r.bg.StrokeWidth = 1.5
	} else {
		r.bg.FillColor = surfaceColor
		r.bg.StrokeColor = borderColor
		r.bg.StrokeWidth = 1
	}
	r.bg.Refresh()
	r.dot.Refresh()
	r.title.Refresh()
	r.sub.Refresh()
	canvas.Refresh(r.t)
}
