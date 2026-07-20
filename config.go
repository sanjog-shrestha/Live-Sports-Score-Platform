package main

import "os"

func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return port
}

func getRedisAddr() string {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	return addr
}

func getFootballDataToken() string {
	return os.Getenv("FOOTBALL_DATA_TOKEN")
}

func getStandingsCompetition() string {
	code := os.Getenv("STANDINGS_COMPETITION")
	if code == "" {
		code = "PL"
	}
	return code
}

func getStandingsSeason() string {
	return os.Getenv("STANDINGS_SEASON")
}

func getBalldontlieToken() string {
	return os.Getenv("BALLDONTLIE_API_KEY")
}

func getApiFootballToken() string {
	return os.Getenv("API_FOOTBALL_KEY")
}

func getApiFootballLeagueID() string {
	id := os.Getenv("API_FOOTBALL_LEAGUE_ID")
	if id == "" {
		id = "39"
	}
	return id
}

func getApiFootballSeason() string {
	season := os.Getenv("API_FOOTBALL_SEASON")
	if season == "" {
		season = getStandingsSeason()
	}
	return season
}
