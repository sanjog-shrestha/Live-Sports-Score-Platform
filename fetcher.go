package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

const footballDataBaseURL = "https://api.football-data.org/v4/matches"

func buildMatchesURL() string {
	to := time.Now()
	from := to.AddDate(0, 0, -7)
	return fmt.Sprintf("%s?dateFrom=%s&dateTo=%s",
		footballDataBaseURL,
		from.Format("2006-01-02"),
		to.Format("2006-01-02"),
	)
}

type fdResponse struct {
	Matches []fdMatch `json:"matches"`
}

type fdMatch struct {
	ID          int    `json:"id"`
	Status      string `json:"status"`
	UTCDate     string `json:"utcDate"`
	Competition struct {
		Name string `json:"name"`
	} `json:"competition"`
	HomeTeam struct {
		Name string `json:"name"`
	} `json:"homeTeam"`
	AwayTeam struct {
		Name string `json:"name"`
	} `json:"awayTeam"`
	Score struct {
		FullTime struct {
			Home *int `json:"home"`
			Away *int `json:"away"`
		} `json:"fullTime"`
	} `json:"score"`
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

func formatMatchDate(utcDate string) string {
	t, err := time.Parse(time.RFC3339, utcDate)
	if err != nil {
		return utcDate
	}
	return t.Format("Jan 2, 2006 15:04") + " UTC"
}

func fetchScores(token string) error {
	req, err := http.NewRequest("GET", buildMatchesURL(), nil)
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
		log.Printf("football-data.org returned status %d", resp.StatusCode)
		return nil
	}

	var fd fdResponse
	if err := json.NewDecoder(resp.Body).Decode(&fd); err != nil {
		return err
	}

	fresh := make([]Score, 0, len(fd.Matches))
	for _, m := range fd.Matches {
		fresh = append(fresh, Score{
			ID:          strconv.Itoa(m.ID),
			Home:        m.HomeTeam.Name,
			Away:        m.AwayTeam.Name,
			HomeScore:   intOrZero(m.Score.FullTime.Home),
			AwayScore:   intOrZero(m.Score.FullTime.Away),
			Status:      m.Status,
			Date:        formatMatchDate(m.UTCDate),
			Competition: m.Competition.Name,
		})
	}

	log.Printf("fetched %d match(es) from football-data.org", len(fresh))

	footballScores = fresh
	mergeScores()
	return nil
}

func intOrZero(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func startFetchLoop(ctx context.Context, token string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Println("stopping football-data.org fetch loop")
			return
		case <-ticker.C:
			if err := fetchScores(token); err != nil {
				log.Printf("warning: fetch failed: %v", err)
			}
		}
	}
}
