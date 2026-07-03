package main

import (
	"context"
	"strconv"
	"time"

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

		var scanResults []cube.ScanResult
		resultList := widget.NewList(
			func() int { return len(scanResults) },
			func() fyne.CanvasObject { return widget.NewLabel("") },
			func(i widget.ListItemID, o fyne.CanvasObject) {
				r := scanResults[i]
				o.(*widget.Label).SetText(r.Name + "  ·  " + r.Address + "  (" + strconv.Itoa(int(r.RSSI)) + " dBm)")
			},
		)
		resultList.OnSelected = func(i widget.ListItemID) {
			if i >= 0 && i < len(scanResults) {
				macEntry.SetText(scanResults[i].Address)
			}
		}
		listBox := container.NewGridWrap(fyne.NewSize(410, 130), resultList)
		listBox.Hide()

		var scanButton *widget.Button
		scanButton = widget.NewButton("Scan for cubes", func() {
			scanButton.Disable()
			scanButton.SetText("Scanning...")
			errorText.Text = ""
			errorText.Refresh()
			go func() {
				results, err := cube.Scan(4 * time.Second)
				scanButton.Enable()
				scanButton.SetText("Scan for cubes")
				if err != nil {
					errorText.Text = err.Error()
					errorText.Refresh()
					return
				}
				scanResults = results
				resultList.UnselectAll()
				resultList.Refresh()
				if len(results) == 0 {
					errorText.Text = "No Bluetooth devices found"
					errorText.Refresh()
					listBox.Hide()
				} else {
					listBox.Show()
				}
			}()
		})

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
			scanButton,
			listBox,
			connectButton,
			container.NewCenter(errorText),
		)

		return container.NewCenter(container.NewGridWrap(fyne.NewSize(430, 500), card(form)))
	})
}
