# Technical notes

How Cubie talks to the cube and keeps its model in sync. Aimed at contributors.

## Adding other cube models

All cube communication lives behind the `cube` package, which exposes a small surface to the rest of the app: a stream of moves, gyroscope orientation, a full-state request, and solved detection. The idea is that other cube models can sit behind that same boundary, with the model-specific protocol and decryption kept inside the driver.

For now only the MoYu Weilong V10 AI is implemented, simply because it's the only smart cube the author ([@hedgeg0d](https://github.com/hedgeg0d)) owns and can test against. If you have a different model and want it supported, contributions are very welcome — decoding the move and state packets for your cube is the main piece to fill in.

## Protocol

Weilong V10 AI BLE protocol reference: https://github.com/lukeburong/weilong-v10-ai-protocol

Traffic is encrypted with AES-128, using a key and IV derived from the cube's MAC address.

Notifications the app reads:

- `0xA5` — move events: the last five moves plus a rolling counter.
- `0xAB` — gyroscope orientation as a quaternion.

Requests the app sends:

- `0xA3` — full facelet state (18 bytes).
- `0xA1` / `0xA4` — model info / battery.

## Tracking moves

The 3D model starts from a solved cube and applies moves as they arrive. Each `0xA5` packet carries a counter and the last five moves, so a dropped or duplicated Bluetooth notification is recovered from that history instead of desyncing by a quarter turn.

That five-move window has a limit. When moves arrive faster than it can cover — rapid slice moves are the worst case, since each `M` is sent as `R'` + `L` — more than five moves can be lost between two packets and the model drifts. The status sync repairs that.

## Status sync

To fix a drift, the driver asks the cube for its full state (`0xA3`) and rebuilds the model from it. The catch is timing: a state request competes with move notifications on BLE, so firing one mid-turn can drop the move happening right then — usually the last move of a burst.

So it never syncs while you're turning. Instead:

- While turning is aggressive — the counter jumps past the five-move window, or at least `burstThreshold` (6) moves land within `burstWindow` (300 ms) — a pending-sync flag is set. Nothing is sent yet.
- Once turning settles (no move for `burstIdle`, 400 ms) and the flag is set, the worker fires one sync and clears the flag. The idle window is deliberately longer than the regrip pauses inside a fast series, so the request never lands on top of a move.
- If a move shows up while a sync is in flight, the flag is re-armed and the model is corrected again after the next pause.

Calm solving never sets the flag, so it never triggers a sync. That keeps requests rare and easy on the cube's battery. A one-shot sync also runs right after connecting (retried, since the first request after connect is often dropped) so the model matches the cube even if it wasn't solved when you connected.

Supporting details:

- All `0xA3` round-trips are serialized (`stateMu`) so the sync worker and the timer/blind poll loops don't race on the shared response channel.
- State requests time out (`stateTimeout`) instead of blocking forever if a response never comes.
- Only this path rebuilds the model (`OnResync`); ordinary solved-detection polls don't touch it.

The constants above live in `cube/cube.go`.

## Facelet reconstruction

`0xA3` returns 18 bytes: six faces, eight non-center stickers each, three bits per sticker (MSB first), value 0–5. Face order is `F B U D L R`, and the values map to `0 Green, 1 Blue, 2 White, 3 Yellow, 4 Orange, 5 Red` in WCA orientation (green front, white top). In a solved state, face group *g* is uniformly value *g*.

`cubestate.ModelFromState` decodes this into the same 54-sticker model the move stream builds. The lookup tables in `cubestate/facelet.go` (`groupToFace`, `faceStickerPos` for the per-face byte order, and `colorByValue`) were worked out against a physical Weilong V10 AI (WCU_MY32) and are pinned by golden tests in `cubestate/facelet_test.go`.
