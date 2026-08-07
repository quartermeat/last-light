//go:build js && wasm

package main

import (
	"encoding/json"
	"syscall/js"
)

type storedRun struct {
	Hours  int        `json:"hours"`
	Events []LogEvent `json:"events"`
}

func saveRunLog(events []LogEvent, hours int) {
	storage := js.Global().Get("localStorage")
	var history []storedRun
	value := storage.Call("getItem", "last-light-run-history")
	if !value.IsNull() && !value.IsUndefined() {
		_ = json.Unmarshal([]byte(value.String()), &history)
	}
	history = append(history, storedRun{Hours: hours, Events: events})
	if len(history) > 10 {
		history = history[len(history)-10:]
	}
	data, _ := json.Marshal(history)
	storage.Call("setItem", "last-light-run-history", string(data))
	latest, _ := json.Marshal(events)
	storage.Call("setItem", "last-light-latest-run-log", string(latest))
	if downloader := js.Global().Get("lastLightDownloadLog"); !downloader.IsUndefined() { downloader.Invoke(string(latest)) }
}

