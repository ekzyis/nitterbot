package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const userAgent = "Mozilla/5.0 (X11; Linux x86_64) Chrome/120.0"

// WaybackApiUrl is the Internet Archive Wayback Machine availability API, which
// reports the closest snapshot for a URL as JSON.
var WaybackApiUrl = "https://archive.org/wayback/available"

var archiveClient = &http.Client{Timeout: 30 * time.Second}

// PaywalledDomains lists host suffixes whose articles are typically paywalled.
// A host matches if it equals an entry or is a subdomain of one.
var PaywalledDomains = []string{
	"nytimes.com",
	"wsj.com",
	"ft.com",
	"bloomberg.com",
	"economist.com",
	"washingtonpost.com",
	"newyorker.com",
	"wired.com",
	"theatlantic.com",
	"businessinsider.com",
	"forbes.com",
	"barrons.com",
	"hbr.org",
	"technologyreview.com",
	"thetimes.co.uk",
	"telegraph.co.uk",
	"foreignpolicy.com",
	"foreignaffairs.com",
	"nature.com",
	"science.org",
	"theinformation.com",
	"latimes.com",
	"bostonglobe.com",
	"chicagotribune.com",
	"seekingalpha.com",
	"marketwatch.com",
	"fortune.com",
	"nikkei.com",
	"scmp.com",
	"theglobeandmail.com",
	"nationalpost.com",
	"medium.com",
	"nymag.com",
	"vanityfair.com",
	"harpers.org",
	"nybooks.com",
	"spectator.co.uk",
	"newstatesman.com",
	"thenation.com",
}

var urlRegexp = regexp.MustCompile(`https?://[^\s<>"'` + "`" + `)\]]+`)

func normalizeHost(host string) string {
	if i := strings.IndexByte(host, ':'); i != -1 {
		host = host[:i]
	}
	return strings.TrimPrefix(strings.ToLower(host), "www.")
}

// IsPaywalled reports whether rawUrl points at a known paywalled domain.
func IsPaywalled(rawUrl string) bool {
	u, err := url.Parse(rawUrl)
	if err != nil || u.Host == "" {
		return false
	}
	host := normalizeHost(u.Host)
	for _, d := range PaywalledDomains {
		if host == d || strings.HasSuffix(host, "."+d) {
			return true
		}
	}
	return false
}

// FindPaywalledUrl returns the first paywalled URL found in an item's link or
// text body, or "" if there is none.
func FindPaywalledUrl(linkUrl, text string) string {
	if IsPaywalled(linkUrl) {
		return linkUrl
	}
	for _, m := range urlRegexp.FindAllString(text, -1) {
		m = strings.TrimRight(m, ".,;:!?)]}\"'")
		if IsPaywalled(m) {
			return m
		}
	}
	return ""
}

// Snapshot is an archived capture of a URL.
type Snapshot struct {
	Url   string
	Label string // human-readable capture time, e.g. "Sep 10, 2026 10:19 UTC"
}

// ArchiveLink returns the Wayback Machine snapshot closest to now for rawUrl, or
// nil if no snapshot exists.
func ArchiveLink(rawUrl string) (*Snapshot, error) {
	endpoint := fmt.Sprintf("%s?url=%s", WaybackApiUrl, url.QueryEscape(rawUrl))

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := archiveClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("archive lookup for %s: unexpected status %d", rawUrl, resp.StatusCode)
	}

	var body struct {
		ArchivedSnapshots struct {
			Closest struct {
				Available bool   `json:"available"`
				Url       string `json:"url"`
				Timestamp string `json:"timestamp"`
			} `json:"closest"`
		} `json:"archived_snapshots"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decoding archive response for %s: %w", rawUrl, err)
	}

	closest := body.ArchivedSnapshots.Closest
	if !closest.Available || closest.Url == "" {
		return nil, nil
	}

	label := closest.Timestamp
	if t, err := time.Parse("20060102150405", closest.Timestamp); err == nil {
		label = t.Format("Jan 2, 2006 15:04 MST")
	}

	return &Snapshot{
		// the API returns http URLs; prefer https
		Url:   strings.Replace(closest.Url, "http://web.archive.org", "https://web.archive.org", 1),
		Label: label,
	}, nil
}
