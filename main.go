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

type NostrClient struct {
	Url  string
	Name string
}

var (
	TwitterUrlRegexp = regexp.MustCompile(`^(?:https?:\/\/)?((www\.)?(twitter|x)\.com)\/`)
	// references:
	// - https://github.com/zedeus/nitter/wiki/Instances
	// - https://status.d420.de/
	NitterClearnetUrls = []string{
		"xcancel.com",
	}

	// since v0.4.0, bot also replaces nostr links with nostr.com so users can pick their client
	NostrUrlRegexp = regexp.MustCompile(
		`^(?:https?:\/\/)?(?:www\.)?` +
			`(?:` +
			`primal\.net\/(?:e\/)?` +
			`|snort\.social\/(?:e\/)?` +
			`|iris\.to\/` +
			`|highlighter\.com\/(?:a\/)?` +
			`|nostter\.app\/` +
			`|coracle\.social\/(?:notes\/)?` +
			`|satellite\.earth\/` +
			`|nostrudel\.ninja\/(?:#\/n\/)?` +
			`)((note|nevent)[a-zA-Z0-9]+)$`)
	NostrClients = []NostrClient{
		// list from nostr.com
		NostrClient{"https://primal.net/e/", "primal.net"},
		NostrClient{"https://snort.social/e/", "snort.social"},
		NostrClient{"https://nostrudel.ninja/#/n/", "nostrudel.ninja"},
		NostrClient{"https://satellite.earth/thread/", "satellite.earth"},
		NostrClient{"https://coracle.social/", "coracle.social"},
		NostrClient{"https://nostter.app/", "nostter.app"},
		NostrClient{"https://highlighter.com/a/", "highlighter.com"},
		NostrClient{"https://iris.to/", "iris.to"},
	}
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
		r, err := c.Items(&sn.ItemsQuery{Sort: "recent", Limit: 100})
		if err != nil {
			log.Println(err)
			SendToNostr(fmt.Sprint(err))
			WaitUntilNext(time.Minute)
			continue
		}

		for _, item := range r.Items {
			if m := TwitterUrlRegexp.FindStringSubmatch(item.Url); m != nil {
				log.Printf("item %d is twitter link\n", item.Id)
				if ItemHasComment(item.Id) {
					log.Printf("item %d already has nitter links comment\n", item.Id)
					continue
				}
				comment := "**Twitter2Nitter**\n\nClearnet: "
				for _, nUrl := range NitterClearnetUrls {
					nitterLink := strings.Replace(item.Url, m[1], nUrl, 1)
					comment += fmt.Sprintf("[%s](%s) | ", nUrl, nitterLink)
				}
				comment = strings.TrimRight(comment, "| ")
				comment += "\n\n_Nitter is a free and open source alternative Twitter front-end focused on privacy and performance. "
				comment += "Click [here](https://github.com/zedeus/nitter) for more information._"
				cId, err := c.CreateComment(item.Id, comment)
				if err != nil {
					log.Println(err)
					SendToNostr(fmt.Sprint(err))
					continue
				}
				log.Printf("created comment %d\n", cId)
				SaveComment(&sn.Comment{Id: cId, Text: comment, ParentId: item.Id})
			} else {
				log.Printf("item %d is not twitter link\n", item.Id)
			}
			if m := NostrUrlRegexp.FindStringSubmatch(item.Url); m != nil {
				log.Printf("item %d is nostr link\n", item.Id)
				if ItemHasComment(item.Id) {
					log.Printf("item %d already has nostr links comment\n", item.Id)
					continue
				}
				noteId := m[1]
				comment := "**Nostr Client Picker**\n\n"
				for _, client := range NostrClients {
					comment += fmt.Sprintf("[%s](%s) | ", client.Name, client.Url+noteId)
				}
				comment = strings.TrimRight(comment, "| ")
				cId, err := c.CreateComment(item.Id, comment)
				if err != nil {
					log.Println(err)
					SendToNostr(fmt.Sprint(err))
					continue
				}
				log.Printf("created comment %d\n", cId)
				SaveComment(&sn.Comment{Id: cId, Text: comment, ParentId: item.Id})
			} else {
				log.Printf("item %d is not nostr link\n", item.Id)
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
