package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	sn "github.com/ekzyis/snappy"
)

var (
	TwitterUrlRegexp = regexp.MustCompile(`(?:https?:\/\/)?(?:www\.)?((?:twitter|x)\.com)\/\w+\/status(?:es)?\/\d+`)
	NitterDomain     = "twiiit.com"
)

func WaitUntilNext(d time.Duration) {
	now := time.Now()
	dur := now.Truncate(d).Add(d).Sub(now)
	log.Println("sleeping for", dur.Round(time.Second))
	time.Sleep(dur)
}

func main() {
	loadEnv()

	c := sn.NewClient(
		sn.WithBaseUrl(os.Getenv("SN_BASE_URL")),
		sn.WithApiKey(os.Getenv("SN_API_KEY")),
	)

	for {
		log.Println("fetching items ...")
		r, err := c.Items(&sn.ItemsQuery{Sort: "new", Type: "all", Limit: 100})
		if err != nil {
			log.Println(err)
			WaitUntilNext(time.Minute)
			continue
		}

		for _, item := range r.Items {
			var m []string
			var comment string

			if m = TwitterUrlRegexp.FindStringSubmatch(item.Url); m != nil {
				comment = strings.Replace(m[0], m[1], NitterDomain, 1)
			} else if m = TwitterUrlRegexp.FindStringSubmatch(item.Text); m != nil {
				comment = strings.Replace(m[0], m[1], NitterDomain, 1)
			}

			if paywallLink := FindPaywalledUrl(item.Url, item.Text); paywallLink != "" {
				if snap, err := ArchiveLink(paywallLink); err == nil {
					comment = strings.Join(
						[]string{
							comment,
						  fmt.Sprintf("Paywall link detected. Archived version: [%s](%s)", snap.Label, snap.Url),
						}, "\n\n",
					)
				} else {
					log.Println("archive lookup failed:", err)
				}
			}

			if comment != "" {
				log.Printf("item %d is twitter link\n", item.Id)

				if ItemHasComment(item.Id) {
					log.Printf("item %d already has nitter links comment\n", item.Id)
					continue
				}

				cId, err := c.CreateComment(item.Id, comment)
				if err != nil {
					log.Println("create comment failed:", err)
					continue
				}

				log.Printf("created comment %d\n", cId)
				SaveComment(&sn.Comment{Id: cId, Text: comment, ParentId: item.Id})
			}
		}

		WaitUntilNext(time.Minute)
	}
}

func loadEnv() {
	var (
		f   *os.File
		s   *bufio.Scanner
		err error
	)

	if f, err = os.Open(".env"); err != nil {
		log.Fatalf("error opening .env: %v", err)
	}
	defer f.Close()

	s = bufio.NewScanner(f)
	s.Split(bufio.ScanLines)
	for s.Scan() {
		line := s.Text()
		parts := strings.SplitN(line, "=", 2)

		// Check if we have exactly 2 parts (key and value)
		if len(parts) == 2 {
			os.Setenv(parts[0], parts[1])
		} else {
			log.Fatalf(".env: invalid line: %s\n", line)
		}
	}

	// Check for errors during scanning
	if err = s.Err(); err != nil {
		fmt.Println("error scanning .env:", err)
	}

}
