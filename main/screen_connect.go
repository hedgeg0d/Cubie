package main

import (
	"context"

	"cubie/cube"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func (a *App) showConnect() {
	a.switchScreen(fyne.NewSize(350, 300), func(context.Context) fyne.CanvasObject {
		macEntry := widget.NewEntry()
		macEntry.SetPlaceHolder("MAC address")
		modelRadio := widget.NewRadioGroup([]string{"Weilong V10 AI"}, func(string) {})
		errorLabel := widget.NewLabel("")

		if config, err := loadConfig(); err == nil {
			macEntry.SetText(config.MACAddress)
			modelRadio.SetSelected(config.Model)
		}

		connectButton := widget.NewButton("Connect", func() {
			var t cube.CubeType
			if modelRadio.Selected == "Weilong V10 AI" {
				t = cube.WeilongV10AI
			}
			c := cube.New(t)
			if err := c.FindAndConnect(macEntry.Text); err != nil {
				errorLabel.SetText(err.Error())
				return
			}
			saveConfig(Config{MACAddress: macEntry.Text, Model: modelRadio.Selected})
			a.cube = c
			a.model = modelRadio.Selected
			c.GreetCube()
			a.showMenu()
		})

		return container.NewVBox(macEntry, modelRadio, connectButton, errorLabel)
	})
}
