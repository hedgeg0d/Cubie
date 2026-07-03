package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

var (
	colDone  = color.RGBA{40, 200, 80, 255}
	colWrong = color.RGBA{225, 55, 55, 255}
	colNext  = color.RGBA{240, 240, 240, 255}
	colTodo  = color.RGBA{110, 113, 130, 255}
)

func newScrambleStrip(n int) (fyne.CanvasObject, []*canvas.Text) {
	texts := make([]*canvas.Text, n)
	objs := make([]fyne.CanvasObject, n)
	for i := range texts {
		t := canvas.NewText("", colTodo)
		t.TextSize = 22
		t.TextStyle = fyne.TextStyle{Bold: true}
		t.Alignment = fyne.TextAlignCenter
		texts[i] = t
		objs[i] = t
	}
	return container.NewGridWrap(fyne.NewSize(48, 34), objs...), texts
}

func paintScrambleStrip(texts []*canvas.Text, scramble []string, idx int, half string, wrongN int) {
	for i, t := range texts {
		if i >= len(scramble) {
			t.Text = ""
			t.Refresh()
			continue
		}
		t.Text = scramble[i]
		if i == idx && half != "" {
			t.Text = half
		}
		t.TextStyle = fyne.TextStyle{Bold: true}
		switch {
		case i < idx:
			t.Color = colDone
		case i == idx && wrongN > 0:
			t.Color = colWrong
		case i == idx:
			t.Color = colNext
		default:
			t.Color = colTodo
		}
		t.Refresh()
	}
}
