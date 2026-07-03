package main

import (
	"context"

	"cubie/cube"
	"cubie/cubestate"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
)

type App struct {
	fyneApp   fyne.App
	window    fyne.Window
	cube      *cube.Cube
	model     string
	cubeModel *cubestate.Model
	miniCube  *CubeView
	hideMini  bool

	cancel context.CancelFunc
}

func NewApp() *App {
	a := &App{fyneApp: fyneapp.New()}
	a.fyneApp.Settings().SetTheme(cubieTheme{})
	a.window = a.fyneApp.NewWindow("Cubie")
	return a
}

func (a *App) Run() {
	a.showConnect()
	a.window.ShowAndRun()
}

func (a *App) switchScreen(size fyne.Size, build func(ctx context.Context) fyne.CanvasObject) {
	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}
	if a.cube != nil {
		a.cube.OnMove = nil
		a.cube.OnState = nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	content := build(ctx)
	if a.hideMini {
		a.miniCube = nil
		a.hideMini = false
	} else {
		content = a.withMiniCube(content)
	}
	a.window.SetContent(content)
	a.wrapCubeHandlers()
	a.window.Resize(size)
}

func (a *App) wrapCubeHandlers() {
	if a.cube == nil {
		return
	}
	onMove := a.cube.OnMove
	onState := a.cube.OnState
	a.cube.OnMove = func(move string) {
		a.applyMiniMove(move)
		if onMove != nil {
			onMove(move)
		}
	}
	a.cube.OnState = func(state [18]byte, solved bool) {
		if solved {
			a.setMiniSolved()
		}
		if onState != nil {
			onState(state, solved)
		}
	}
}

func (a *App) applyMiniMove(move string) {
	if a.cubeModel == nil {
		a.cubeModel = cubestate.NewSolved()
	}
	if a.miniCube != nil {
		a.miniCube.ApplyMove(move)
	} else {
		a.cubeModel.Apply(move)
	}
}

func (a *App) setMiniSolved() {
	a.cubeModel = cubestate.NewSolved()
	if a.miniCube != nil {
		a.miniCube.SetModel(a.cubeModel)
	}
}

func (a *App) disconnect() error {
	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}
	if a.cube == nil {
		return nil
	}
	err := a.cube.Disconnect()
	a.cube = nil
	return err
}
