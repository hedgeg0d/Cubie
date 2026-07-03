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
	colTodo  = color.RGBA{120, 120, 120, 255}
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
	a.switchScreen(fyne.NewSize(680, 560), func(ctx context.Context) fyne.CanvasObject {
		solves := loadSolves()
		scramble := cubestate.GenerateScrambleQuarter(scrambleLen)

		scrambleTexts := make([]*canvas.Text, scrambleLen)
		scrambleObjs := make([]fyne.CanvasObject, scrambleLen)
		for i := range scrambleTexts {
			t := canvas.NewText("", colTodo)
			t.TextSize = 20
			t.Alignment = fyne.TextAlignCenter
			scrambleTexts[i] = t
			scrambleObjs[i] = t
		}
		scrambleGrid := container.NewGridWrap(fyne.NewSize(46, 30), scrambleObjs...)

		timeLabel := widget.NewLabel("0.00")
		timeLabel.TextStyle = fyne.TextStyle{Bold: true}
		hintLabel := widget.NewLabel("")
		statsLabel := widget.NewLabel("")

		ctl := &timerCtl{}

		setHint := func(text string) {
			hintLabel.SetText(text)
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
				t.TextStyle = fyne.TextStyle{}
				switch {
				case i < idx:
					t.Color = colDone
				case i == idx && errN > 0:
					t.Color = colWrong
					t.TextStyle = fyne.TextStyle{Bold: true}
				case i == idx:
					t.Color = colNext
					t.TextStyle = fyne.TextStyle{Bold: true}
				default:
					t.Color = colTodo
				}
				t.Refresh()
			}

			switch phase {
			case tsSolving:
			case tsReady:
				setHint("Scrambled! Turn to start the solve")
			default:
				if errN > 0 {
					setHint("Wrong move — undo " + undo + " (" + itoa(errN) + " to fix)")
				} else {
					setHint("Scramble: green = done, white = next")
				}
			}
		}

		refreshStats := func() {
			statsLabel.SetText(
				"best: " + formatMs(best(solves)) +
					"   ao5: " + formatMs(averageOf(solves, 5)) +
					"   ao12: " + formatMs(averageOf(solves, 12)) +
					"   mean: " + formatMs(mean(solves)) +
					"   solves: " + itoa(len(solves)),
			)
			statsLabel.Refresh()
		}

		list := widget.NewList(
			func() int { return len(solves) },
			func() fyne.CanvasObject { return widget.NewLabel("") },
			func(i widget.ListItemID, o fyne.CanvasObject) {
				s := solves[len(solves)-1-i]
				label := formatMs(s.Ms)
				if s.Penalty != "" {
					label += " (" + s.Penalty + ")"
				}
				o.(*widget.Label).SetText(label)
			},
		)

		resetScramble := func() {
			scramble = cubestate.GenerateScrambleQuarter(scrambleLen)
			ctl.mu.Lock()
			ctl.phase = tsScramble
			ctl.idx = 0
			ctl.wrongStack = nil
			ctl.mu.Unlock()
			timeLabel.SetText("0.00")
			timeLabel.Refresh()
			updateUI()
		}

		finalize := func(elapsed int64) {
			solves = append(solves, Solve{
				Ms:       elapsed,
				Scramble: cubestate.ScrambleString(scramble),
				At:       time.Now().Unix(),
			})
			saveSolves(solves)
			timeLabel.SetText(formatMs(elapsed))
			timeLabel.Refresh()
			refreshStats()
			list.Refresh()
			resetScramble()
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

			setHint("Solving...")

			go func() {
				ticker := time.NewTicker(30 * time.Millisecond)
				defer ticker.Stop()
				for {
					select {
					case <-sctx.Done():
						return
					case <-ticker.C:
						timeLabel.SetText(formatMs(time.Since(start).Milliseconds()))
						timeLabel.Refresh()
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

		controls := container.NewHBox(
			widget.NewButton("+2", func() { penalty("+2") }),
			widget.NewButton("DNF", func() { penalty("DNF") }),
			widget.NewButton("Delete", deleteLast),
			widget.NewButton("New Scramble", resetScramble),
			widget.NewButton("Back", a.showMenu),
		)

		top := container.NewVBox(scrambleGrid, timeLabel, hintLabel, statsLabel, controls)
		return container.NewBorder(top, nil, nil, nil, list)
	})
}
