package main

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/ekzyis/sn-goapi"
)

var (
	TwitterUrlRegexp = regexp.MustCompile(`^(?:https?:\/\/)?((www\.)?(twitter|x)\.com)\/`)
	// references:
	// - https://github.com/zedeus/nitter/wiki/Instances
	// - https://status.d420.de/
	NitterClearnetUrls = []string{
		"nitter.privacydev.net",
	}
)

func WaitUntilNext(d time.Duration) {
	now := time.Now()
	dur := now.Truncate(d).Add(d).Sub(now)
	log.Println("sleeping for", dur.Round(time.Second))
	time.Sleep(dur)
}

func CheckNotifications() {
	var prevHasNewNotes bool
	for {
		log.Println("Checking notifications ...")
		hasNewNotes, err := sn.CheckNotifications()
		if err != nil {
			SendToNostr(fmt.Sprint(err))
		} else {
			if !prevHasNewNotes && hasNewNotes {
				// only send on "rising edge"
				SendToNostr("new notifications")
				log.Println("Forwarded notifications to monitoring")
			} else if hasNewNotes {
				log.Println("Notifications already forwarded")
			}
		}
		prevHasNewNotes = hasNewNotes
		WaitUntilNext(time.Hour)
	}
}

func SessionKeepAlive() {
	for {
		log.Println("Refresh session using GET /api/auth/session ...")
		sn.RefreshSession()
		WaitUntilNext(time.Hour)
	}
}

func main() {
	go CheckNotifications()
	go SessionKeepAlive()
	for {
		log.Println("fetching items ...")
		r, err := sn.Items(&sn.ItemsQuery{Sort: "recent", Limit: 21})
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
				cId, err := sn.CreateComment(item.Id, comment)
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
		}

		WaitUntilNext(time.Minute)
	}
}
