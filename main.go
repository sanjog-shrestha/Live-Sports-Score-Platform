package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type Score struct {
	ID        string `json:"id"`
	Home      string `json:"home"`
	Away      string `json:"away"`
	HomeScore int    `json:"homeScore"`
	AwayScore int    `json:"awayScore"`
}

var scores []Score
var scoresMu sync.Mutex
var redisClient = redis.NewClient(&redis.Options{
	Addr: getRedisAddr(),
})

const dataFile = "data.json"
const scoresCacheKey = "scores:all"
const scoresCacheTTL = 30 * time.Second

func loadScores(path string) error {
	raw, err := os.ReadFile(path)
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

func healthHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		http.Error(w, "redis unavailable: "+err.Error(), http.StatusServiceUnavailable)
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

	if err := redisClient.Set(ctx, scoresCacheKey, data, scoresCacheTTL); err != nil {
		log.Printf("warning: redis SET failed: %v", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	w.Write(data)
}

func matchHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	scoresMu.Lock()
	defer scoresMu.Unlock()
	for _, s := range scores {
		if s.ID == id {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(s)
			return
		}
	}
	http.NotFound(w, r)
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
		log.Printf("warning: failed to invalidate check %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(s)
}

func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return port
}

func getRedisAddr() string {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	return addr
}

func main() {
	if err := loadScores(dataFile); err != nil {
		log.Fatalf("failed to load data.json: %v", err)
	}

	http.HandleFunc("GET /health", healthHandler)
	http.HandleFunc("GET /api/scores", scoresHandler)
	http.HandleFunc("GET /api/scores/{id}", matchHandler)
	http.HandleFunc("POST /api/scores", addScoreHandler)

	port := getPort()
	log.Println("listening on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
