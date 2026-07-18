package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/redis/go-redis/v9"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		http.Error(w, "redis unreachable: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.Write([]byte("ok"))
}

func scoresHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	cached, err := redisClient.Get(ctx, scoresCacheKey).Result()
	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.Write([]byte(cached))
		return
	}
	if err != redis.Nil {
		log.Printf("warning: redis GET failed: %v", err)
	}

	scoresMu.Lock()
	data, err := json.Marshal(scores)
	scoresMu.Unlock()
	if err != nil {
		http.Error(w, "failed to encode scores", http.StatusInternalServerError)
		return
	}

	if err := redisClient.Set(ctx, scoresCacheKey, data, scoresCacheTTL).Err(); err != nil {
		log.Printf("warning: redis SET failed: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	w.Write(data)
}

func matchHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	scoresMu.Lock()
	var found Score
	foundOK := false
	for _, s := range scores {
		if s.ID == id {
			found = s
			foundOK = true
			break
		}
	}
	scoresMu.Unlock()

	if !foundOK {
		http.NotFound(w, r)
		return
	}

	detail := MatchDetail{Score: found}

	if !strings.HasPrefix(id, "nba-") {
		if token := getApiFootballToken(); token != "" {
			goals, err := fetchMatchGoals(token, id)
			if err != nil {
				log.Printf("warning: failed to fetch goals for match %s %v", id, err)
			} else {
				detail.Goals = goals
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(detail)
}

func addScoreHandler(w http.ResponseWriter, r *http.Request) {
	var s Score
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	scoresMu.Lock()
	scores = append(scores, s)
	err := saveScores(dataFile)
	scoresMu.Unlock()

	if err != nil {
		log.Printf("warning: failed to persist scores: %v", err)
	}

	if err := redisClient.Del(context.Background(), scoresCacheKey).Err(); err != nil {
		log.Printf("warning: failed to invalidate cache: %v", err)
	}

	hub.broadcast(map[string]interface{}{"event": "score_added", "score": s})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(s)
}

func standingsHandler(w http.ResponseWriter, r *http.Request) {
	standingsMu.Lock()
	defer standingsMu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(standings)
}
