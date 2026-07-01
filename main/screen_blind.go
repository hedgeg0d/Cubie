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
	blindIdle = iota
	blindMemo
	blindExec
)

type blindCtl struct {
	mu        sync.Mutex
	phase     int
	memoStart time.Time
	execStart time.Time
	memoMs    int64
	stopFn    context.CancelFunc
}

func (a *App) showBlind() {
	a.switchScreen(fyne.NewSize(650, 550), func(ctx context.Context) fyne.CanvasObject {
		attempts := loadAttempts()
		scramble := cubestate.GenerateScramble(20)

		scrambleLabel := widget.NewLabel(cubestate.ScrambleString(scramble))
		scrambleLabel.Wrapping = fyne.TextWrapWord
		memoLabel := widget.NewLabel("Memo: 0.00")
		execLabel := widget.NewLabel("Exec: 0.00")
		hintLabel := widget.NewLabel("Scramble, then Start Memo")
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

		newScramble := func() {
			scramble = cubestate.GenerateScramble(20)
			scrambleLabel.SetText(cubestate.ScrambleString(scramble))
			scrambleLabel.Show()
			scrambleLabel.Refresh()
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
			if success {
				hintLabel.SetText("Solved! Start Memo for next")
			} else {
				hintLabel.SetText("Missed. Start Memo for next")
			}
			hintLabel.Refresh()
			newScramble()
		}

		startMemo := widget.NewButton("Start Memo", func() {
			ctl.mu.Lock()
			if ctl.phase != blindIdle {
				ctl.mu.Unlock()
				return
			}
			ctl.phase = blindMemo
			ctl.memoStart = time.Now()
			start := ctl.memoStart
			ctl.mu.Unlock()

			hintLabel.SetText("Memorize, then turn to start solving")
			hintLabel.Refresh()

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

		a.cube.OnMove = func(string) {
			ctl.mu.Lock()
			if ctl.phase != blindMemo {
				ctl.mu.Unlock()
				return
			}
			ctl.phase = blindExec
			ctl.memoMs = time.Since(ctl.memoStart).Milliseconds()
			ctl.execStart = time.Now()
			sctx, cancel := context.WithCancel(ctx)
			ctl.stopFn = cancel
			execStart := ctl.execStart
			memoMs := ctl.memoMs
			ctl.mu.Unlock()

			scrambleLabel.Hide()
			memoLabel.SetText("Memo: " + formatMs(memoMs))
			memoLabel.Refresh()
			hintLabel.SetText("Solving blind...")
			hintLabel.Refresh()

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
						ctl.phase = blindIdle
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
		}

		giveUp := widget.NewButton("Give Up", func() {
			ctl.mu.Lock()
			if ctl.phase != blindExec {
				ctl.mu.Unlock()
				return
			}
			ctl.phase = blindIdle
			elapsed := time.Since(ctl.execStart).Milliseconds()
			if ctl.stopFn != nil {
				ctl.stopFn()
			}
			ctl.mu.Unlock()
			record(elapsed, false)
		})

		refreshStats()

		controls := container.NewHBox(startMemo, giveUp, widget.NewButton("Back", a.showMenu))
		top := container.NewVBox(scrambleLabel, memoLabel, execLabel, hintLabel, statsLabel, controls)
		return container.NewBorder(top, nil, nil, nil, list)
	})
}
