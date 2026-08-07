# Last Light

A small top-down pixel survival-sim prototype built with Go and Ebitengine, targeting WASM first. The presentation is a compact wilderness map with a classic action-adventure feel: explore, find resources, and make it back to camp before the weather turns.

## Controls

- WASD / arrow keys: move
- E: gather nearby wood or food (guaranteed when available)
- P: fish at the shoreline (55% success chance)
- H: hunt nearby moving game (50% success chance)
- F: light a fire at camp (2 wood)
- B: build shelter at camp (6 wood)
- Q: eat
- R: restart

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




