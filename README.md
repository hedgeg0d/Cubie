# Cubie

Desktop app for Bluetooth smart Rubik's cubes. Currently supports the **Weilong V10 AI**.

Planned modes:

- **Controller** — use the cube as a virtual gamepad (Linux `uinput`). *In progress.*
- **Timer** — speedcubing timer with scramble generation and ao5/ao12 stats. *Planned.*
- **Blind trainer** — memo/execution timing trainer for blindfold solving. *Planned.*

## Requirements

- Linux (controller mode uses `/dev/uinput`; requires write access to it)
- A Bluetooth adapter
- Go 1.24+

## Build & run

```sh
go build ./...
go run ./main
```

On first launch enter your cube's Bluetooth MAC address and pick the model, then Connect.

Controller mode needs access to `/dev/uinput`. Either run with sufficient privileges or add a udev rule granting your user access.

## Protocol

Weilong V10 AI BLE protocol reference: https://github.com/lukeburong/weilong-v10-ai-protocol

Encryption is AES-128 with a key/IV derived from the cube's MAC address.
