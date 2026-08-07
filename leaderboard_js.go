//go:build js && wasm

package main

import (
	"encoding/json"
	"strings"
	"syscall/js"
)

func scoreStorageKey() string {
	parts := strings.Split(version, ".")
	if len(parts) >= 2 {
		return "last-light-leaderboard-" + parts[0] + "." + parts[1]
	}
	return "last-light-leaderboard-" + version
}

func loadScores() []Score {
	value := js.Global().Get("localStorage").Call("getItem", scoreStorageKey())
	if value.IsNull() || value.IsUndefined() {
		return nil
	}
	var scores []Score
	if json.Unmarshal([]byte(value.String()), &scores) != nil {
		return nil
	}
	return scores
}
func saveScores(scores []Score) {
	data, _ := json.Marshal(scores)
	js.Global().Get("localStorage").Call("setItem", scoreStorageKey(), string(data))
}
func startLeaderboardSync(g *Game) {
	responseJSON := js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		if len(args) == 0 {
			return nil
		}
		return args[0].Call("json")
	})
	applyScores := js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		defer responseJSON.Release()
		if len(args) == 0 {
			return nil
		}
		var scores []Score
		if json.Unmarshal([]byte(args[0].String()), &scores) == nil {
			g.scores = scores
			if g.submittedScore.Name == "" {
				for _, score := range scores {
					if score.Name == "NEO" {
						g.submittedScore = score
						break
					}
				}
			}
			if g.submittedScore.Name != "" {
				g.submittedRank = scoreRank(g.scores, g.submittedScore)
			}
			saveScores(scores)
		}
		return nil
	})
	promise := js.Global().Call("fetch", "/api/leaderboard")
	promise = promise.Call("then", responseJSON)
	promise.Call("then", applyScores)
}
func getInitials() string {
	value := js.Global().Call("prompt", "NEW LEADERBOARD SCORE\nEnter 3 characters:", "YOU")
	if value.IsNull() || value.IsUndefined() {
		return "YOU"
	}
	name := strings.ToUpper(value.String())
	name = strings.Map(func(r rune) rune {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return -1
	}, name)
	if len(name) < 3 {
		name += "___"
	}
	return name[:3]
}
