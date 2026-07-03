package main

import (
	"context"

	"cubie/cube"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func (a *App) showConnect() {
	a.switchScreen(fyne.NewSize(520, 500), func(context.Context) fyne.CanvasObject {
		title := heading("CUBIE", 34)
		tagline := caption("Smart cube companion")

		macEntry := widget.NewEntry()
		macEntry.SetPlaceHolder("MAC address  (AA:BB:CC:DD:EE:FF)")
		modelRadio := widget.NewRadioGroup([]string{"Weilong V10 AI"}, func(string) {})

		errorText := caption("")
		errorText.Color = errorColor
		errorText.TextSize = 13

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
				errorText.Text = err.Error()
				errorText.Refresh()
				return
			}
			errorText.Text = ""
			errorText.Refresh()
			saveConfig(Config{MACAddress: macEntry.Text, Model: modelRadio.Selected})
			a.cube = c
			a.model = modelRadio.Selected
			a.setMiniSolved()
			c.GreetCube()
			a.showMenu()
		})
		connectButton.Importance = widget.HighImportance

		form := container.NewVBox(
			container.NewCenter(title),
			container.NewCenter(tagline),
			widget.NewSeparator(),
			caption("Cube model"),
			modelRadio,
			caption("Bluetooth address"),
			macEntry,
			connectButton,
			container.NewCenter(errorText),
		)

		return container.NewCenter(container.NewGridWrap(fyne.NewSize(410, 360), card(form)))
	})
}
