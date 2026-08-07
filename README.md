# Last Light

A small top-down pixel survival-sim prototype built with Go and Ebitengine, targeting WASM first. The presentation is a compact wilderness map with a classic action-adventure feel: explore, find resources, and make it back to camp before the weather turns.

## Controls

- WASD / arrow keys: move
- Q: context interaction: gather, hunt, or fish based on proximity
- 1: light a fire at camp (2 wood)
- 2: build shelter at camp (6 wood)
- Esc: restart

Numbered keys are the learned-skill slots; fire and shelter currently occupy 1 and 2.

## Run locally

```powershell
$env:GOOS="js"; $env:GOARCH="wasm"; go build -buildvcs=false -o web/game.wasm .
Copy-Item "$(go env GOROOT)\lib\wasm\wasm_exec.js" web\wasm_exec.js
python -m http.server 8080 -d web
```

Open `http://localhost:8080`.

## Versioning

Versions follow a three-part release rule:

- Patch (`v0.0.x`): every new build, including fixes, tuning, and balance changes.
- Feature (`v0.x.0`): a meaningful new feature; reset the patch number to zero.
- Shareable (`v1.0.0` or later): promoted when Jeremy feels the game is ready to share with someone.











