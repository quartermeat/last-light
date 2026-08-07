package main

import "sort"

func recordScore(scores []int, score int) []int {
	if score <= 0 {
		return scores
	}
	scores = append(scores, score)
	sort.Sort(sort.Reverse(sort.IntSlice(scores)))
	if len(scores) > 5 {
		scores = scores[:5]
	}
	saveScores(scores)
	return scores
}
