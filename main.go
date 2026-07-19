package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"
)

const dataFile = "data.json"
const dbFile = "goals.db"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := loadScores(dataFile); err != nil {
		log.Fatalf("failed to load %s: %v", dataFile, err)
	}

	if err := initDB(dbFile); err != nil {
		log.Fatalf("failed to open database %s: %v", dbFile, err)
	}
	defer db.Close()

	if token := getFootballDataToken(); token != "" {
		log.Println("FOOTBALL_DATA_TOKEN set -- polling football-data.org every 60s")
		go startFetchLoop(ctx, token, 60*time.Second)
		if err := fetchStandings(token); err != nil {
			log.Printf("warning: initial standings fetch failed: %v", err)
		}
		go startStandingsLoop(ctx, token, 5*time.Minute)
	} else {
		log.Println("FOOTBALL_DATA_TOKEN not set -- serving local data.json only")
	}

	if token := getBalldontlieToken(); token != "" {
		log.Println("BALLDONTLIE_API_KEY set -- polling balldontlie.io every 60s")
		if err := fetchNBAGames(token); err != nil {
			log.Printf("warning: initial NBA fetch failed: %v", err)
		}
		go startNBALoop(ctx, token, 60*time.Second)
	} else {
		log.Println("BALLDONTLIE_API_KEY not set -- skipping NBA scores")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /api/scores", scoresHandler)
	mux.HandleFunc("GET /api/scores/{id}", matchHandler)
	mux.HandleFunc("POST /api/scores", addScoreHandler)
	mux.HandleFunc("GET /api/standings", standingsHandler)
	mux.HandleFunc("GET /ws", wsHandler)
	mux.Handle("/", http.FileServer(http.Dir("static")))

	srv := &http.Server{
		Addr:    ":" + getPort(),
		Handler: mux,
	}

	go func() {
		log.Println("listening on " + srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutdown signal received, draining connections...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("warning: forced shutdown: %v", err)
	}

	if err := redisClient.Close(); err != nil {
		log.Printf("warning: failed to close redis client: %v", err)
	}

	log.Println("shutdown complete")
}
