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

type PlayerStat struct {
	Rank        int    `json:"rank"`
	Name        string `json:"name"`
	Team        string `json:"team"`
	Appearances int    `json:"appearances"`
	Goals       int    `json:"goals"`
	Assists     int    `json:"assists"`
}

type afTopScorersResponse struct {
	Errors   interface{}     `json:"errors"`
	Results  int             `json:"results"`
	Response []afPlayerEntry `json:"response"`
}

type afPlayerEntry struct {
	Player struct {
		Name string `json:"name"`
	} `json:"player"`
	Statistics []struct {
		Team struct {
			Name string `json:"name"`
		} `json:"team"`
		Games struct {
			Appearences int `json:"appearences"`
		} `json:"games"`
		Goals struct {
			Total   *int `json:"total"`
			Assists *int `json:"assists"`
		} `json:"goals"`
	} `json:"statistics"`
}

var topScorers = []PlayerStat{}
var topScorersMu sync.Mutex

const topScorersCacheTTL = 2 * time.Hour

func topScorersCacheKey() string {
	return getApiFootballLeagueID() + ":" + getApiFootballSeason()
}

func loadTopScorersOnStartup(token string) error {
	if cached, found := getCachedTopScorers(topScorersCacheKey(), topScorersCacheTTL); found {
		log.Printf("loaded %d top scorer(s) from database cache -- skipping API call", len(cached))
		topScorersMu.Lock()
		topScorers = cached
		topScorersMu.Unlock()
		return nil
	}
	return fetchTopScorers(token)
}

func fetchTopScorers(token string) error {
	req, err := http.NewRequest("GET", apiFootballBase+"/players/topscorers", nil)
	if err != nil {
		return err
	}
	req.Header.Set("x-apisports-key", token)

	q := req.URL.Query()
	q.Set("league", getApiFootballLeagueID())
	q.Set("season", getApiFootballSeason())
	req.URL.RawQuery = q.Encode()

	log.Printf("api-football request: %s", req.URL.String())

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("api-football returned status %d", resp.StatusCode)
	}

	var out afTopScorersResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}

	log.Printf("api-football response: results=%d errors=%v", out.Results, out.Errors)

	if errMap, ok := out.Errors.(map[string]interface{}); ok && len(errMap) > 0 {
		return fmt.Errorf("api-football error: %v", errMap)
	}

	fresh := make([]PlayerStat, 0, len(out.Response))
	for i, entry := range out.Response {
		if len(entry.Statistics) == 0 {
			continue
		}
		stat := entry.Statistics[0]
		fresh = append(fresh, PlayerStat{
			Rank:        i + 1,
			Name:        entry.Player.Name,
			Team:        stat.Team.Name,
			Appearances: stat.Games.Appearences,
			Goals:       intOrZero(stat.Goals.Total),
			Assists:     intOrZero(stat.Goals.Assists),
		})
	}

	topScorersMu.Lock()
	topScorers = fresh
	topScorersMu.Unlock()

	if err := setCachedTopScorers(topScorersCacheKey(), fresh); err != nil {
		log.Printf("warning: failed to persist top scorers cache: %v", err)
	}

	log.Printf("fetched top scorers: %d player(s)", len(fresh))
	hub.broadcast(map[string]interface{}{"event": "top_scorers_updated"})
	return nil
}

func startTopScorersLoop(ctx context.Context, token string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Println("stopping top scorers fetch loop")
			return
		case <-ticker.C:
			if err := fetchTopScorers(token); err != nil {
				log.Printf("warning: top scorers fetch failed: %v", err)
			}
		}
	}
}
