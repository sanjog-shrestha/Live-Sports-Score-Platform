package main

type Goal struct {
	Minute int    `json:"minute"`
	Team   string `json:"team"`
	Scorer string `json:"scorer"`
}

type MatchDetail struct {
	Score
	Goals []Goal `json:"goals,omitempty"`
}

type Score struct {
	ID          string `json:"id"`
	Home        string `json:"home"`
	Away        string `json:"away"`
	HomeScore   int    `json:"homeScore"`
	AwayScore   int    `json:"awayScore"`
	Status      string `json:"status,omitempty"`
	Date        string `json:"date,omitempty"`
	Competition string `json:"competition,omitempty"`
}

type StandingRow struct {
	Position int    `json:"position"`
	Team     string `json:"team"`
	Played   int    `json:"played"`
	Won      int    `json:"won"`
	Draw     int    `json:"draw"`
	Lost     int    `json:"lost"`
	Points   int    `json:"points"`
	GoalDiff int    `json:"goalDiff"`
}
