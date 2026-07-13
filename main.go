package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

type Score struct {
	ID        string `json:"id"`
	Home      string `json:"home"`
	Away      string `json:"away"`
	HomeScore int    `json:"homeScore"`
	AwayScore int    `json:"awayScore"`
}

var scores []Score

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
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scores)
}

func matchHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	for _, s := range scores {
		if s.ID == id {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(s)
			return
		}
	}
	http.NotFound(w, r)
}

func main() {
	if err := loadScores("data.json"); err != nil {
		log.Fatalf("failed to load data.json: %v", err)
	}

	http.HandleFunc("GET /health", healthHandler)
	http.HandleFunc("GET /api/scores", scoresHandler)
	http.HandleFunc("GET /api/scores/{id}", matchHandler)

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
