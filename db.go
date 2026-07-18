package main

import (
	"database/sql"
	"encoding/json"
	"log"

	_ "modernc.org/sqlite"
)

var db *sql.DB

func initDB(path string) error {
	var err error
	db, err = sql.Open("sqlite", path)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS goal_cache (
		match_id TEXT PRIMARY KEY,
		goals_json TEXT NOT NULL,
		cached_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

func getCachedGoals(matchID string) ([]Goal, bool) {
	var goalsJSON string
	err := db.QueryRow(`SELECT goals_json FROM goal_cache WHERE match_id = ?`, matchID).Scan(&goalsJSON)
	if err != nil {
		return nil, false
	}

	var goals []Goal
	if err := json.Unmarshal([]byte(goalsJSON), &goals); err != nil {
		log.Printf("warning: corrupt cached goals for %s: %v", matchID, err)
		return nil, false
	}
	return goals, true
}

func setCachedGoals(matchID string, goals []Goal) error {
	data, err := json.Marshal(goals)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		INSERT INTO goal_cache (match_id, goals_json) VALUES (?, ?) ON CONFLICT(match_id) DO UPDATE SET goals_json = excluded.goals_json
		`, matchID, string(data))
	return err
}
