package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"
)

const balldontlieGamesURL = "https://api.balldontlie.io/nba/v1/games"

type bdlResponse struct {
	Data []bdlGame `json:"data"`
}

type bdlGame struct {
	ID               int    `json:"id"`
	Status           string `json:"status"`
	Datetime         string `json:"datetime"`
	HomeTeamScore    int    `json:"home_team_score"`
	VisitorTeamScore int    `json:"visitor_team_score"`
	HomeTeam         struct {
		FullName string `json:"full_name"`
	} `json:"home_team"`
	VisitorTeam struct {
		FullName string `json:"full_name"`
	} `json:"visitor_team"`
}

func fetchNBAGames(token string) error {
	req, err := http.NewRequest("GET", balldontlieGamesURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", token)

	q := req.URL.Query()
	q.Set("dates[]", time.Now().Format("2006-01-02"))
	req.URL.RawQuery = q.Encode()

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("ballsdontlie.io returned status %d", resp.StatusCode)
		return nil
	}

	var bdl bdlResponse
	if err := json.NewDecoder(resp.Body).Decode(&bdl); err != nil {
		return err
	}

	fresh := make([]Score, 0, len(bdl.Data))
	for _, g := range bdl.Data {
		fresh = append(fresh, Score{
			ID:          "nba-" + strconv.Itoa(g.ID),
			Home:        g.HomeTeam.FullName,
			Away:        g.VisitorTeam.FullName,
			HomeScore:   g.HomeTeamScore,
			AwayScore:   g.VisitorTeamScore,
			Status:      g.Status,
			Date:        formatMatchDate(g.Datetime),
			Competition: "NBA",
		})
	}

	log.Printf("fetched %d NBA game(s) from ballsdontlie.io", len(fresh))

	nbaScores = fresh
	mergeScores()
	return nil
}

func startNBALoop(token string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		if err := fetchNBAGames(token); err != nil {
			log.Printf("warning: NBA fetch failed: %v", err)
		}
	}
}
