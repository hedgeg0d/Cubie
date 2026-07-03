# Cubie

Desktop app for Bluetooth smart Rubik's cubes. Currently supports the **Weilong V10 AI**.

Modes:

- **3D Cube** — live software-rendered cube, drag with the mouse to rotate. Reflects the physical cube's turns. Press "Cube is solved (sync)" while the cube is solved to align the model.
- **Controller** — use the cube as a virtual gamepad (Linux `uinput`). Bind face turns to buttons, and bind the gyroscope to buttons (tilt gestures) or analog axes.
- **Timer** — speedcubing timer with scramble generation and ao5/ao12 stats.
- **Blind trainer** — memo/execution timing trainer for blindfold solving.

The 3D view tracks state by applying moves from the last sync point (a solved cube), so sync once before use.

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
