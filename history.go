package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type MatchHistoryRow struct {
	MatchID     string `json:"matchId"`
	Home        string `json:"home"`
	Away        string `json:"away"`
	HomeScore   int    `json:"homeScore"`
	AwayScore   int    `json:"awayScore"`
	Status      string `json:"status"`
	Competition string `json:"competition"`
	Date        string `json:"date"`
	UpdatedAt   string `json:"updatedAt"`
}

func recordMatchHistory(s Score) error {
	_, err := db.Exec(`
		INSERT INTO match_history (match_id, home, away, home_score, away_score, status, competition, match_date, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(match_id) DO UPDATE SET
			home_score = excluded.home_score,
			away_score = excluded.away_score,
			status = excluded.status,
			updated_at = excluded.updated_at
		`, s.ID, s.Home, s.Away, s.HomeScore, s.AwayScore, s.Status, s.Competition, s.Date, time.Now().UTC().Format(time.RFC3339))
	return err
}

func recordMatchHistoryBatch(scoresList []Score) {
	for _, s := range scoresList {
		if err := recordMatchHistory(s); err != nil {
			log.Printf("warning: failed to record match history for %s: %v", s.ID, err)
		}
	}
}

func getMatchHistory(limit int) ([]MatchHistoryRow, error) {
	rows, err := db.Query(`
		SELECT match_id, home, away, home_score, away_score, status, competition, match_date, updated_at
		FROM match_history
		ORDER BY updated_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history := make([]MatchHistoryRow, 0)
	for rows.Next() {
		var h MatchHistoryRow
		if err := rows.Scan(&h.MatchID, &h.Home, &h.Away, &h.HomeScore, &h.AwayScore, &h.Status, &h.Competition, &h.Date, &h.UpdatedAt); err != nil {
			return nil, err
		}
		history = append(history, h)
	}
	return history, rows.Err()
}

func matchHistoryHandler(w http.ResponseWriter, r *http.Request) {
	history, err := getMatchHistory(100)
	if err != nil {
		log.Printf("warning: failed to query match history: %v", err)
		http.Error(w, "failed to load match history", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

type GoalHistoryRow struct {
	MatchID     string `json:"matchId"`
	Minute      int    `json:"minute"`
	Team        string `json:"team"`
	Scorer      string `json:"scorer"`
	Competition string `json:"competition"`
	RecordedAt  string `json:"recordedAt"`
}

func recordGoalHistory(matchID, competition string, goals []Goal) error {
	recordedAt := time.Now().UTC().Format(time.RFC3339)
	for _, g := range goals {
		_, err := db.Exec(`
			INSERT OR IGNORE INTO goal_history (match_id, minute, team, scorer, competition, recorded_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, matchID, g.Minute, g.Team, g.Scorer, competition, recordedAt)
		if err != nil {
			return err
		}
	}
	return nil
}

func getGoalHistory(limit int) ([]GoalHistoryRow, error) {
	rows, err := db.Query(`
		SELECT match_id, minute, team, scorer, competition, recorded_at
		FROM goal_history
		ORDER BY recorded_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history := make([]GoalHistoryRow, 0)
	for rows.Next() {
		var g GoalHistoryRow
		if err := rows.Scan(&g.MatchID, &g.Minute, &g.Team, &g.Scorer, &g.Competition, &g.RecordedAt); err != nil {
			return nil, err
		}
		history = append(history, g)
	}
	return history, rows.Err()
}

func goalHistoryHandler(w http.ResponseWriter, r *http.Request) {
	history, err := getGoalHistory(200)
	if err != nil {
		log.Printf("warning: failed to query goal history: %v", err)
		http.Error(w, "failed to load goal history", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

type StandingsSnapshotRow struct {
	Competition string        `json:"competition"`
	Season      string        `json:"season"`
	RecordedAt  string        `json:"recordedAt"`
	Standings   []StandingRow `json:"standings"`
}

func getLatestStandingsSnapshot(competition, season string) (string, error) {
	var data string
	err := db.QueryRow(`
		SELECT standings_json FROM standings_snapshots
		WHERE competition = ? AND season = ?
		ORDER BY recorded_at DESC
		LIMIT 1 
	`, competition, season).Scan(&data)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return data, nil
}

func recordStandingsSnapshot(competition, season string, rows []StandingRow) error {
	data, err := json.Marshal(rows)
	if err != nil {
		return err
	}

	latest, err := getLatestStandingsSnapshot(competition, season)
	if err != nil {
		return err
	}
	if latest == string(data) {
		return nil
	}

	_, err = db.Exec(`
		INSERT INTO standings_snapshots (competition, season, standings_json, recorded_at)
		VALUES (?, ?, ?, ?)
	`, competition, season, string(data), time.Now().UTC().Format(time.RFC3339))
	return err
}

func getStandingsSnapshots(limit int) ([]StandingsSnapshotRow, error) {
	rows, err := db.Query(`
		SELECT competition, season, standings_json, recorded_at
		FROM standings_snapshots
		ORDER BY recorded_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]StandingsSnapshotRow, 0)
	for rows.Next() {
		var comp, season, standingsJSON, recordedAt string
		if err := rows.Scan(&comp, &season, &standingsJSON, &recordedAt); err != nil {
			return nil, err
		}
		var parsed []StandingRow
		if err := json.Unmarshal([]byte(standingsJSON), &parsed); err != nil {
			log.Printf("warning: corrupt standings snapshot at %s: %v", recordedAt, err)
			continue
		}
		result = append(result, StandingsSnapshotRow{
			Competition: comp,
			Season:      season,
			RecordedAt:  recordedAt,
			Standings:   parsed,
		})
	}
	return result, rows.Err()
}

func standingsSnapshotsHandler(w http.ResponseWriter, r *http.Request) {
	snapshots, err := getStandingsSnapshots(50)
	if err != nil {
		log.Printf("warning: failed to query standings snapshots: %v", err)
		http.Error(w, "failed to load standings history", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snapshots)
}
