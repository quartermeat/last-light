package main

type LogEvent struct {
	Hour    float64 `json:"hour"`
	Kind    string  `json:"kind"`
	Details string  `json:"details"`
}
