//go:build !js || !wasm

package main

func loadScores() []int { return nil }
func saveScores([]int)  {}
