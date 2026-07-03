package main

import (
	"context"
	"strconv"
	"sync/atomic"
	"time"

	"cubie/controller"
	"cubie/cube"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

var controllerMoves = []string{"R", "R'", "L", "L'", "U", "U'", "D", "D'", "F", "F'", "B", "B'"}

const controllerSettingsFile = "controller.json"
const actionNone = "(none)"

var gyroSources = []string{"Pitch", "Roll", "Yaw"}
var tiltDirections = []string{"+", "-"}
var tiltModes = []string{"hold", "tap"}

type GyroTiltBinding struct {
	Source    string  `json:"source"`
	Direction string  `json:"direction"`
	Action    string  `json:"action"`
	Mode      string  `json:"mode"`
	Threshold float64 `json:"threshold"`
}

type GyroAxisBinding struct {
	Source   string  `json:"source"`
	Target   string  `json:"target"`
	Deadzone float64 `json:"deadzone"`
	Range    float64 `json:"range"`
	Invert   bool    `json:"invert"`
}

type ControllerSettings struct {
	Bindings      map[string]string `json:"bindings"`
	HoldMs        int               `json:"hold_ms"`
	ReleaseFactor float64           `json:"release_factor"`
	Smoothing     float64           `json:"smoothing"`
	TiltBindings  []GyroTiltBinding `json:"tilt_bindings"`
	AxisBindings  []GyroAxisBinding `json:"axis_bindings"`
	Neutral       *cube.Quaternion  `json:"neutral,omitempty"`
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
	if s.ReleaseFactor <= 0 {
		s.ReleaseFactor = 0.7
	}
	if s.Smoothing <= 0 {
		s.Smoothing = 0.35
	}
	return s
}

func (s ControllerSettings) snapshot() ControllerSettings {
	c := s
	c.Bindings = make(map[string]string, len(s.Bindings))
	for k, v := range s.Bindings {
		c.Bindings[k] = v
	}
	c.TiltBindings = append([]GyroTiltBinding(nil), s.TiltBindings...)
	c.AxisBindings = append([]GyroAxisBinding(nil), s.AxisBindings...)
	if s.Neutral != nil {
		n := *s.Neutral
		c.Neutral = &n
	}
	return c
}

func (a *App) showController() {
	a.switchScreen(fyne.NewSize(760, 720), func(ctx context.Context) fyne.CanvasObject {
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
		var cfg atomic.Pointer[ControllerSettings]
		publish := func() {
			snap := settings.snapshot()
			cfg.Store(&snap)
		}
		publish()

		a.cube.OnMove = func(move string) {
			s := cfg.Load()
			action, ok := s.Bindings[move]
			if ok && action != "" && action != actionNone {
				go c.Press(action, time.Duration(s.HoldMs))
			}
		}

		sphere := NewGyroSphere()
		preview := &gyroPreview{}

		tabs := container.NewAppTabs(
			container.NewTabItem("Buttons", buildButtonsTab(&settings, publish)),
			container.NewTabItem("Gyro tilts", buildTiltTab(&settings, publish)),
			container.NewTabItem("Gyro axes", buildAxisTab(&settings, publish)),
			container.NewTabItem("Live", buildLiveTab(sphere, preview, &settings, publish)),
		)

		a.runGyroController(ctx, c, &cfg, sphere, preview)

		save := widget.NewButton("Save", func() {
			writeJSON(controllerSettingsFile, settings)
		})
		save.Importance = widget.HighImportance

		return container.NewBorder(
			container.NewPadded(heading("Controller builder", 24)),
			container.NewPadded(container.NewHBox(save, widget.NewButton("Back", a.showMenu))),
			nil, nil,
			tabs,
		)
	})
}

func buildButtonsTab(settings *ControllerSettings, publish func()) fyne.CanvasObject {
	actionOptions := append([]string{actionNone}, controller.Actions...)
	formItems := make([]*widget.FormItem, 0, len(controllerMoves))
	for _, move := range controllerMoves {
		move := move
		sel := widget.NewSelect(actionOptions, func(action string) {
			settings.Bindings[move] = action
			publish()
		})
		if cur, ok := settings.Bindings[move]; ok && cur != "" {
			sel.SetSelected(cur)
		} else {
			sel.SetSelected(actionNone)
		}
		formItems = append(formItems, widget.NewFormItem(move, sel))
	}
	form := widget.NewForm(formItems...)

	holdLabel := caption("Hold (ms): " + strconv.Itoa(settings.HoldMs))
	holdSlider := widget.NewSlider(40, 200)
	holdSlider.Step = 1
	holdSlider.Value = float64(settings.HoldMs)
	holdSlider.OnChanged = func(v float64) {
		settings.HoldMs = int(v)
		holdLabel.Text = "Hold (ms): " + strconv.Itoa(settings.HoldMs)
		holdLabel.Refresh()
		publish()
	}

	return container.NewVScroll(container.NewVBox(
		card(container.NewVBox(heading("Tap hold time", 16), holdLabel, holdSlider)),
		card(form),
	))
}
