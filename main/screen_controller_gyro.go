package main

import (
	"context"
	"strconv"
	"sync/atomic"
	"time"

	"cubie/controller"
	"cubie/cube"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type gyroPreview struct {
	pitch, roll, yaw *canvas.Text
	axisOut          *widget.Label
}

func degStr(v float64) string {
	return strconv.FormatFloat(v, 'f', 0, 64) + "°"
}

func labeled(name string, w fyne.CanvasObject) fyne.CanvasObject {
	return container.NewVBox(caption(name), w)
}

func eulerComponent(e Euler, source string) float64 {
	switch source {
	case "Roll":
		return e.Roll
	case "Yaw":
		return e.Yaw
	default:
		return e.Pitch
	}
}

func buildTiltTab(settings *ControllerSettings, publish func()) fyne.CanvasObject {
	actionOptions := append([]string{actionNone}, controller.Actions...)
	rows := container.NewVBox()
	var rebuild func()
	rebuild = func() {
		rows.RemoveAll()
		for i := range settings.TiltBindings {
			rows.Add(tiltRow(settings, i, actionOptions, publish, rebuild))
		}
		rows.Refresh()
	}
	add := widget.NewButton("Add tilt binding", func() {
		settings.TiltBindings = append(settings.TiltBindings, GyroTiltBinding{
			Source: "Pitch", Direction: "+", Action: actionNone, Mode: "hold", Threshold: 15,
		})
		publish()
		rebuild()
	})
	add.Importance = widget.HighImportance
	rebuild()
	return container.NewVScroll(container.NewVBox(
		card(container.NewVBox(
			heading("Tilt gestures", 16),
			caption("Tilt the cube past the threshold to trigger a button. Hold keeps it pressed; tap fires once."),
			add,
		)),
		rows,
	))
}

func tiltRow(settings *ControllerSettings, i int, actionOptions []string, publish, rebuild func()) fyne.CanvasObject {
	tb := settings.TiltBindings[i]
	src := widget.NewSelect(gyroSources, func(v string) { settings.TiltBindings[i].Source = v; publish() })
	src.SetSelected(tb.Source)
	dir := widget.NewSelect(tiltDirections, func(v string) { settings.TiltBindings[i].Direction = v; publish() })
	dir.SetSelected(tb.Direction)
	act := widget.NewSelect(actionOptions, func(v string) { settings.TiltBindings[i].Action = v; publish() })
	if tb.Action == "" {
		act.SetSelected(actionNone)
	} else {
		act.SetSelected(tb.Action)
	}
	mode := widget.NewSelect(tiltModes, func(v string) { settings.TiltBindings[i].Mode = v; publish() })
	mode.SetSelected(tb.Mode)

	thrLabel := caption("Threshold: " + degStr(tb.Threshold))
	thr := widget.NewSlider(2, 45)
	thr.Step = 1
	thr.Value = tb.Threshold
	thr.OnChanged = func(v float64) {
		settings.TiltBindings[i].Threshold = v
		thrLabel.Text = "Threshold: " + degStr(v)
		thrLabel.Refresh()
		publish()
	}

	remove := widget.NewButton("Remove", func() {
		settings.TiltBindings = append(settings.TiltBindings[:i], settings.TiltBindings[i+1:]...)
		publish()
		rebuild()
	})
	remove.Importance = widget.DangerImportance

	top := container.NewGridWithColumns(4,
		labeled("Axis", src), labeled("Direction", dir), labeled("Button", act), labeled("Mode", mode))
	return card(container.NewVBox(top, thrLabel, thr, container.NewHBox(remove)))
}

func buildAxisTab(settings *ControllerSettings, publish func()) fyne.CanvasObject {
	rows := container.NewVBox()
	var rebuild func()
	rebuild = func() {
		rows.RemoveAll()
		for i := range settings.AxisBindings {
			rows.Add(axisRow(settings, i, publish, rebuild))
		}
		rows.Refresh()
	}
	add := widget.NewButton("Add axis binding", func() {
		settings.AxisBindings = append(settings.AxisBindings, GyroAxisBinding{
			Source: "Pitch", Target: "Right Y", Deadzone: 5, Range: 45,
		})
		publish()
		rebuild()
	})
	add.Importance = widget.HighImportance
	rebuild()
	return container.NewVScroll(container.NewVBox(
		card(container.NewVBox(
			heading("Analog axes", 16),
			caption("Map a rotation angle (relative to neutral) onto a stick or trigger. Deadzone is the center dead band."),
			add,
		)),
		rows,
	))
}

func axisRow(settings *ControllerSettings, i int, publish, rebuild func()) fyne.CanvasObject {
	ab := settings.AxisBindings[i]
	src := widget.NewSelect(gyroSources, func(v string) { settings.AxisBindings[i].Source = v; publish() })
	src.SetSelected(ab.Source)
	tgt := widget.NewSelect(controller.Axes, func(v string) { settings.AxisBindings[i].Target = v; publish() })
	tgt.SetSelected(ab.Target)
	inv := widget.NewCheck("Invert", func(b bool) { settings.AxisBindings[i].Invert = b; publish() })
	inv.SetChecked(ab.Invert)

	dzLabel := caption("Deadzone: " + degStr(ab.Deadzone))
	dz := widget.NewSlider(0, 45)
	dz.Step = 1
	dz.Value = ab.Deadzone
	dz.OnChanged = func(v float64) {
		settings.AxisBindings[i].Deadzone = v
		dzLabel.Text = "Deadzone: " + degStr(v)
		dzLabel.Refresh()
		publish()
	}

	rgLabel := caption("Range: " + degStr(ab.Range))
	rg := widget.NewSlider(5, 90)
	rg.Step = 1
	rg.Value = ab.Range
	rg.OnChanged = func(v float64) {
		settings.AxisBindings[i].Range = v
		rgLabel.Text = "Range: " + degStr(v)
		rgLabel.Refresh()
		publish()
	}

	remove := widget.NewButton("Remove", func() {
		settings.AxisBindings = append(settings.AxisBindings[:i], settings.AxisBindings[i+1:]...)
		publish()
		rebuild()
	})
	remove.Importance = widget.DangerImportance

	top := container.NewGridWithColumns(3, labeled("Axis", src), labeled("Target", tgt), labeled(" ", inv))
	return card(container.NewVBox(top, dzLabel, dz, rgLabel, rg, container.NewHBox(remove)))
}

func buildLiveTab(sphere *GyroSphere, preview *gyroPreview, settings *ControllerSettings, publish func()) fyne.CanvasObject {
	pCard, pVal := statCard("Pitch")
	rCard, rVal := statCard("Roll")
	yCard, yVal := statCard("Yaw")
	preview.pitch, preview.roll, preview.yaw = pVal, rVal, yVal
	axisOut := widget.NewLabel("")
	preview.axisOut = axisOut

	sphere.SetMinSize(fyne.NewSize(160, 160))

	calibrate := widget.NewButton("Calibrate neutral", func() {
		q := sphere.Quaternion()
		settings.Neutral = &q
		publish()
	})
	calibrate.Importance = widget.HighImportance

	smLabel := caption("Smoothing: " + strconv.FormatFloat(settings.Smoothing, 'f', 2, 64))
	sm := widget.NewSlider(0.05, 0.9)
	sm.Step = 0.05
	sm.Value = settings.Smoothing
	sm.OnChanged = func(v float64) {
		settings.Smoothing = v
		smLabel.Text = "Smoothing: " + strconv.FormatFloat(v, 'f', 2, 64)
		smLabel.Refresh()
		publish()
	}

	rfLabel := caption("Release factor: " + strconv.FormatFloat(settings.ReleaseFactor, 'f', 2, 64))
	rf := widget.NewSlider(0.3, 0.95)
	rf.Step = 0.05
	rf.Value = settings.ReleaseFactor
	rf.OnChanged = func(v float64) {
		settings.ReleaseFactor = v
		rfLabel.Text = "Release factor: " + strconv.FormatFloat(v, 'f', 2, 64)
		rfLabel.Refresh()
		publish()
	}

	stats := container.NewGridWithColumns(3, pCard, rCard, yCard)
	return container.NewVScroll(container.NewVBox(
		card(container.NewVBox(heading("Live orientation", 16), container.NewCenter(sphere), stats)),
		card(container.NewVBox(heading("Axis output", 16), axisOut)),
		card(container.NewVBox(
			heading("Neutral pose", 16),
			caption("Hold the cube in your rest pose, then calibrate so tilts are measured from there."),
			calibrate,
		)),
		card(container.NewVBox(heading("Response", 16), smLabel, sm, rfLabel, rf)),
	))
}

func (a *App) runGyroController(ctx context.Context, c *controller.Controller, cfg *atomic.Pointer[ControllerSettings], sphere *GyroSphere, preview *gyroPreview) {
	go func() {
		ticker := time.NewTicker(16 * time.Millisecond)
		defer ticker.Stop()
		displayed := cube.Quaternion{W: 1}
		var active []bool
		tick := 0

		release := func() {
			s := cfg.Load()
			if s == nil {
				return
			}
			for i, tb := range s.TiltBindings {
				if i < len(active) && active[i] && tb.Mode == "hold" {
					c.SetButton(tb.Action, false)
				}
			}
			for _, ab := range s.AxisBindings {
				if code, ok := controller.AxisCode(ab.Target); ok {
					c.SetAxis(code, 0)
				}
			}
		}

		for {
			select {
			case <-ctx.Done():
				release()
				return
			case <-ticker.C:
			}

			s := cfg.Load()
			if s == nil {
				continue
			}
			target := a.cube.Gyro()
			if target == (cube.Quaternion{}) {
				continue
			}
			displayed = quatNlerp(displayed, target, s.Smoothing)

			neutral := cube.Quaternion{W: 1}
			if s.Neutral != nil {
				neutral = *s.Neutral
			}
			e := relativeEuler(neutral, displayed)

			if len(active) != len(s.TiltBindings) {
				active = make([]bool, len(s.TiltBindings))
			}

			c.Frame(func(w func(eventType, code, value int32)) {
				for _, ab := range s.AxisBindings {
					code, ok := controller.AxisCode(ab.Target)
					if !ok {
						continue
					}
					v := angleToAxis(eulerComponent(e, ab.Source), ab.Deadzone, ab.Range, ab.Invert)
					w(controller.EV_ABS, int32(code), v)
				}
			})

			for i, tb := range s.TiltBindings {
				if tb.Action == "" || tb.Action == actionNone {
					continue
				}
				dir := 1.0
				if tb.Direction == "-" {
					dir = -1.0
				}
				val := dir * eulerComponent(e, tb.Source)
				onT := tb.Threshold
				offT := onT * s.ReleaseFactor
				if !active[i] && val >= onT {
					active[i] = true
					if tb.Mode == "tap" {
						go c.Press(tb.Action, time.Duration(s.HoldMs))
					} else {
						c.SetButton(tb.Action, true)
					}
				} else if active[i] && val < offT {
					active[i] = false
					if tb.Mode != "tap" {
						c.SetButton(tb.Action, false)
					}
				}
			}

			tick++
			if tick%4 == 0 {
				sphere.SetQuaternion(displayed)
				updatePreview(preview, e, s)
			}
		}
	}()
}

func updatePreview(preview *gyroPreview, e Euler, s *ControllerSettings) {
	if preview.pitch != nil {
		preview.pitch.Text = degStr(e.Pitch)
		preview.pitch.Refresh()
		preview.roll.Text = degStr(e.Roll)
		preview.roll.Refresh()
		preview.yaw.Text = degStr(e.Yaw)
		preview.yaw.Refresh()
	}
	if preview.axisOut != nil {
		out := ""
		for _, ab := range s.AxisBindings {
			v := angleToAxis(eulerComponent(e, ab.Source), ab.Deadzone, ab.Range, ab.Invert)
			out += ab.Target + " (" + ab.Source + "): " + strconv.Itoa(int(v)) + "\n"
		}
		if out == "" {
			out = "No axes bound."
		}
		preview.axisOut.SetText(out)
	}
}
