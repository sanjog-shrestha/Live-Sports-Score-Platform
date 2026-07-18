package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const apiFootballBase = "https://v3.football.api-sports.io"

type afFixture struct {
	Fixture struct {
		ID int `json:"id"`
	} `json:"fixture"`
	Teams struct {
		Home struct {
			Name string `json:"name"`
		} `json:"home"`
		Away struct {
			Name string `json:"name"`
		} `json:"away"`
	} `json:"teams"`
	Events []struct {
		Time struct {
			Elapsed int `json:"elapsed"`
		} `json:"time"`
		Team struct {
			Name string `json:"name"`
		} `json:"team"`
		Player struct {
			Name string `json:"name"`
		} `json:"player"`
		Type string `json:"type"`
	} `json:"events"`
}

type afResponse struct {
	Errors   interface{} `json:"errors"`
	Results  int         `json:"results"`
	Response []afFixture `json:"response"`
}

func apiFootballRequest(token, path string, params map[string]string) (*afResponse, error) {
	req, err := http.NewRequest("GET", apiFootballBase+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-apisports-key", token)

	q := req.URL.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	req.URL.RawQuery = q.Encode()

	log.Printf("api-football request: %s", req.URL.String())

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api-football returned status %d", resp.StatusCode)
	}

	var out afResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}

	log.Printf("api-football response: results=%d errors=%v", out.Results, out.Errors)

	return &out, nil
}

var clubSuffixWords = map[string]bool{
	"fc": true, "sc": true, "afc": true, "cf": true, "ac": true,
	"ec": true, "af": true, "sk": true, "fk": true, "cd": true, "ca": true,
}

var accentReplacer = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a", "å", "a",
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i",
	"ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o", "ø", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u",
	"ñ", "n", "ç", "c", "ý", "y",
)

func teamTokens(name string) map[string]bool {
	name = accentReplacer.Replace(strings.ToLower(name))
	name = strings.ReplaceAll(name, "-", " ")
	tokens := map[string]bool{}
	for _, word := range strings.Fields(name) {
		if clubSuffixWords[word] || len(word) <= 2 {
			continue
		}
		tokens[word] = true
	}
	return tokens
}

func sameTeam(a, b string) bool {
	tokensA, tokensB := teamTokens(a), teamTokens(b)
	if len(tokensA) == 0 || len(tokensB) == 0 {
		return false
	}
	return isSubset(tokensA, tokensB) || isSubset(tokensB, tokensA)
}

func isSubset(small, big map[string]bool) bool {
	for word := range small {
		if !big[word] {
			return false
		}
	}
	return true
}

func findFixtureID(token, dateISO, home, away string) (int, error) {
	out, err := apiFootballRequest(token, "/fixtures", map[string]string{"date": dateISO})
	if err != nil {
		return 0, err
	}

	if errMap, ok := out.Errors.(map[string]interface{}); ok {
		if planMsg, ok := errMap["plan"]; ok {
			return 0, fmt.Errorf("date %s not available on free plan: %v", dateISO, planMsg)
		}
	}
	for _, f := range out.Response {
		if sameTeam(f.Teams.Home.Name, home) && sameTeam(f.Teams.Away.Name, away) {
			return f.Fixture.ID, nil
		}
	}
	return 0, fmt.Errorf("no matching fixture found for %s vs %s on %s", home, away, dateISO)
}

func isDateOnly(displayDate string) (string, error) {
	trimmed := strings.TrimSuffix(displayDate, " UTC")
	t, err := time.Parse("Jan 2, 2006 15:04", trimmed)
	if err != nil {
		return "", err
	}
	return t.Format("2006-01-02"), nil
}

func fetchMatchGoals(token, matchId string) ([]Goal, error) {
	if goals, found := getCachedGoals(matchId); found {
		return goals, nil
	}

	scoresMu.Lock()
	var match Score
	matchOK := false
	for _, s := range scores {
		if s.ID == matchId {
			match = s
			matchOK = true
			break
		}
	}
	scoresMu.Unlock()

	if !matchOK {
		return nil, nil
	}

	dateISO, err := isDateOnly(match.Date)
	if err != nil {
		log.Printf("warning: could not parse date %q for match %s: %v", match.Date, matchId, err)
		return nil, nil
	}

	fixtureID, err := findFixtureID(token, dateISO, match.Home, match.Away)
	if err != nil {
		log.Printf("could not locate API-Football fixture: %v", err)
		return nil, nil
	}

	out, err := apiFootballRequest(token, "/fixtures", map[string]string{"id": strconv.Itoa(fixtureID)})
	if err != nil {
		return nil, err
	}
	if len(out.Response) == 0 {
		return nil, nil
	}

	goals := make([]Goal, 0)
	for _, e := range out.Response[0].Events {
		if e.Type != "Goal" {
			continue
		}
		goals = append(goals, Goal{
			Minute: e.Time.Elapsed,
			Team:   e.Team.Name,
			Scorer: e.Player.Name,
		})
	}

	if match.Status == "FINISHED" {
		if err := setCachedGoals(matchId, goals); err != nil {
			log.Printf("warning: failed to persist goals for %s: %v", matchId, err)
		}
	}
	return goals, nil
}
