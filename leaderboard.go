package main

import "sort"

type Score struct {
	Name  string `json:"name"`
	Hours int    `json:"hours"`
}

func qualifiesScore(scores []Score, hours int) bool {
	return hours > 0 && (len(scores) < 100 || hours > scores[len(scores)-1].Hours)
}
func recordScore(scores []Score, score int, name string) []Score {
	if !qualifiesScore(scores, score) {
		return scores
	}
	scores = append(scores, Score{Name: name, Hours: score})
	sort.Slice(scores, func(i, j int) bool { return scores[i].Hours > scores[j].Hours })
	if len(scores) > 100 {
		scores = scores[:100]
	}
	saveScores(scores)
	return scores
}

func scoreRank(scores []Score, target Score) int {
	for i, score := range scores {
		if score == target {
			return i + 1
		}
	}
	return 0
}
