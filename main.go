package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
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

func loadScores(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, &scores)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}

func scoresHandler(w http.ResponseWriter, r *http.Request) {
	scoresMu.Lock()
	defer scoresMu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scores)
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
	scoresMu.Unlock()

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

func main() {
	if err := loadScores("data.json"); err != nil {
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
