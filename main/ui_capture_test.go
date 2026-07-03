package main

import (
	"image/png"
	"os"
	"testing"

	"cubie/cube"
	"cubie/cubestate"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func cubestateSample() *cubestate.Model {
	m := cubestate.NewSolved()
	for _, mv := range []string{"R", "U", "R'", "U'", "F", "R", "F'"} {
		m.Apply(mv)
	}
	return m
}

func capture(t *testing.T, path string, size fyne.Size, content fyne.CanvasObject) {
	w := test.NewWindow(content)
	defer w.Close()
	w.Resize(size)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, w.Canvas().Capture()); err != nil {
		t.Fatal(err)
	}
}

func TestCaptureUI(t *testing.T) {
	app := test.NewApp()
	app.Settings().SetTheme(cubieTheme{})
	dir := t.TempDir()

	cubeView := NewCubeView(cubestateSample())
	cubeView.SetMinSize(fyne.NewSize(320, 320))
	gyro := NewGyroSphere()
	gyro.SetQuaternion(cube.Quaternion{W: 0.82, X: 0.31, Y: 0.34, Z: 0.33})
	movesStrip, updateMoves := newMovesStrip()
	updateMoves([]string{"F", "R", "U'", "R'", "U"})
	followCheck := widget.NewCheck("Rotate cube with gyroscope", nil)
	sync := widget.NewButton("Mark solved (sync)", nil)
	sync.Importance = widget.HighImportance

	left := card(container.NewBorder(nil, container.NewCenter(caption("Drag to rotate")), nil, nil, cubeView))
	right := card(container.NewVBox(
		heading("Gyroscope", 18), container.NewCenter(gyro), followCheck, widget.NewSeparator(),
		heading("Last moves", 16), container.NewCenter(movesStrip), layout.NewSpacer(), sync,
	))
	grid := container.NewGridWithColumns(2, left, right)

	modeTiles := container.NewGridWithColumns(3,
		NewModeTile("Controller", "Cube as a gamepad", accentCyan, nil),
		NewModeTile("Timer", "Speedcubing timer", accentGreen, nil),
		NewModeTile("Blind", "Memo & exec timing", accentAmber, nil),
	)
	disconnect := widget.NewButton("Disconnect", nil)
	disconnect.Importance = widget.DangerImportance

	title := heading("CUBIE", 28)
	batteryPill := pill(heading("Battery 70%", 15), accentGreen)
	header := container.NewBorder(nil, nil,
		container.NewVBox(title, caption("Weilong V10 AI")),
		container.NewCenter(batteryPill),
	)

	home := container.NewPadded(container.NewBorder(
		container.NewPadded(header),
		container.NewVBox(container.NewPadded(modeTiles), container.NewPadded(disconnect)),
		nil, nil,
		grid,
	))
	capture(t, dir+"/ui_home.png", fyne.NewSize(980, 680), home)
}
