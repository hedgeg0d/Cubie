# Cubie

Desktop app for Bluetooth smart Rubik's cubes. Currently supports the **Weilong V10 AI**.

Modes:

- **3D Cube** — live software-rendered cube, drag with the mouse to rotate. Reflects the physical cube's turns. Press "Cube is solved (sync)" while the cube is solved to align the model.
- **Controller** — use the cube as a virtual gamepad (Linux `uinput`). Bind face turns to buttons, and bind the gyroscope to buttons (tilt gestures) or analog axes.
- **Timer** — speedcubing timer with scramble generation (including double turns like `U2`) and ao5/ao12 stats. Double turns can be executed either direction — after the first quarter the guidance shows the single turn (`U` or `U'`) needed to finish. Select a solve in the list to apply +2/DNF, delete it, or open **Details** (also via double-click) showing the scramble, TPS, and two reconstructions: **as performed** — moves interleaved with the cube rotations (x/y/z) at the point they happened, with subsequent moves re-expressed in the solver's frame — and **rotation-agnostic** — the raw move sequence in the cube's fixed frame. Cube rotations are inferred from the gyroscope during the solve, so their axis/direction is approximate.
- **Blind trainer** — memo/execution timing trainer for blindfold solving, with the same guided scrambler as the timer (turn-by-turn, wrong-move undo hints, double turns either direction). Start Memo unlocks once the cube is scrambled. It shows a ready memo for the scramble in your active lettering scheme — corners first, then edges (Old-Pochmann-style sticker cycles; corner buffer UBL, edge buffer UF). The **Lettering** button opens the scheme editor.
- **Lettering** — assign a memo symbol to every non-center sticker on a flat cube net; click a sticker and type (any alphabet, e.g. Cyrillic). Symbols update live on the net. Multiple named schemes are saved as profiles (defaults to Speffz) in `lettering_profiles.json`. Opened from the blind trainer.

The 3D view tracks state by applying moves from the last sync point (a solved cube), so sync once before use. Move packets carry a counter and the last five moves, so dropped or duplicated Bluetooth notifications are recovered from the history instead of desyncing the model by a quarter turn. When moves arrive faster than that five-move window can cover — e.g. rapid slice (`M`) moves, each sent as `R'`+`L` — the app requests the cube's full state and rebuilds the model from ground truth (see below).

## Requirements

- Linux (controller mode uses `/dev/uinput`; requires write access to it)
- A Bluetooth adapter
- Go 1.24+

## Build & run

```sh
go build ./...
go run ./main
```

On first launch pick the model and connect. You can either **Scan for cubes** (lists nearby named Bluetooth devices, strongest signal first — tap one to fill its address) or type the Bluetooth MAC address directly, then Connect.

Controller mode needs access to `/dev/uinput`. Either run with sufficient privileges or add a udev rule granting your user access.

### Controller builder

Bindings are grouped into named **profiles** (selector in the top-right, with New/Delete). Switch profiles to keep separate layouts per game; the active profile and all profiles are saved to `controller_profiles.json`.

The Controller screen is organised into tabs:

- **Buttons** — map each of the 12 face turns to a gamepad button, and set the tap hold time.
- **Gyro tilts** — bind tilt gestures to buttons. Each binding picks an axis (Pitch/Roll/Yaw), a direction, a target button, a mode, and an activation threshold in degrees. *Hold* keeps the button pressed while the cube is tilted past the threshold; *Tap* fires a single press when the threshold is crossed. A release factor adds hysteresis so buttons don't chatter near the threshold.
- **Gyro axes** — map a rotation angle onto an analog target (Left/Right stick X/Y, LT, RT). Each binding has a deadzone (center dead band, degrees), a range (angle mapped to full deflection), and an invert toggle.
- **Live** — a live orientation sphere, current Pitch/Roll/Yaw and axis outputs, smoothing/release-factor sliders, and a **Calibrate neutral** button.

A live input monitor (a gamepad graphic) sits below the tabs on every tab: buttons glow when triggered, and the sticks and triggers move with the bound axes, so you can watch exactly what fires while tuning thresholds and calibrating.

The gyroscope reports absolute orientation, so tilts are measured relative to a calibrated neutral pose. Hold the cube in your rest position and press **Calibrate neutral** first; the neutral is saved with the rest of the bindings. Press **Save** to persist everything to `controller_profiles.json`.

## Protocol

Weilong V10 AI BLE protocol reference: https://github.com/lukeburong/weilong-v10-ai-protocol

Encryption is AES-128 with a key/IV derived from the cube's MAC address.

### Status sync

The move stream (`0xA5`) drops packets under load; a gap wider than the five-move
history cannot be reconstructed. To repair a desync the driver requests the cube's full
state (`0xA3`) and rebuilds the model from it — but never *during* turning, since a
state request competes with the move notifications on BLE and can drop the last move.

Instead a flag is used: while turning is aggressive (counter jumps past the five-move
window, or ≥ `burstThreshold` (6) moves within a sliding `burstWindow` (300 ms)) a
pending-sync flag is set — no request is sent yet. The background worker only fires the
sync once turning has settled: no move for `burstIdle` (400 ms) *and* the flag is set,
after which the flag clears. The idle window is deliberately longer than the regrip
gaps within a fast series, so the request never coincides with a move packet. As a
safety net, if a move does arrive while a sync is in flight, the flag is re-armed so the
model is corrected again after the next settle. This means very few requests
(energy-friendly), no lost moves, and complete state by the time it runs.

Calm solving never sets the flag, so it never triggers a sync. A one-shot sync also
runs right after connecting (retried until the cube answers, since the first request
after connect can be dropped), so the model reflects the cube's real state even if it
wasn't solved when connected. State requests time out (`stateTimeout`) instead of
blocking forever if a response is lost. Only this path rebuilds the model (`OnResync`); ordinary
solved-detection polls do not. All `0xA3` round-trips
are serialized (`stateMu`) so the worker and the timer/blind poll loops never race on
the shared response channel. Constants live in `cube/cube.go`.

### Facelet-state reconstruction

`0xA3` returns 18 bytes: six faces × eight non-center stickers × 3 bits (MSB-first),
each value 0–5. The face order is `F B U D L R` and color values are
`0 Green, 1 Blue, 2 White, 3 Yellow, 4 Orange, 5 Red` (WCA green-front/white-top);
in the solved state face group *g* is uniformly value *g*. `cubestate.ModelFromState`
decodes these into the same 54-sticker model the move stream builds. The tables in
`cubestate/facelet.go` (`groupToFace`, `faceStickerPos` — per-face byte order,
row-major — and `colorByValue`) were locked against a physical Weilong V10 AI
(WCU_MY32) and are pinned by golden tests in `cubestate/facelet_test.go`.
