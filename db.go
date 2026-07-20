package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"time"

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
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS topscorers_cache (
			cache_key TEXT PRIMARY KEY,
			stats_json TEXT NOT NULL,
			cached_at TEXT NOT NULL
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
		INSERT INTO goal_cache (match_id, goals_json) VALUES (?, ?) 
		ON CONFLICT(match_id) DO UPDATE SET goals_json = excluded.goals_json
		`, matchID, string(data))
	return err
}

func getCachedTopScorers(cacheKey string, maxAge time.Duration) ([]PlayerStat, bool) {
	var statsJSON, cachedAtStr string
	err := db.QueryRow(`SELECT stats_json, cached_at FROM topscorers_cache WHERE cache_key = ?`, cacheKey).
		Scan(&statsJSON, &cachedAtStr)
	if err != nil {
		return nil, false
	}

	cachedAt, err := time.Parse(time.RFC3339, cachedAtStr)
	if err != nil {
		log.Printf("warning: corrupt cached_at for %s: %v", cacheKey, err)
		return nil, false
	}
	if time.Since(cachedAt) > maxAge {
		return nil, false
	}

	var stats []PlayerStat
	if err := json.Unmarshal([]byte(statsJSON), &stats); err != nil {
		log.Printf("warning: corrupt cached top scorers for %s: %v", cacheKey, err)
		return nil, false
	}
	return stats, true
}

func setCachedTopScorers(cacheKey string, stats []PlayerStat) error {
	data, err := json.Marshal(stats)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
		INSERT INTO topscorers_cache (cache_key, stats_json, cached_at) VALUES (?, ?, ?)
		ON CONFLICT(cache_key) DO UPDATE SET stats_json = excluded.stats_json, cached_at = excluded.cached_at
	`, cacheKey, string(data), time.Now().UTC().Format(time.RFC3339))
	return err
}
