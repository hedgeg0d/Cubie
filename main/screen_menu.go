package main

import (
	"context"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func (a *App) showMenu() {
	a.switchScreen(fyne.NewSize(400, 350), func(ctx context.Context) fyne.CanvasObject {
		batteryLabel := widget.NewLabel("Battery: --%")

		go a.pollBattery(ctx, batteryLabel)

		return container.NewVBox(
			widget.NewLabel(a.model),
			batteryLabel,
			widget.NewSeparator(),
			widget.NewButton("Controller", a.showController),
			widget.NewButton("Timer", a.showTimer),
			widget.NewButton("Blind Trainer", a.showBlind),
			widget.NewSeparator(),
			widget.NewButton("Disconnect", func() {
				a.disconnect()
				a.showConnect()
			}),
		)
	})
}

func (a *App) pollBattery(ctx context.Context, label *widget.Label) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		a.cube.UpdatePowerInfo()
		label.SetText("Battery: " + strconv.Itoa(a.cube.Power) + "%")
		label.Refresh()
		select {
		case <-ctx.Done():
			return
		case <-time.After(60 * time.Second):
		}
	}
}
