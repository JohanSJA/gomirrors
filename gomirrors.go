package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"time"
)

type URL struct {
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

type ByAge []URL

func (a ByAge) Len() int           { return len(a) }
func (a ByAge) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByAge) Less(i, j int) bool { return a[i].LastSync < a[j].LastSync }

type Status struct {
	Cutoff         int    `json:"cutoff"`
	CheckFrequency int    `json:"check_frequency"`
	NumChecks      int    `json:"num_checks"`
	LastCheck      string `json:"last_check"`
	Version        int    `json:"version"`
	URLs           []URL  `json:"urls"`
}

type MirrorRate struct {
	URL  URL
	Rate float64
}

type ByRate []MirrorRate

func (r ByRate) Len() int           { return len(r) }
func (r ByRate) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }
func (r ByRate) Less(i, j int) bool { return r[i].Rate < r[j].Rate }

func Mirrors() (Status, error) {
	var mirror Status

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

func Rate(mirror URL) MirrorRate {
	log.Printf("Rating %v", mirror.URL)

	mr := MirrorRate{
		URL: mirror,
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
	sizeInMb := float64(size) / 1024 / 1024

	rate := sizeInMb / float64(dur)

	return MirrorRate{
		URL:  mirror,
		Rate: rate,
	}
}

func Rates(mirrors []URL) []MirrorRate {
	mr := make(chan MirrorRate)
	pool := make(chan bool, 5)

	for _, m := range mirrors {
		go func(mu URL, mr chan MirrorRate, p chan bool) {
			p <- true
			mr <- Rate(mu)
			<-p
		}(m, mr, pool)
	}

	var rates []MirrorRate
	for range mirrors {
		r := <-mr
		rates = append(rates, r)
	}

	return rates
}

func FilterHTTP(mirrors []URL) []URL {
	var filter []URL

	for _, m := range mirrors {
		if m.Protocol == "http" {
			filter = append(filter, m)
		}
	}

	return filter
}

func main() {
	log.SetPrefix("# ")

	log.Print("gomirrors started")

	ms, err := Mirrors()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Println()
	log.Printf("Number of mirrors: %d", len(ms.URLs))

	mirrors := FilterHTTP(ms.URLs)
	log.Println()
	log.Printf("%d HTTP mirror:", len(mirrors))
	for i, m := range mirrors {
		log.Printf("%3d %s", i+1, m.URL)
	}

	sort.Sort(sort.Reverse(ByAge(mirrors)))
	mirrors = mirrors[:50]
	log.Println()
	log.Printf("%d Latest Sync HTTP Mirror:", len(mirrors))
	for i, m := range mirrors {
		log.Printf("%3d %-50s %s", i+1, m.URL, m.LastSync)
	}

	log.Println()
	log.Print("Rating Mirrors")
	rates := Rates(mirrors)

	sort.Sort(sort.Reverse(ByRate(rates)))
	log.Println()
	log.Printf("%d Latest Sync HTTP Mirror Sort By Rate:", len(rates))
	for i, r := range rates {
		log.Printf("%3d %-50s %.4f", i+1, r.URL.URL, r.Rate)
	}

	log.Println()
	log.Print("Writing mirrorlist")
	fmt.Println()
	for _, r := range rates {
		fmt.Printf("Server = %s$repo/os/$arch\n", r.URL.URL)
	}
}
