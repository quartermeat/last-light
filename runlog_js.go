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

func saveRunLog(events []LogEvent, hours int, name string) {
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
	payload, _ := json.Marshal(struct { Name string `json:"name"`; Hours int `json:"hours"`; Events []LogEvent `json:"events"` }{name, hours, events})`r`n`options := map[string]interface{}{`r`n`t"method": "POST",`r`n`t"headers": map[string]interface{}{"Content-Type": "application/json"},`r`n`t"body": string(payload),`r`n}`r`n`promise := js.Global().Call("fetch", "/api/runs", js.ValueOf(options))`r`n`promise.Call("catch", js.FuncOf(func(js.Value, []js.Value) interface{} { return nil }))
}


