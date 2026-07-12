package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type Score struct {
	Home      string `json:"home"`
	Away      string `json:"away"`
	HomeScore int    `json:"homeScore"`
	AwayScore int    `json:"awayScore"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}

func scoresHandler(w http.ResponseWriter, r *http.Request) {
	scores := []Score{
		{Home: "Riverside FC", Away: "Harbor United", HomeScore: 1, AwayScore: 0},
		{Home: "Summit Athletic", Away: "Ironclad City", HomeScore: 2, AwayScore: 2},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scores)
}

func main() {
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/api/scores", scoresHandler)

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
