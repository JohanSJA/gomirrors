package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"sort"
	"time"
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

type ByAge []MirrorURL

func (a ByAge) Len() int           { return len(a) }
func (a ByAge) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByAge) Less(i, j int) bool { return a[i].LastSync < a[j].LastSync }

type MirrorStatus struct {
	Cutoff         int         `json:"cutoff"`
	CheckFrequency int         `json:"check_frequency"`
	NumChecks      int         `json:"num_checks"`
	LastCheck      string      `json:"last_check"`
	Version        int         `json:"version"`
	URLs           []MirrorURL `json:"urls"`
}

type MirrorRate struct {
	URL  MirrorURL
	Rate float64
}

type ByRate []MirrorRate

func (r ByRate) Len() int           { return len(r) }
func (r ByRate) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }
func (r ByRate) Less(i, j int) bool { return r[i].Rate < r[j].Rate }

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

func Rate(mirror MirrorURL) MirrorRate {
	mr := MirrorRate{
		URL:  mirror,
		Rate: math.MaxFloat64,
	}

	url := mirror.URL + "core/os/i686/core.db"
	start := time.Now()
	resp, err := http.Get(url)
	end := time.Now()
	if err != nil {
		return mr
	}
	defer resp.Body.Close()

	dur := end.Sub(start).Seconds()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return mr
	}
	size := len(body)

	rate := float64(size) / dur

	return MirrorRate{
		URL:  mirror,
		Rate: rate,
	}
}

func Rates(mirrors []MirrorURL) []MirrorRate {
	mr := make(chan MirrorRate)

	for _, m := range mirrors {
		go func(mu MirrorURL, mr chan MirrorRate) {
			mr <- Rate(mu)
		}(m, mr)
	}

	var rates []MirrorRate
	for range mirrors {
		r := <-mr
		rates = append(rates, r)
	}

	return rates
}

func main() {
	log.Print("gomirrors started")

	ms, err := Mirrors()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("Mirror Status: %+v", ms)

	sort.Sort(sort.Reverse(ByAge(ms.URLs)))
	mirrors := ms.URLs[:200]
	log.Println()
	log.Printf("200 Latest Sync Mirror: %+v", mirrors)

	rates := Rates(mirrors)
	sort.Sort(sort.Reverse(ByRate(rates)))
	log.Println()
	log.Printf("Mirror Rates: %+v", rates)
}
