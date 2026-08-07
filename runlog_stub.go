//go:build !js || !wasm

package main

func saveRunLog([]LogEvent, int, string, []ReplayFrame) {}
