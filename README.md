# Cubie

Cubie is a desktop app for Bluetooth smart cubes. Connect your cube and use it as a live 3D viewer, a speedcubing timer, a blindfold trainer, or a game controller.

<p align="center">
  <img src="https://github.com/user-attachments/assets/4785fa13-de47-427f-9802-2ed09bdb01ee" width="600" alt="Live 3D viewer">
</p>

<p align="center">
  <img src="https://github.com/user-attachments/assets/11531199-c1ff-46a4-af9f-69e18fd6d7ce" width="49%" alt="Speedcubing timer">
  <img src="https://github.com/user-attachments/assets/9bcf33e6-b092-4ea7-860c-0084933862f3" width="49%" alt="Blindfold trainer">
</p>

## Supported cubes

- MoYu Weilong V10 AI

More models are planned. If you own another smart cube and want it supported, see [TECHNICAL.md](TECHNICAL.md) — help is welcome.

## Modes

**3D Cube** — a 3D cube on screen that mirrors your physical cube as you turn it. Drag with the mouse to look around. If the model ever drifts out of sync, press "Mark solved (sync)" while your cube is solved to line it back up.

**Timer** — a speedcubing timer with scramble generation and ao5/ao12 averages. Scrambles can include double turns like `U2`; you can turn those either way, and after the first quarter the app shows which single turn finishes it. Pick a solve from the list to add +2/DNF, delete it, or open Details for the scramble, TPS, and a reconstruction of the solve (both as performed, with cube rotations, and as a plain move sequence).

**Blind trainer** — memo and execution timer for blindfold solving, with the same guided scrambler as the timer (turn by turn, with undo hints for wrong moves). Once the cube is scrambled it shows a ready memo in your lettering scheme — corners first, then edges, Old-Pochmann style. The Lettering button opens the scheme editor.

**Lettering** — assign a letter or symbol to every sticker on a flat cube net. Click a sticker and type; any alphabet works (Cyrillic and so on). Schemes are saved as named profiles, defaulting to Speffz.

**Controller** — turn the cube into a virtual gamepad (Linux only). Map face turns to buttons and the gyroscope to buttons or analog sticks. More on this below.

## Requirements

- Linux (controller mode needs `/dev/uinput`)
- A Bluetooth adapter
- Go 1.24 or newer

## Building and running

```sh
go build ./...
go run ./main
```

On the first screen, pick your cube model and connect. You can either hit Scan to list nearby Bluetooth devices (strongest signal first — tap one to fill in its address) or paste the cube's Bluetooth MAC address yourself.

Controller mode writes to `/dev/uinput`, so you need access to it: either run with enough privileges or add a udev rule for your user.

## Controller builder

Bindings live in named profiles (top-right selector, with New/Delete), so you can keep a separate layout per game. Everything is saved to `controller_profiles.json`.

The Controller screen has a few tabs:

- **Buttons** — map each of the 12 face turns to a gamepad button and set how long a tap is held.
- **Gyro tilts** — bind tilt gestures to buttons. Pick an axis (pitch, roll, or yaw), a direction, a button, and a threshold in degrees. Hold keeps the button down while the cube is tilted past the threshold; Tap fires once when you cross it. A release factor adds a little hysteresis so buttons don't chatter.
- **Gyro axes** — map a rotation angle to an analog output (stick X/Y or triggers), with a deadzone, a range, and an invert toggle.
- **Live** — an orientation sphere, live pitch/roll/yaw readouts, smoothing sliders, and a Calibrate neutral button.

A gamepad graphic sits under the tabs and lights up as inputs fire, so you can see exactly what's happening while you tune.

The gyroscope reports absolute orientation, so tilts are measured against a neutral pose. Hold the cube however you rest it and press Calibrate neutral first. Press Save to write everything to disk.

## Technical details

Protocol notes, how the app handles dropped moves, and how to add other cube models are in [TECHNICAL.md](TECHNICAL.md).
