package main

import (
	"context"
	"image/color"
	"sync"
	"time"

	"cubie/cubestate"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const (
	tsScramble = iota
	tsReady
	tsSolving
)

const scrambleLen = 20

var (
	colDone  = color.RGBA{40, 200, 80, 255}
	colWrong = color.RGBA{225, 55, 55, 255}
	colNext  = color.RGBA{240, 240, 240, 255}
	colTodo  = color.RGBA{110, 113, 130, 255}

	segIdle    = color.RGBA{205, 208, 222, 255}
	segRunning = color.RGBA{0x7C, 0x5C, 0xFF, 255}
	segDone    = color.RGBA{52, 211, 153, 255}
)

type timerCtl struct {
	mu         sync.Mutex
	phase      int
	idx        int
	wrongStack []string
	start      time.Time
	stopFn     context.CancelFunc
}

func (a *App) showTimer() {
	a.switchScreen(fyne.NewSize(780, 880), func(ctx context.Context) fyne.CanvasObject {
		solves := loadSolves()
		scramble := cubestate.GenerateScrambleQuarter(scrambleLen)

		scrambleTexts := make([]*canvas.Text, scrambleLen)
		scrambleObjs := make([]fyne.CanvasObject, scrambleLen)
		for i := range scrambleTexts {
			t := canvas.NewText("", colTodo)
			t.TextSize = 22
			t.TextStyle = fyne.TextStyle{Bold: true}
			t.Alignment = fyne.TextAlignCenter
			scrambleTexts[i] = t
			scrambleObjs[i] = t
		}
		scrambleGrid := container.NewGridWrap(fyne.NewSize(48, 34), scrambleObjs...)

		display := NewRollingTimer()
		display.minSize = fyne.NewSize(380, 140)
		hintLabel := caption("")
		hintLabel.Alignment = fyne.TextAlignCenter

		bestCard, bestVal := statCard("best")
		ao5Card, ao5Val := statCard("ao5")
		ao12Card, ao12Val := statCard("ao12")
		meanCard, meanVal := statCard("mean")
		statsRow := container.NewGridWithColumns(4, bestCard, ao5Card, ao12Card, meanCard)

		ctl := &timerCtl{}

		refreshStats := func() {
			bestVal.Text = formatMs(best(solves))
			ao5Val.Text = formatMs(averageOf(solves, 5))
			ao12Val.Text = formatMs(averageOf(solves, 12))
			meanVal.Text = formatMs(mean(solves))
			bestVal.Refresh()
			ao5Val.Refresh()
			ao12Val.Refresh()
			meanVal.Refresh()
		}

		list := widget.NewList(
			func() int { return len(solves) },
			func() fyne.CanvasObject { return widget.NewLabel("") },
			func(i widget.ListItemID, o fyne.CanvasObject) {
				s := solves[len(solves)-1-i]
				label := formatMs(s.Ms)
				if s.Penalty != "" {
					label += "  (" + s.Penalty + ")"
				}
				o.(*widget.Label).SetText(label)
			},
		)

		setHint := func(text string) {
			hintLabel.Text = text
			hintLabel.Refresh()
		}

		updateUI := func() {
			ctl.mu.Lock()
			phase, idx := ctl.phase, ctl.idx
			errN := len(ctl.wrongStack)
			undo := ""
			if errN > 0 {
				undo = cubestate.InvertMove(ctl.wrongStack[errN-1])
			}
			ctl.mu.Unlock()

			for i, t := range scrambleTexts {
				t.Text = scramble[i]
				t.TextStyle = fyne.TextStyle{Bold: true}
				switch {
				case i < idx:
					t.Color = colDone
				case i == idx && errN > 0:
					t.Color = colWrong
				case i == idx:
					t.Color = colNext
				default:
					t.Color = colTodo
				}
				t.Refresh()
			}

			switch phase {
			case tsSolving:
			case tsReady:
				setHint("Scrambled — turn to start the solve")
			default:
				if errN > 0 {
					setHint("Wrong move — undo " + undo + "  (" + itoa(errN) + " to fix)")
				} else {
					setHint("Scramble the cube: green done · white next")
				}
			}
		}

		resetScramble := func() {
			scramble = cubestate.GenerateScrambleQuarter(scrambleLen)
			ctl.mu.Lock()
			ctl.phase = tsScramble
			ctl.idx = 0
			ctl.wrongStack = nil
			ctl.mu.Unlock()
			display.SetColor(segIdle)
			display.SetText("0.00")
			updateUI()
		}

		finalize := func(elapsed int64) {
			solves = append(solves, Solve{
				Ms:       elapsed,
				Scramble: cubestate.ScrambleString(scramble),
				At:       time.Now().Unix(),
			})
			saveSolves(solves)
			display.SetColor(segDone)
			display.SetText(formatMs(elapsed))
			refreshStats()
			list.Refresh()
			resetScramble()
			display.SetColor(segDone)
			display.SetText(formatMs(elapsed))
			setHint("Solved " + formatMs(elapsed) + " — scramble again")
		}

		startSolve := func() {
			ctl.mu.Lock()
			ctl.phase = tsSolving
			ctl.start = time.Now()
			sctx, cancel := context.WithCancel(ctx)
			ctl.stopFn = cancel
			start := ctl.start
			ctl.mu.Unlock()

			display.SetColor(segRunning)
			setHint("Solving...")

			go func() {
				ticker := time.NewTicker(30 * time.Millisecond)
				defer ticker.Stop()
				for {
					select {
					case <-sctx.Done():
						return
					case <-ticker.C:
						display.SetText(formatMs(time.Since(start).Milliseconds()))
					}
				}
			}()

			go func() {
				for {
					select {
					case <-sctx.Done():
						return
					default:
					}
					a.cube.UpdateState()
					if a.cube.IsSolved() {
						ctl.mu.Lock()
						if ctl.phase != tsSolving {
							ctl.mu.Unlock()
							return
						}
						ctl.phase = tsScramble
						elapsed := time.Since(ctl.start).Milliseconds()
						ctl.stopFn()
						ctl.mu.Unlock()
						finalize(elapsed)
						return
					}
					time.Sleep(120 * time.Millisecond)
				}
			}()
		}

		a.cube.OnMove = func(move string) {
			ctl.mu.Lock()
			switch ctl.phase {
			case tsScramble:
				if len(ctl.wrongStack) > 0 {
					top := ctl.wrongStack[len(ctl.wrongStack)-1]
					if move == cubestate.InvertMove(top) {
						ctl.wrongStack = ctl.wrongStack[:len(ctl.wrongStack)-1]
					} else {
						ctl.wrongStack = append(ctl.wrongStack, move)
					}
				} else if move == scramble[ctl.idx] {
					ctl.idx++
					if ctl.idx == len(scramble) {
						ctl.phase = tsReady
					}
				} else {
					ctl.wrongStack = append(ctl.wrongStack, move)
				}
				ctl.mu.Unlock()
				updateUI()
			case tsReady:
				ctl.mu.Unlock()
				startSolve()
			default:
				ctl.mu.Unlock()
			}
		}

		penalty := func(p string) {
			if len(solves) == 0 {
				return
			}
			last := &solves[len(solves)-1]
			if last.Penalty == p {
				last.Penalty = ""
			} else {
				last.Penalty = p
			}
			saveSolves(solves)
			refreshStats()
			list.Refresh()
		}

		deleteLast := func() {
			if len(solves) == 0 {
				return
			}
			solves = solves[:len(solves)-1]
			saveSolves(solves)
			refreshStats()
			list.Refresh()
		}

		resetScramble()
		refreshStats()

		plus2 := widget.NewButton("+2", func() { penalty("+2") })
		plus2.Importance = widget.WarningImportance
		dnf := widget.NewButton("DNF", func() { penalty("DNF") })
		dnf.Importance = widget.DangerImportance
		del := widget.NewButton("Delete", deleteLast)
		newScr := widget.NewButton("New Scramble", resetScramble)
		newScr.Importance = widget.HighImportance
		controls := container.NewGridWithColumns(4, plus2, dnf, del, newScr)

		header := container.NewBorder(nil, nil,
			heading("Timer", 26), widget.NewButton("Back", a.showMenu),
		)

		timerCard := card(container.NewVBox(
			container.NewCenter(container.NewGridWrap(fyne.NewSize(420, 150), display)),
			hintLabel,
		))

		top := container.NewVBox(
			container.NewPadded(header),
			card(scrambleGrid),
			timerCard,
			statsRow,
			container.NewPadded(controls),
		)

		reserve := container.NewGridWrap(
			fyne.NewSize(1, 195),
			canvas.NewRectangle(color.RGBA{0, 0, 0, 0}),
		)

		return container.NewPadded(container.NewBorder(
			top, reserve, nil, nil,
			card(list),
		))
	})
}
