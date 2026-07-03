package main

import (
	"context"
	"strconv"
	"time"

	"cubie/cube"
	"cubie/cubestate"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

func (a *App) showMenu() {
	if a.cubeModel == nil {
		a.cubeModel = cubestate.NewSolved()
	}
	cubeView := NewCubeView(a.cubeModel)
	cubeView.SetMinSize(fyne.NewSize(320, 320))
	gyroSphere := NewGyroSphere()

	a.hideMini = true
	a.switchScreen(fyne.NewSize(980, 700), func(ctx context.Context) fyne.CanvasObject {
		title := heading("CUBIE", 28)
		subtitle := caption(a.model)

		batteryText := heading("Battery --%", 15)
		batteryPill := pill(batteryText, accentGreen)
		go a.pollBattery(ctx, func(s string) {
			batteryText.Text = s
			batteryText.Refresh()
		})

		movesStrip, updateMoves := newMovesStrip()
		a.cube.OnMove = func(string) {
			updateMoves(a.cube.LastMovesList())
		}
		updateMoves(a.cube.LastMovesList())

		go func() {
			ticker := time.NewTicker(33 * time.Millisecond)
			defer ticker.Stop()
			displayed := cube.Quaternion{W: 1}
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					target := a.cube.Gyro()
					if target == (cube.Quaternion{}) {
						continue
					}
					displayed = quatNlerp(displayed, target, 0.3)
					gyroSphere.SetQuaternion(displayed)
					cubeView.SetGyro(displayed)
				}
			}
		}()

		followCheck := widget.NewCheck("Rotate cube with gyroscope", func(b bool) {
			cubeView.SetFollowGyro(b)
		})

		syncButton := widget.NewButton("Mark solved (sync)", func() {
			a.setMiniSolved()
		})
		syncButton.Importance = widget.HighImportance

		header := container.NewBorder(nil, nil,
			container.NewVBox(title, subtitle),
			container.NewCenter(batteryPill),
		)

		left := card(container.NewBorder(
			nil, container.NewCenter(caption("Drag to rotate")), nil, nil,
			cubeView,
		))

		right := card(container.NewVBox(
			heading("Gyroscope", 18),
			container.NewCenter(gyroSphere),
			followCheck,
			widget.NewSeparator(),
			heading("Last moves", 16),
			container.NewCenter(movesStrip),
			layout.NewSpacer(),
			syncButton,
		))

		grid := container.NewGridWithColumns(2, left, right)

		modeTiles := container.NewGridWithColumns(4,
			NewModeTile("Controller", "Cube as a gamepad", accentCyan, a.showController),
			NewModeTile("Timer", "Speedcubing timer", accentGreen, a.showTimer),
			NewModeTile("Blind", "Memo & exec timing", accentAmber, a.showBlind),
			NewModeTile("Lettering", "Blind memo scheme", accentColor, a.showLettering),
		)

		disconnect := widget.NewButton("Disconnect", func() {
			a.disconnect()
			a.showConnect()
		})
		disconnect.Importance = widget.DangerImportance

		bottom := container.NewVBox(
			container.NewPadded(modeTiles),
			container.NewPadded(disconnect),
		)

		return container.NewPadded(container.NewBorder(
			container.NewPadded(header),
			bottom,
			nil, nil,
			grid,
		))
	})
	a.miniCube = cubeView
}

func (a *App) pollBattery(ctx context.Context, set func(string)) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		a.cube.UpdatePowerInfo()
		set("Battery " + strconv.Itoa(a.cube.Power) + "%")
		select {
		case <-ctx.Done():
			return
		case <-time.After(60 * time.Second):
		}
	}
}
