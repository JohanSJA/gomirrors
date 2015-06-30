package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type MirrorURL struct {
	Protocol       string  `json:"protocol"`
	URL            string  `json:"url"`
	Country        string  `json:"country"`
	LastSync       string  `json:"last_sync"`
	Delay          int     `json:"delay"`
	Score          float64 `json:"score"`
	CompletionPct  float64 `json:"completion_pct"`
	CountryCode    string  `json:"country_code"`
	DurationStdDev float64 `json:"duration_stddev"`
	DurationAvg    float64 `json:"duration_avg"`
}

type MirrorStatus struct {
	Cutoff         int         `json:"cutoff"`
	CheckFrequency int         `json:"check_frequency"`
	NumChecks      int         `json:"num_checks"`
	LastCheck      string      `json:"last_check"`
	Version        int         `json:"version"`
	URLs           []MirrorURL `json:"urls"`
}

func Mirrors() (MirrorStatus, error) {
	var mirror MirrorStatus

	resp, err := http.Get("https://www.archlinux.org/mirrors/status/json/")
	if err != nil {
		return mirror, err
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&mirror); err != nil {
		return mirror, err
	}

	return mirror, nil
}

func main() {
	log.Print("gomirrors started")

	ms, err := Mirrors()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("Mirror Status: %+v", ms)
}
