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
		"nitter.net",
		"nitter.it",
		"nitter.cz",
		"nitter.at",
		"nitter.unixfox.eu",
		"nitter.poast.org",
		"nitter.privacydev.net",
		"nitter.d420.de",
		"nitter.sethforprivacy.com",
		"nitter.nicfab.eu",
		"bird.habedieeh.re",
		"nitter.salastil.com",
		"nt.ggtyler.dev",
	}
	NitterOnionUrls = []string{
		"nitter7bryz3jv7e3uekphigvmoyoem4al3fynerxkj22dmoxoq553qd.onion",
		"26oq3gioiwcmfojub37nz5gzbkdiqp7fue5kvye7d4txv4ny6fb4wwid.onion",
		"vfaomgh4jxphpbdfizkm5gbtjahmei234giqj4facbwhrfjtcldauqad.onion",
		"nitraeju2mipeziu2wtcrqsxg7h62v5y4eqgwi75uprynkj74gevvuqd.onion",
		"codeine3hsqnnkb3dsu6ft4tunlomr3lmuml5hcoqmfkgiqfv2brdqqd.onion",
	}
	NitterI2PUrls = []string{
		"axd6uavsstsrvstva4mzlzh4ct76rc6zdug3nxdgeitrzczhzf4q.b32.i2p",
		"u6ikd6zndl3c4dsdq4mmujpntgeevdk5qzkfb57r4tnfeccrn2qa.b32.i2p",
		"gseczlzmiv23p5vhsktyd7whquq2uy3c5fgkmdohh453qp3daoua.b32.i2p",
		"tm4rwkeysv3zz3q5yacyr4rlmca2c4etkdobfvuqzt6vsfsu4weq.b32.i2p",
		"vernzdedoxuflrrxc4vbatbkpjh4k22ecgiqgimdiif62onhagva.b32.i2p",
	}
	NitterLokinetUrls = []string{
		"nitter.priv.loki/",
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
				comment := "**Twitter2Nitter**\n\nClearnet:\n\n"
				for _, nUrl := range NitterClearnetUrls {
					nitterLink := strings.Replace(item.Url, m[1], nUrl, 1)
					comment += fmt.Sprintf("[%s](%s) | ", nUrl, nitterLink)
				}
				comment = strings.TrimRight(comment, "| ")
				comment += "\n\nTor:\n\n"
				for _, nUrl := range NitterOnionUrls {
					nitterLink := strings.Replace(item.Url, m[1], nUrl, 1)
					nitterLink = strings.Replace(nitterLink, "https://", "http://", 1)
					comment += fmt.Sprintf("[%s..%s](%s) | ", nUrl[:12], nUrl[len(nUrl)-12:], nitterLink)
				}
				comment = strings.TrimRight(comment, "| ")
				comment += "\n\nI2P:\n\n"
				for _, nUrl := range NitterI2PUrls {
					nitterLink := strings.Replace(item.Url, m[1], nUrl, 1)
					nitterLink = strings.Replace(nitterLink, "https://", "http://", 1)
					comment += fmt.Sprintf("[%s..%s](%s) | ", nUrl[:12], nUrl[len(nUrl)-12:], nitterLink)
				}
				comment = strings.TrimRight(comment, "| ")
				comment += "\n\nLokinet:\n\n"
				for _, nUrl := range NitterLokinetUrls {
					nitterLink := strings.Replace(item.Url, m[1], nUrl, 1)
					nitterLink = strings.Replace(nitterLink, "https://", "http://", 1)
					comment += fmt.Sprintf("[%s](%s)\n", nUrl, nitterLink)
				}
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
