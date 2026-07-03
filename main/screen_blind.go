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

const (
	blindScramble = iota
	blindReady
	blindMemo
	blindExec
)

type blindCtl struct {
	mu         sync.Mutex
	phase      int
	idx        int
	half       string
	wrongStack []string
	memoStart  time.Time
	execStart  time.Time
	memoMs     int64
	stopFn     context.CancelFunc
}

func (a *App) showBlind() {
	a.switchScreen(fyne.NewSize(650, 600), func(ctx context.Context) fyne.CanvasObject {
		attempts := loadAttempts()
		scramble := cubestate.GenerateScramble(scrambleLen)

		scrambleGrid, scrambleTexts := newScrambleStrip(scrambleLen)
		memoLabel := widget.NewLabel("Memo: 0.00")
		execLabel := widget.NewLabel("Exec: 0.00")
		hintLabel := widget.NewLabel("")
		statsLabel := widget.NewLabel("")

		refreshStats := func() {
			statsLabel.SetText(
				"success: " + itoa(successCount(attempts)) + "/" + itoa(len(attempts)) +
					" (" + itoa(successRate(attempts)) + "%)" +
					"   best: " + formatMs(bestTotal(attempts)),
			)
			statsLabel.Refresh()
		}

		list := widget.NewList(
			func() int { return len(attempts) },
			func() fyne.CanvasObject { return widget.NewLabel("") },
			func(i widget.ListItemID, o fyne.CanvasObject) {
				at := attempts[len(attempts)-1-i]
				result := "OK"
				if !at.Success {
					result = "DNF"
				}
				o.(*widget.Label).SetText(
					result + "  memo " + formatMs(at.MemoMs) + "  exec " + formatMs(at.ExecMs),
				)
			},
		)

		ctl := &blindCtl{}

		setHint := func(text string) {
			hintLabel.SetText(text)
			hintLabel.Refresh()
		}

		updateScramble := func() {
			ctl.mu.Lock()
			phase, idx, half := ctl.phase, ctl.idx, ctl.half
			errN := len(ctl.wrongStack)
			undo := ""
			if errN > 0 {
				undo = cubestate.InvertMove(ctl.wrongStack[errN-1])
			}
			ctl.mu.Unlock()

			paintScrambleStrip(scrambleTexts, scramble, idx, half, errN)

			switch phase {
			case blindReady:
				setHint("Scrambled — press Start Memo")
			case blindScramble:
				if errN > 0 {
					setHint("Wrong move — undo " + undo + "  (" + itoa(errN) + " to fix)")
				} else if half != "" {
					setHint("Double turn — one more " + half)
				} else {
					setHint("Scramble the cube: green done · white next")
				}
			}
		}

		newScramble := func() {
			scramble = cubestate.GenerateScramble(scrambleLen)
			ctl.mu.Lock()
			ctl.phase = blindScramble
			ctl.idx = 0
			ctl.half = ""
			ctl.wrongStack = nil
			ctl.mu.Unlock()
			memoLabel.SetText("Memo: 0.00")
			execLabel.SetText("Exec: 0.00")
			memoLabel.Refresh()
			execLabel.Refresh()
			scrambleGrid.Show()
			updateScramble()
		}

		record := func(execMs int64, success bool) {
			attempts = append(attempts, BlindAttempt{
				MemoMs:   ctl.memoMs,
				ExecMs:   execMs,
				Success:  success,
				Scramble: cubestate.ScrambleString(scramble),
				At:       time.Now().Unix(),
			})
			saveAttempts(attempts)
			refreshStats()
			list.Refresh()
			newScramble()
		}

		startMemo := widget.NewButton("Start Memo", func() {
			ctl.mu.Lock()
			if ctl.phase != blindReady {
				ctl.mu.Unlock()
				return
			}
			ctl.phase = blindMemo
			ctl.memoStart = time.Now()
			start := ctl.memoStart
			ctl.mu.Unlock()

			scrambleGrid.Hide()
			setHint("Memorize, then turn to start solving")

			go func() {
				ticker := time.NewTicker(50 * time.Millisecond)
				defer ticker.Stop()
				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						ctl.mu.Lock()
						memo := ctl.phase == blindMemo
						ctl.mu.Unlock()
						if !memo {
							return
						}
						memoLabel.SetText("Memo: " + formatMs(time.Since(start).Milliseconds()))
						memoLabel.Refresh()
					}
				}
			}()
		})

		a.cube.OnMove = func(move string) {
			ctl.mu.Lock()
			switch ctl.phase {
			case blindScramble:
				var advanced bool
				ctl.half, ctl.wrongStack, advanced = scrambleStep(scramble[ctl.idx], move, ctl.half, ctl.wrongStack)
				if advanced {
					ctl.idx++
					if ctl.idx == len(scramble) {
						ctl.phase = blindReady
					}
				}
				ctl.mu.Unlock()
				updateScramble()
			case blindMemo:
				ctl.phase = blindExec
				ctl.memoMs = time.Since(ctl.memoStart).Milliseconds()
				ctl.execStart = time.Now()
				sctx, cancel := context.WithCancel(ctx)
				ctl.stopFn = cancel
				execStart := ctl.execStart
				memoMs := ctl.memoMs
				ctl.mu.Unlock()

				memoLabel.SetText("Memo: " + formatMs(memoMs))
				memoLabel.Refresh()
				setHint("Solving blind...")

				go func() {
					ticker := time.NewTicker(30 * time.Millisecond)
					defer ticker.Stop()
					for {
						select {
						case <-sctx.Done():
							return
						case <-ticker.C:
							execLabel.SetText("Exec: " + formatMs(time.Since(execStart).Milliseconds()))
							execLabel.Refresh()
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
							if ctl.phase != blindExec {
								ctl.mu.Unlock()
								return
							}
							ctl.phase = blindScramble
							elapsed := time.Since(execStart).Milliseconds()
							ctl.stopFn()
							ctl.mu.Unlock()
							execLabel.SetText("Exec: " + formatMs(elapsed))
							execLabel.Refresh()
							record(elapsed, true)
							return
						}
						time.Sleep(120 * time.Millisecond)
					}
				}()
			default:
				ctl.mu.Unlock()
			}
		}

		giveUp := widget.NewButton("Give Up", func() {
			ctl.mu.Lock()
			if ctl.phase != blindExec {
				ctl.mu.Unlock()
				return
			}
			elapsed := time.Since(ctl.execStart).Milliseconds()
			if ctl.stopFn != nil {
				ctl.stopFn()
			}
			ctl.mu.Unlock()
			record(elapsed, false)
		})

		newScramble()
		refreshStats()

		controls := container.NewHBox(startMemo, giveUp, widget.NewButton("Back", a.showMenu))
		top := container.NewVBox(card(scrambleGrid), memoLabel, execLabel, hintLabel, statsLabel, controls)
		return container.NewBorder(top, nil, nil, nil, list)
	})
}
