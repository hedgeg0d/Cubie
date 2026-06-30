package main

import (
	"context"

	"cubie/cube"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
)

type App struct {
	fyneApp fyne.App
	window  fyne.Window
	cube    *cube.Cube
	model   string

	cancel context.CancelFunc
}

func NewApp() *App {
	a := &App{fyneApp: fyneapp.New()}
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
	a.window.SetContent(build(ctx))
	a.window.Resize(size)
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
