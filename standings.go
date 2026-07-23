package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

func standingsURL() string {
	url := fmt.Sprintf("https://api.football-data.org/v4/competitions/%s/standings", getStandingsCompetition())
	if season := getStandingsSeason(); season != "" {
		url += "?season=" + season
	}
	return url
}

type fdStandingsResponse struct {
	Standings []struct {
		Type  string `json:"type"`
		Table []struct {
			Position int `json:"position"`
			Team     struct {
				Name string `json:"name"`
			} `json:"team"`
			PlayedGames    int `json:"playedGames"`
			Won            int `json:"won"`
			Draw           int `json:"draw"`
			Lost           int `json:"lost"`
			Points         int `json:"points"`
			GoalDifference int `json:"goalDifference"`
		} `json:"table"`
	} `json:"standings"`
}

var standings []StandingRow
var standingsMu sync.Mutex

func fetchStandings(token string) error {
	req, err := http.NewRequest("GET", standingsURL(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Auth-Token", token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("standings request returned status %d", resp.StatusCode)
		return nil
	}

	var fd fdStandingsResponse
	if err := json.NewDecoder(resp.Body).Decode(&fd); err != nil {
		return err
	}

	var fresh []StandingRow
	for _, s := range fd.Standings {
		if s.Type != "TOTAL" {
			continue
		}
		for _, row := range s.Table {
			fresh = append(fresh, StandingRow{
				Position: row.Position,
				Team:     row.Team.Name,
				Played:   row.PlayedGames,
				Won:      row.Won,
				Draw:     row.Draw,
				Lost:     row.Lost,
				Points:   row.Points,
				GoalDiff: row.GoalDifference,
			})
		}
	}

	standingsMu.Lock()
	standings = fresh
	standingsMu.Unlock()

	if err := recordStandingsSnapshot(getStandingsCompetition(), getStandingsSeason(), fresh); err != nil {
		log.Printf("warning: failed to record standings snapshot: %v", err)
	}

	log.Printf("fetched standings: %d team(s)", len(fresh))
	hub.broadcast(map[string]interface{}{"event": "standings_updated"})
	return nil
}

func startStandingsLoop(ctx context.Context, token string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Println("stopping standings fetch loop")
			return
		case <-ticker.C:
			if err := fetchStandings(token); err != nil {
				log.Printf("warning: standings fetch failed: %v", err)
			}
		}
	}
}
