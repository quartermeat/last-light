//go:build js && wasm

package main

import (
	"encoding/json"
	"syscall/js"
)

const scoreStorageKey = "last-light-leaderboard"

func loadScores() []int {
	value := js.Global().Get("localStorage").Call("getItem", scoreStorageKey)
	if value.IsNull() || value.IsUndefined() {
		return nil
	}
	var scores []int
	if json.Unmarshal([]byte(value.String()), &scores) != nil {
		return nil
	}
	return scores
}

func saveScores(scores []int) {
	data, _ := json.Marshal(scores)
	js.Global().Get("localStorage").Call("setItem", scoreStorageKey, string(data))
}
