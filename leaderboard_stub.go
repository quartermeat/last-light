//go:build !js || !wasm

package main

func loadScores() []Score { return nil }
func saveScores([]Score)  {}
func getInitials() string { return "YOU" }
