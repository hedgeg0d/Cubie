package main

import (
	"context"
	"image/color"
	"strconv"
	"strings"
	"sync"
	"time"

	"cubie/cubestate"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func fallback(s, alt string) string {
	if s == "" {
		return alt
	}
	return s
}

func reconstruct(events []SolveEvent) (withRot, agnostic string, nMoves, nRot int) {
	o := cubestate.NewOrientation()
	var a, b []string
	for _, e := range events {
		switch e.Kind {
		case "rot":
			a = append(a, e.Val)
			o.Apply(e.Val)
			nRot++
		case "move":
			a = append(a, o.Remap(e.Val))
			b = append(b, e.Val)
			nMoves++
		}
	}
	return strings.Join(a, " "), strings.Join(b, " "), nMoves, nRot
}

const (
	tsScramble = iota
	tsReady
	tsSolving
)

const scrambleLen = 20

var (
	segIdle    = color.RGBA{205, 208, 222, 255}
	segRunning = color.RGBA{0x7C, 0x5C, 0xFF, 255}
	segDone    = color.RGBA{52, 211, 153, 255}
)

func scrambleStep(cur, move, half string, wrongStack []string) (string, []string, bool) {
	if len(wrongStack) > 0 {
		top := wrongStack[len(wrongStack)-1]
		if move == cubestate.InvertMove(top) {
			return half, wrongStack[:len(wrongStack)-1], false
		}
		return half, append(wrongStack, move), false
	}
	if strings.HasSuffix(cur, "2") {
		face := cur[:1]
		if half == "" {
			if move == face || move == face+"'" {
				return move, wrongStack, false
			}
			return half, append(wrongStack, move), false
		}
		if move == half {
			return "", wrongStack, true
		}
		if move == cubestate.InvertMove(half) {
			return "", wrongStack, false
		}
		return half, append(wrongStack, move), false
	}
	if move == cur {
		return half, wrongStack, true
	}
	return half, append(wrongStack, move), false
}

type timerCtl struct {
	mu         sync.Mutex
	phase      int
	idx        int
	half       string
	wrongStack []string
	events     []SolveEvent
	start      time.Time
	stopFn     context.CancelFunc
}

func (a *App) showTimer() {
	a.switchScreen(fyne.NewSize(780, 880), func(ctx context.Context) fyne.CanvasObject {
		sessions := loadSessions()
		active := sessions.Active
		solves := sessions.Sessions[active]
		if solves == nil {
			solves = []Solve{}
		}
		scramble := cubestate.GenerateScramble(scrambleLen)

		scrambleGrid, scrambleTexts := newScrambleStrip(scrambleLen)

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

		saveCurrent := func() {
			sessions.Sessions[active] = solves
			saveSessions(sessions)
		}

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

		selected := -1

		showDetails := func() {
			if selected < 0 || selected >= len(solves) {
				return
			}
			s := solves[len(solves)-1-selected]
			withRot, agnostic, nMoves, nRot := reconstruct(s.Events)
			tps := ""
			if s.Ms > 0 && nMoves > 0 {
				tps = "  ·  " + strconv.FormatFloat(float64(nMoves)/(float64(s.Ms)/1000), 'f', 2, 64) + " tps"
			}
			penalty := s.Penalty
			if penalty == "" {
				penalty = "none"
			}
			solLabel := widget.NewLabel(fallback(withRot, "not recorded"))
			solLabel.Wrapping = fyne.TextWrapWord
			agLabel := widget.NewLabel(fallback(agnostic, "not recorded"))
			agLabel.Wrapping = fyne.TextWrapWord
			scrLabel := widget.NewLabel(s.Scramble)
			scrLabel.Wrapping = fyne.TextWrapWord
			rows := container.NewVBox(
				heading(formatMs(s.Ms)+tps, 22),
				caption(time.Unix(s.At, 0).Format("2006-01-02 15:04")+"   ·   penalty: "+penalty),
				widget.NewSeparator(),
				caption("Scramble"),
				scrLabel,
				caption("Solution — as performed ("+itoa(nMoves)+" moves, "+itoa(nRot)+" rotations)"),
				solLabel,
				caption("Solution — rotation-agnostic"),
				agLabel,
			)
			d := dialog.NewCustom("Solve details", "Close", container.NewVScroll(rows), a.window)
			d.Resize(fyne.NewSize(520, 460))
			d.Show()
		}

		var lastSel widget.ListItemID = -1
		var lastSelAt time.Time
		list.OnSelected = func(id widget.ListItemID) {
			selected = id
			if id == lastSel && time.Since(lastSelAt) < 500*time.Millisecond {
				showDetails()
			}
			lastSel = id
			lastSelAt = time.Now()
		}
		list.OnUnselected = func(widget.ListItemID) { selected = -1 }

		setHint := func(text string) {
			hintLabel.Text = text
			hintLabel.Refresh()
		}

		var penalty func(string)

		updateUI := func() {
			ctl.mu.Lock()
			phase, idx := ctl.phase, ctl.idx
			half := ctl.half
			errN := len(ctl.wrongStack)
			undo := ""
			if errN > 0 {
				undo = cubestate.InvertMove(ctl.wrongStack[errN-1])
			}
			ctl.mu.Unlock()

			paintScrambleStrip(scrambleTexts, scramble, idx, half, errN)

			switch phase {
			case tsSolving:
			case tsReady:
				setHint("Scrambled — turn to start the solve")
			default:
				if errN > 0 {
					setHint("Wrong move — undo " + undo + "  (" + itoa(errN) + " to fix)")
				} else if half != "" {
					setHint("Double turn — one more " + half)
				} else {
					setHint("Scramble the cube: green done · white next")
				}
			}
		}

		resetScramble := func() {
			scramble = cubestate.GenerateScramble(scrambleLen)
			ctl.mu.Lock()
			ctl.phase = tsScramble
			ctl.idx = 0
			ctl.half = ""
			ctl.wrongStack = nil
			ctl.events = nil
			ctl.mu.Unlock()
			display.SetColor(segIdle)
			display.SetText("0.00")
			updateUI()
		}

		finalize := func(elapsed int64, penalty string) {
			ctl.mu.Lock()
			events := ctl.events
			ctl.events = nil
			ctl.mu.Unlock()
			solves = append(solves, Solve{
				Ms:       elapsed,
				Scramble: cubestate.ScrambleString(scramble),
				Penalty:  penalty,
				At:       time.Now().Unix(),
				Events:   events,
			})
			saveCurrent()
			selected = -1
			list.UnselectAll()
			refreshStats()
			list.Refresh()
			resetScramble()
			display.SetColor(segDone)
			display.SetText(formatMs(elapsed))
			if penalty == "DNF" {
				setHint("DNF — scramble again")
			} else {
				setHint("Solved " + formatMs(elapsed) + " — scramble again")
			}
		}

		dnfRunning := func() {
			ctl.mu.Lock()
			if ctl.phase != tsSolving {
				ctl.mu.Unlock()
				penalty("DNF")
				return
			}
			ctl.phase = tsScramble
			elapsed := time.Since(ctl.start).Milliseconds()
			ctl.stopFn()
			ctl.mu.Unlock()
			finalize(elapsed, "DNF")
		}

		startSolve := func() {
			ctl.mu.Lock()
			ctl.phase = tsSolving
			ctl.start = time.Now()
			ctl.events = nil
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
				ticker := time.NewTicker(33 * time.Millisecond)
				defer ticker.Stop()
				var det rotationDetector
				for {
					select {
					case <-sctx.Done():
						return
					case <-ticker.C:
					}
					t := time.Since(start).Milliseconds()
					label, ok := det.feed(t, a.cube.Gyro())
					if !ok {
						continue
					}
					ctl.mu.Lock()
					if ctl.phase == tsSolving {
						ctl.events = append(ctl.events, SolveEvent{T: t, Kind: "rot", Val: label})
					}
					ctl.mu.Unlock()
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
						finalize(elapsed, "")
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
				var advanced bool
				ctl.half, ctl.wrongStack, advanced = scrambleStep(scramble[ctl.idx], move, ctl.half, ctl.wrongStack)
				if advanced {
					ctl.idx++
					if ctl.idx == len(scramble) {
						ctl.phase = tsReady
					}
				}
				ctl.mu.Unlock()
				updateUI()
			case tsReady:
				ctl.mu.Unlock()
				startSolve()
			case tsSolving:
				ctl.events = append(ctl.events, SolveEvent{T: time.Since(ctl.start).Milliseconds(), Kind: "move", Val: move})
				ctl.mu.Unlock()
			default:
				ctl.mu.Unlock()
			}
		}

		targetIdx := func() int {
			if selected >= 0 && selected < len(solves) {
				return len(solves) - 1 - selected
			}
			if len(solves) > 0 {
				return len(solves) - 1
			}
			return -1
		}

		penalty = func(p string) {
			i := targetIdx()
			if i < 0 {
				return
			}
			last := &solves[i]
			if last.Penalty == p {
				last.Penalty = ""
			} else {
				last.Penalty = p
			}
			saveCurrent()
			refreshStats()
			list.Refresh()
		}

		deleteSelected := func() {
			i := targetIdx()
			if i < 0 {
				return
			}
			solves = append(solves[:i], solves[i+1:]...)
			selected = -1
			list.UnselectAll()
			saveCurrent()
			refreshStats()
			list.Refresh()
		}

		resetScramble()
		refreshStats()

		sessionSel := widget.NewSelect(sessions.names(), func(name string) {
			if name == active {
				return
			}
			sessions.Sessions[active] = solves
			sessions.Active = name
			saveSessions(sessions)
			a.showTimer()
		})
		sessionSel.Selected = active

		newSessBtn := widget.NewButton("New", func() {
			nameEntry := widget.NewEntry()
			nameEntry.SetPlaceHolder("Session name")
			dialog.ShowForm("New session", "Create", "Cancel",
				[]*widget.FormItem{widget.NewFormItem("Name", nameEntry)},
				func(ok bool) {
					name := nameEntry.Text
					if !ok || name == "" {
						return
					}
					sessions.Sessions[active] = solves
					sessions.Sessions[name] = nil
					sessions.Active = name
					saveSessions(sessions)
					a.showTimer()
				}, a.window)
		})

		delSessBtn := widget.NewButton("Delete", func() {
			if len(sessions.Sessions) <= 1 {
				return
			}
			delete(sessions.Sessions, active)
			for name := range sessions.Sessions {
				sessions.Active = name
				break
			}
			saveSessions(sessions)
			a.showTimer()
		})
		delSessBtn.Importance = widget.DangerImportance

		plus2 := widget.NewButton("+2", func() { penalty("+2") })
		plus2.Importance = widget.WarningImportance
		dnf := widget.NewButton("DNF", dnfRunning)
		dnf.Importance = widget.DangerImportance
		details := widget.NewButton("Details", showDetails)
		del := widget.NewButton("Delete", deleteSelected)
		newScr := widget.NewButton("New Scramble", resetScramble)
		newScr.Importance = widget.HighImportance
		controls := container.NewGridWithColumns(5, plus2, dnf, details, del, newScr)

		header := container.NewBorder(nil, nil,
			heading("Timer", 26),
			container.NewHBox(sessionSel, newSessBtn, delSessBtn, widget.NewButton("Back", a.showMenu)),
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
