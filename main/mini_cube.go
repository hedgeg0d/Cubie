package main

import (
	"image/color"

	"cubie/cubestate"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

func (a *App) withMiniCube(content fyne.CanvasObject) fyne.CanvasObject {
	if a.cube == nil {
		a.miniCube = nil
		return content
	}
	model := a.cubeModel
	if model == nil {
		model = cubestate.NewSolved()
	}
	a.miniCube = NewCubeView(model)
	a.miniCube.SetMinSize(fyne.NewSize(120, 120))

	bg := canvas.NewRectangle(color.RGBA{18, 18, 22, 210})
	bg.CornerRadius = 12
	bg.StrokeColor = color.RGBA{70, 70, 80, 255}
	bg.StrokeWidth = 1

	badge := container.NewStack(bg, container.NewPadded(a.miniCube))
	return container.NewStack(
		content,
		container.New(&miniCubeLayout{size: 128, marginX: 16, bottomBar: 70}, badge),
	)
}

type miniCubeLayout struct {
	size      float32
	marginX   float32
	bottomBar float32
}

func (l *miniCubeLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	objects[0].Resize(fyne.NewSize(l.size, l.size))
	objects[0].Move(fyne.NewPos(l.marginX, size.Height-l.size-l.bottomBar))
}

func (l *miniCubeLayout) MinSize([]fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(1, 1)
}
