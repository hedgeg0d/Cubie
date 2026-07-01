package main

import (
	"context"
	"sync"
	"time"

	"cubie/cubestate"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type timerCtl struct {
	mu      sync.Mutex
	running bool
	start   time.Time
	stopFn  context.CancelFunc
}

func (a *App) showTimer() {
	a.switchScreen(fyne.NewSize(650, 550), func(ctx context.Context) fyne.CanvasObject {
		solves := loadSolves()
		scramble := cubestate.GenerateScramble(20)

		timeLabel := widget.NewLabel("0.00")
		timeLabel.TextStyle = fyne.TextStyle{Bold: true}
		scrambleLabel := widget.NewLabel(cubestate.ScrambleString(scramble))
		scrambleLabel.Wrapping = fyne.TextWrapWord
		statsLabel := widget.NewLabel("")
		hintLabel := widget.NewLabel("Scramble the cube, then turn to start")

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

		newScramble := func() {
			scramble = cubestate.GenerateScramble(20)
			scrambleLabel.SetText(cubestate.ScrambleString(scramble))
			scrambleLabel.Refresh()
		}

		ctl := &timerCtl{}

		var finalize func(elapsed int64)
		finalize = func(elapsed int64) {
			solves = append(solves, Solve{
				Ms:       elapsed,
				Scramble: cubestate.ScrambleString(scramble),
				At:       time.Now().Unix(),
			})
			saveSolves(solves)
			timeLabel.SetText(formatMs(elapsed))
			timeLabel.Refresh()
			hintLabel.SetText("Solved. Scramble again to continue")
			hintLabel.Refresh()
			refreshStats()
			list.Refresh()
			newScramble()
		}

		a.cube.OnMove = func(string) {
			ctl.mu.Lock()
			if ctl.running {
				ctl.mu.Unlock()
				return
			}
			ctl.running = true
			ctl.start = time.Now()
			sctx, cancel := context.WithCancel(ctx)
			ctl.stopFn = cancel
			start := ctl.start
			ctl.mu.Unlock()

			hintLabel.SetText("Solving...")
			hintLabel.Refresh()

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
						if !ctl.running {
							ctl.mu.Unlock()
							return
						}
						ctl.running = false
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

		refreshStats()

		controls := container.NewHBox(
			widget.NewButton("+2", func() { penalty("+2") }),
			widget.NewButton("DNF", func() { penalty("DNF") }),
			widget.NewButton("Delete", deleteLast),
			widget.NewButton("New Scramble", newScramble),
			widget.NewButton("Back", a.showMenu),
		)

		top := container.NewVBox(scrambleLabel, timeLabel, hintLabel, statsLabel, controls)
		return container.NewBorder(top, nil, nil, nil, list)
	})
}
