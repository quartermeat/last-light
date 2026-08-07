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
