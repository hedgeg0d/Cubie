package main

import (
	"context"

	"cubie/cubestate"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func (a *App) showCube3D() {
	a.switchScreen(fyne.NewSize(500, 560), func(ctx context.Context) fyne.CanvasObject {
		view := NewCubeView(cubestate.NewSolved())

		a.cube.OnMove = func(move string) {
			view.ApplyMove(move)
		}

		hint := widget.NewLabel("Drag to rotate. Turn the cube to update.")

		sync := widget.NewButton("Cube is solved (sync)", func() {
			view.SetModel(cubestate.NewSolved())
		})

		controls := container.NewHBox(sync, widget.NewButton("Back", a.showMenu))
		return container.NewBorder(hint, controls, nil, nil, view)
	})
}
