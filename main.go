package main

import (
	"log"
	"net/http"
	"time"
)

const dataFile = "data.json"
const dBFile = "goals.db"

func main() {
	if err := loadScores(dataFile); err != nil {
		log.Fatalf("failed to load %s: %v", dataFile, err)
	}

	if err := initDB(dBFile); err != nil {
		log.Fatalf("failed to open database %s: %v", dBFile, err)
	}

	if token := getFootballDataToken(); token != "" {
		log.Println("FOOTBALL_DATA_TOKEN set -- polling football-data.org ever 60s")
		go startFetchLoop(token, 60*time.Second)
		if err := fetchStandings(token); err != nil {
			log.Printf("warning: initial standings fetch failed: %v", err)
		}
		go startStandingsLoop(token, 5*time.Minute)
	} else {
		log.Println("FOOTBALL_DATA_TOKEN not set -- serving local data.json only")
	}

	if token := getBalldontlieToken(); token != "" {
		log.Println("BALLDONTLIE_API_KEY set -- polling balldontlie.io ever 60s")
		if err := fetchNBAGames(token); err != nil {
			log.Printf("warning: initial NBA fetch failed: %v", err)
		}
		go startNBALoop(token, 60*time.Second)
	} else {
		log.Println("BALLDONTLIE_API_KEY not set -- skipping NBA scores")
	}

	http.HandleFunc("GET /health", healthHandler)
	http.HandleFunc("GET /api/scores", scoresHandler)
	http.HandleFunc("GET /api/scores/{id}", matchHandler)
	http.HandleFunc("POST /api/scores", addScoreHandler)
	http.HandleFunc("GET /api/standings", standingsHandler)
	http.HandleFunc("GET /ws", wsHandler)
	http.Handle("/", http.FileServer(http.Dir("static")))

	port := getPort()
	log.Println("listening on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
