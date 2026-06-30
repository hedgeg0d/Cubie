package main

import (
	"context"
	"strconv"
	"time"

	"cubie/controller"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

var controllerMoves = []string{"R", "R'", "L", "L'", "U", "U'", "D", "D'", "F", "F'", "B", "B'"}

const controllerSettingsFile = "controller.json"
const actionNone = "(none)"

type ControllerSettings struct {
	Bindings map[string]string `json:"bindings"`
	HoldMs   int               `json:"hold_ms"`
}

func loadControllerSettings() ControllerSettings {
	s := ControllerSettings{Bindings: map[string]string{}, HoldMs: 50}
	readJSON(controllerSettingsFile, &s)
	if s.Bindings == nil {
		s.Bindings = map[string]string{}
	}
	if s.HoldMs == 0 {
		s.HoldMs = 50
	}
	return s
}

func (a *App) showController() {
	a.switchScreen(fyne.NewSize(600, 550), func(ctx context.Context) fyne.CanvasObject {
		c := &controller.Controller{}
		if err := c.Init(); err != nil {
			return container.NewVBox(
				widget.NewLabel("Controller init failed:"),
				widget.NewLabel(err.Error()),
				widget.NewLabel("Check access to /dev/uinput."),
				widget.NewButton("Back", a.showMenu),
			)
		}
		go func() {
			<-ctx.Done()
			c.Close()
		}()

		settings := loadControllerSettings()
		hold := settings.HoldMs

		a.cube.OnMove = func(move string) {
			action, ok := settings.Bindings[move]
			if ok && action != "" && action != actionNone {
				go c.Press(action, time.Duration(hold))
			}
		}

		actionOptions := append([]string{actionNone}, controller.Actions...)
		formItems := make([]*widget.FormItem, 0, len(controllerMoves))
		for _, move := range controllerMoves {
			move := move
			sel := widget.NewSelect(actionOptions, func(action string) {
				settings.Bindings[move] = action
			})
			if cur, ok := settings.Bindings[move]; ok && cur != "" {
				sel.SetSelected(cur)
			} else {
				sel.SetSelected(actionNone)
			}
			formItems = append(formItems, widget.NewFormItem(move, sel))
		}
		form := widget.NewForm(formItems...)

		holdLabel := widget.NewLabel("Hold (ms): " + strconv.Itoa(hold))
		holdSlider := widget.NewSlider(40, 200)
		holdSlider.Step = 1
		holdSlider.Value = float64(hold)
		holdSlider.OnChanged = func(v float64) {
			hold = int(v)
			holdLabel.SetText("Hold (ms): " + strconv.Itoa(hold))
		}

		saveButton := widget.NewButton("Save Bindings", func() {
			settings.HoldMs = hold
			writeJSON(controllerSettingsFile, settings)
		})

		return container.NewBorder(
			container.NewVBox(widget.NewLabel("Controller mode"), container.NewHBox(holdLabel, holdSlider)),
			container.NewHBox(saveButton, widget.NewButton("Back", a.showMenu)),
			nil, nil,
			container.NewVScroll(form),
		)
	})
}
