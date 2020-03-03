package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"unicode/utf8"

	"github.com/SlyMarbo/rss"
	"github.com/gdamore/tcell"
	"github.com/mattn/go-runewidth"
	bolt "go.etcd.io/bbolt"
)

//Item is an rss feed item
type Item struct {
	Read int       `json:"read"`
	I    *rss.Item `json:"item"`
}

func main() {
	db, err := bolt.Open("lel.db", 0666, nil)
	fatal(err)
	defer db.Close()
	var url string
	if len(os.Args) == 1 {
		url = "http://carlgene.com/blog/feed/atom/"
	} else {
		url = os.Args[1]
	}
	feed, err := rss.Fetch(url)
	fatal(err)
	populateDB(db, feed)
	s, err := tcell.NewScreen()
	fatal(err)
	err = s.Init()
	fatal(err)
	defer s.Fini()
	currentItem := 0
mainloop:
	for {
		ev := s.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyEscape:
				break mainloop
			case tcell.KeyDown:
				if currentItem < len(feed.Items)-1 {
					currentItem++
				}
			case tcell.KeyUp:
				if currentItem > 0 {
					currentItem--
				}
			case tcell.KeyRune:
				switch ev.Rune() {
				case 'o':
					openURL(db, feed.Items[currentItem].Link)
				case 'q':
					break mainloop
				case 'j':
					if currentItem < len(feed.Items)-1 {
						currentItem++
					}
				case 'k':
					if currentItem > 0 {
						currentItem--
					}
				}
			}
		}
		scroll(db, s, currentItem, feed)
		s.Sync()
	}
}

func markRead(db *bolt.DB, url string) {
	err := db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("rss"))
		buf := bucket.Get([]byte(url))
		var i Item
		err := json.Unmarshal(buf, &i)
		if err != nil {
			return err
		}
		i.Read = 1
		buf, err = json.Marshal(i)
		if err != nil {
			return err
		}
		bucket.Put([]byte(url), buf)
		return nil
	})
	fatal(err)
}

func populateDB(db *bolt.DB, feed *rss.Feed) {
	err := db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("rss"))
		if err != nil {
			return err
		}
		for _, f := range feed.Items {
			key := bucket.Get([]byte(f.Link))
			if key == nil {
				i := Item{I: f, Read: 0}
				buf, err := json.Marshal(i)
				if err != nil {
					return err
				}
				bucket.Put([]byte(f.Link), buf)
			}
		}
		return nil
	})
	fatal(err)
}

func openURL(db *bolt.DB, url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default:
		cmd = "xdg-open"
	}
	args = append(args, url)
	err := exec.Command(cmd, args...).Start()
	fatal(err)
	markRead(db, url)
}

func scroll(db *bolt.DB, s tcell.Screen, item int, feed *rss.Feed) {
	w, _ := s.Size()
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("rss"))
		for y, f := range feed.Items {
			read := bucket.Get([]byte(f.Link))[8]
			var result string
			var style tcell.Style
			if read == 48 {
				result = fmt.Sprintf("%4d%2s %s", y, "N", f.Title)
				style = tcell.StyleDefault.Bold(true)
			} else {
				result = fmt.Sprintf("%4d%2s %s", y, " ", f.Title)
				style = tcell.StyleDefault
			}
			for utf8.RuneCountInString(result) < w {
				result += " "
			}
			if y == item {
				print(s, 0, y, style.Reverse(true), result)
			} else {
				print(s, 0, y, style, result)
			}
			i := Item{I: f, Read: 0}
			buf, err := json.Marshal(i)
			if err != nil {
				return err
			}
			bucket.Put([]byte(f.Link), buf)
		}
		return nil
	})
	fatal(err)
}

func print(s tcell.Screen, x, y int, style tcell.Style, text string) {
	for _, c := range text {
		s.SetCell(x, y, style, c)
		x += runewidth.RuneWidth(c)
	}
}

func fatal(e error) {
	if e != nil {
		log.Fatal(e)
	}
}
