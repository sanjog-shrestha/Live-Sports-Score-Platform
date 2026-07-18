package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"sync"
)

var scores []Score
var scoresMu sync.Mutex
var footballScores []Score
var nbaScores []Score

func loadScores(path string) error {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		log.Printf("%s not found - starting with an empty score list", path)
		scores = []Score{}
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, &scores)
}

func saveScores(path string) error {
	data, err := json.MarshalIndent(scores, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func mergeScores() {
	scoresMu.Lock()
	merged := make([]Score, 0, len(footballScores)+len(nbaScores))
	merged = append(merged, footballScores...)
	merged = append(merged, nbaScores...)
	scores = merged
	err := saveScores(dataFile)
	scoresMu.Unlock()

	if err != nil {
		log.Printf("warning: failed to persist merged scores: %v", err)
	}

	if err := redisClient.Del(context.Background(), scoresCacheKey).Err(); err != nil {
		log.Printf("warning: failed to invalidate cache: %v", err)
	}

	hub.broadcast(map[string]interface{}{"event": "scores_refreshed"})
}
