package main

import (
	"context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func (a *App) showBlind() {
	a.switchScreen(fyne.NewSize(600, 450), func(context.Context) fyne.CanvasObject {
		return container.NewVBox(
			widget.NewLabel("Blind Trainer mode"),
			widget.NewLabel("(coming in Phase 5)"),
			widget.NewButton("Back", a.showMenu),
		)
	})
}
