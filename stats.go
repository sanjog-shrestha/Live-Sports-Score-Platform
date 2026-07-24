package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
)

type SeasonStats struct {
	TotalMatches         int         `json:"totalMatches"`
	TotalGoals           int         `json:"totalGoals"`
	AverageGoalsPerMatch float64     `json:"averageGoalsPerMatch"`
	HomeWins             int         `json:"homeWins"`
	AwayWins             int         `json:"awayWins"`
	Draws                int         `json:"draws"`
	BiggestWin           *BiggestWin `json:"biggestWin,omitempty"`
	TopScoringTeam       *TeamGoals  `json:"topScoringTeam,omitempty"`
	MostCleanSheets      *TeamCount  `json:"mostCleanSheets,omitempty"`
}

type BiggestWin struct {
	Home      string `json:"home"`
	Away      string `json:"away"`
	HomeScore int    `json:"homeScore"`
	AwayScore int    `json:"awayScore"`
	Diff      int    `json:"diff"`
}

type TeamGoals struct {
	Team  string `json:"team"`
	Goals int    `json:"goals"`
}

type TeamCount struct {
	Team  string `json:"team"`
	Count int    `json:"count"`
}

func getSeasonStats() (SeasonStats, error) {
	var stats SeasonStats

	row := db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(home_score + away_score), 0)
		FROM match_history WHERE status = 'FINISHED'	
	`)
	if err := row.Scan(&stats.TotalMatches, &stats.TotalGoals); err != nil {
		return stats, err
	}
	if stats.TotalMatches > 0 {
		stats.AverageGoalsPerMatch = float64(stats.TotalGoals) / float64(stats.TotalMatches)
	}

	row = db.QueryRow(`
		SELECT
			SUM(CASE WHEN home_score > away_score THEN 1 ELSE 0 END),
			SUM(CASE WHEN away_score > home_score THEN 1 ELSE 0 END),
			SUM(CASE WHEN home_score = away_score THEN 1 ELSE 0 END)
		FROM match_history WHERE status = 'FINISHED'
	`)
	if err := row.Scan(&stats.HomeWins, &stats.AwayWins, &stats.Draws); err != nil {
		return stats, err
	}

	var bw BiggestWin
	row = db.QueryRow(`
		SELECT home, away, home_score, away_score, ABS(home_score - away_score) AS diff 
		FROM match_history
		WHERE status = 'FINISHED'
		ORDER BY diff DESC
		LIMIT 1 
	`)
	switch err := row.Scan(&bw.Home, &bw.Away, &bw.HomeScore, &bw.AwayScore, &bw.Diff); err {
	case nil:
		stats.BiggestWin = &bw
	case sql.ErrNoRows:
	default:
		return stats, err
	}

	var ts TeamGoals
	row = db.QueryRow(`
		SELECT team, SUM(goals) AS total_goals FROM (
			SELECT home AS team, home_score AS goals FROM match_history WHERE status = 'FINISHED'
			UNION ALL
			SELECT away AS team, away_score AS goals FROM match_history WHERE status = 'FINISHED'
		)
		GROUP BY team
		ORDER BY total_goals DESC
		LIMIT 1
	`)
	switch err := row.Scan(&ts.Team, &ts.Goals); err {
	case nil:
		stats.TopScoringTeam = &ts
	case sql.ErrNoRows:
	default:
		return stats, err
	}

	var cs TeamCount
	row = db.QueryRow(`
		SELECT team, COUNT(*) AS clean_sheets FROM (
			SELECT home AS team FROM match_history WHERE status = 'FINISHED' AND away_score = 0
			UNION ALL
			SELECT away AS team FROM match_history WHERE status = 'FINISHED' AND home_score = 0
		)
		GROUP BY team
		ORDER BY clean_sheets DESC
		LIMIT 1
	`)
	switch err := row.Scan(&cs.Team, &cs.Count); err {
	case nil:
		stats.MostCleanSheets = &cs
	case sql.ErrNoRows:
	default:
		return stats, err
	}

	return stats, nil
}

func seasonStatsHandler(w http.ResponseWriter, r *http.Request) {
	stats, err := getSeasonStats()
	if err != nil {
		log.Printf("warning: failed to compute season stats: %v", err)
		http.Error(w, "failed to compute season stats", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
