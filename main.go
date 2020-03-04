package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"time"
	"unicode/utf8"

	"github.com/SlyMarbo/rss"
	"github.com/gdamore/tcell"
	"github.com/mattn/go-runewidth"
	bolt "go.etcd.io/bbolt"
)

const (
	READ_OFFSET = 8  // which byte is responsible for read status flag in a bolt key
	ZERO        = 48 // 0 in ascii. means false, not read
	TIMER       = 30 // reload feeds every this many minutes
)

//Item is an rss feed item
type Item struct {
	Read  int       `json:"read"`
	Title string    `json:"title"`
	I     *rss.Item `json:"item"`
}

func main() {
	currentItem := 0
	coldStart := true
	maxTitle := 0
	dir, err := os.UserConfigDir()
	fatal(err)
	db, err := bolt.Open(filepath.FromSlash(dir+filepath.FromSlash("/lydia/db")), 0600, nil)
	fatal(err)
	defer db.Close()
	items := make([]Item, 0, 256)
	s, err := tcell.NewScreen()
	fatal(err)
	go func() {
		for {
			items = populateDB(db, &maxTitle)
			scroll(db, s, &currentItem, items, &coldStart, maxTitle)
			s.Sync()
			time.Sleep(TIMER * time.Minute)
		}
	}()
	err = s.Init()
	fatal(err)
	defer s.Fini()
mainloop:
	for {
		ev := s.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyEscape:
				break mainloop
			case tcell.KeyDown:
				_, h := s.Size()
				if currentItem < h-1 {
					currentItem++
				}
			case tcell.KeyUp:
				if currentItem > 0 {
					currentItem--
				}
			case tcell.KeyRune:
				switch ev.Rune() {
				case 'o':
					openURL(db, items[currentItem].I.Link)
					currentItem++
					scroll(db, s, &currentItem, items, &coldStart, maxTitle)
				case 'q':
					break mainloop
				case 'j':
					_, h := s.Size()
					if currentItem < h-1 {
						currentItem++
					}
				case 'k':
					if currentItem > 0 {
						currentItem--
					}
				}
				coldStart = false
			}
		}
		scroll(db, s, &currentItem, items, &coldStart, maxTitle)
		s.Sync()
	}
}

func date(d time.Time) string {
	if time.Now().Unix()-d.Unix() > 24*60*60 {
		return fmt.Sprintf("%s %02d", d.Month().String()[:3], d.Day())
	}
	return fmt.Sprintf("%02d:%02d", d.Hour(), d.Minute())

}

func leng(s string) int {
	l := 0
	for _, r := range s {
		l += runewidth.RuneWidth(r)
	}
	if l > 20 {
		l = 20
	}
	return l
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

func populateDB(db *bolt.DB, maxTitle *int) []Item {
	dir, err := os.UserConfigDir()
	fatal(err)
	file, err := os.Open(filepath.FromSlash(dir + filepath.FromSlash("/lydia/urls")))
	fatal(err)
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var items []Item = make([]Item, 0, 256)
	for scanner.Scan() {
		text := scanner.Text()
		if text[0] == '#' {
			continue
		}
		f, err := rss.Fetch(scanner.Text())
		fatal(err)
		if leng(f.Title) > *maxTitle {
			*maxTitle = leng(f.Title)
		}
		for _, i := range f.Items {
			var item Item
			item = Item{
				Read:  0,
				Title: f.Title,
				I:     i,
			}
			items = append(items, item)
		}
	}
	fatal(scanner.Err())
	sort.Slice(items, func(i, j int) bool {
		return items[i].I.Date.Unix() > items[j].I.Date.Unix()
	})
	err = db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("rss"))
		if err != nil {
			return err
		}
		for _, i := range items {
			key := bucket.Get([]byte(i.I.Link))
			if key == nil {
				buf, err := json.Marshal(i)
				if err != nil {
					return err
				}
				bucket.Put([]byte(i.I.Link), buf)
			}
		}
		return nil
	})
	fatal(err)
	return items
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

func scroll(db *bolt.DB, s tcell.Screen, item *int, items []Item, coldStart *bool, maxTitle int) {
	w, _ := s.Size()
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("rss"))
		for y, i := range items {
			read := bucket.Get([]byte(i.I.Link))[READ_OFFSET]
			if *coldStart {
				if read != ZERO {
					*item++
				} else {
					*coldStart = false
				}
			}
			var result string
			var style tcell.Style
			if read == ZERO {
				result = fmt.Sprintf("%6s│%[2]*s│%s", date(i.I.Date), maxTitle, i.Title, i.I.Title)
				style = tcell.StyleDefault.Bold(true)
			} else {
				result = fmt.Sprintf("%6s│%[2]*s│%s", date(i.I.Date), maxTitle, i.Title, i.I.Title)
				style = tcell.StyleDefault
			}
			for utf8.RuneCountInString(result) < w {
				result += " "
			}
			if y == *item {
				print(s, 0, y, style.Reverse(true), result)
			} else {
				print(s, 0, y, style, result)
			}
			buf, err := json.Marshal(i)
			if err != nil {
				return err
			}
			bucket.Put([]byte(i.I.Link), buf)
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
