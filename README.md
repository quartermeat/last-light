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

Nutrition is simulated behind the scenes: calories, protein, carbohydrates, fat, and fiber are added by meals and consumed by movement and actions. The player only sees the hunger result. Risky vegetation can cause sickness when automatically eaten.

## Leaderboard

The SQLite backend is the leaderboard source of truth. The browser keeps a local cache for startup, then synchronizes the top 100 from `GET /api/leaderboard`. Qualifying runs use a classic three-character initials entry, and scores are measured in in-game hours. Completed runs retain structured gameplay logs and replay snapshots. The 15-minute clock starts a fresh leaderboard season.

## Strategy runner

Run thousands of headless strategy attempts and print the best survival result:

```powershell
go run ./cmd/runner -trials 10000 -seed 1 -address http://127.0.0.1:8080 -submit
```

Submitted tester runs use the initials `NEO`. Add `-fishing=false` to measure a no-fishing strategy. On the run-over screen, keys `1` through `6` replay only the visible leaderboard rows at 8x speed.

Use `-submit-count 100` to fill the SQL leaderboard with the top 100 simulated runs from the trial batch.

## Single-process local run

The optional Go/SQLite backend can serve both the game and API from one process:

```powershell
.\scripts\start-local.ps1
```

Open `http://127.0.0.1:8080/`. Use `-Address "127.0.0.1:8090"` if the port is occupied. The service exposes `GET /api/leaderboard`, `POST /api/runs`, and `GET /healthz`. Set `LAST_LIGHT_DB` to choose the SQLite file path.


