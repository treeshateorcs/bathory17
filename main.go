package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	dir, err := os.UserConfigDir()
	fatal(39, err)
	db, err := bolt.Open(filepath.FromSlash(dir+filepath.FromSlash("/lydia/db")), 0600, nil)
	fatal(41, err)
	defer db.Close()
	s, err := tcell.NewScreen()
	fatal(45, err)
	firstRun := false
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("unread"))
		if b == nil {
			firstRun = true
		}
		return err
	})
	err = s.Init()
	fatal(68, err)
	defer s.Fini()
	if firstRun {
		populateDB(s, db)
		scroll(db, s, &currentItem)
		s.Show()
	} else {
		go func() {
			for {
				populateDB(s, db)
				scroll(db, s, &currentItem)
				s.Show()
				time.Sleep(TIMER * time.Minute)
			}
		}()
	}
mainloop:
	for {
		_, h := s.Size()
		ev := s.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyEscape:
				break mainloop
			case tcell.KeyDown:
				if currentItem < h-1 {
					currentItem++
				}
			case tcell.KeyUp:
				if currentItem > 0 {
					currentItem--
				}
			case tcell.KeyRune:
				switch ev.Rune() {
				case 'A':
					markReadAll(db)
				case 'd':
					if currentItem < h {
						markRead(db, currentItem)
						if currentItem != h-1 {
							currentItem++
						}
					}
				case 'o':
					if currentItem < h {
						openURL(db, currentItem)
						if currentItem != h-1 {
							currentItem++
						}
					}
				case 'q':
					break mainloop
				case 'j':
					if currentItem < h-1 {
						currentItem++
					}
				case 'k':
					if currentItem > 0 {
						currentItem--
					}
				case 'r':
					go populateDB(s, db)
				}
			}
		}
		scroll(db, s, &currentItem)
		s.Show()
	}
}

func markReadAll(db *bolt.DB) {
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("unread"))
		if b == nil {
			fatal(128, errors.New("no bucket"))
		}
		c := b.Cursor()
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			var item Item
			err := json.Unmarshal(v, &item)
			if err != nil {
				return err
			}
			item.Read = 1
			buf, err := json.Marshal(item)
			if err != nil {
				return err
			}
			b.Put(k, buf)
		}
		return nil
	})
	fatal(153, err)
}

func markRead(db *bolt.DB, index int) {
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("unread"))
		if b == nil {
			fatal(154, errors.New("no bucket"))
		}
		c := b.Cursor()
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			index--
			if k == nil {
				fatal(230, errors.New("no key for current item"))
			}
			if index == -1 {
				var item Item
				err := json.Unmarshal(v, &item)
				if err != nil {
					return err
				}
				item.Read = 1
				buf, err := json.Marshal(item)
				if err != nil {
					return err
				}
				b.Put(k, buf)
				break
			}
		}
		return nil
	})
	fatal(179, err)
}

func leng(s string) int {
	l := 0
	for _, c := range s {
		l += runewidth.RuneWidth(c)
	}
	return l
}

func date(d time.Time) string {
	if time.Now().Unix()-d.Unix() > 24*60*60 {
		return fmt.Sprintf("%s %02d", d.Month().String()[:3], d.Day())
	}
	return fmt.Sprintf("%02d:%02d", d.Hour(), d.Minute())

}

func populateDB(s tcell.Screen, db *bolt.DB) {
	dir, err := os.UserConfigDir()
	fatal(158, err)
	file, err := os.Open(filepath.FromSlash(dir + filepath.FromSlash("/lydia/urls")))
	if err != nil {
		log.Fatalf("You need to create a new file - %s, and add a few feeds in it, one per line", filepath.FromSlash(dir+filepath.FromSlash("/lydia/urls")))
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var items []Item = make([]Item, 0, 256)
	i := 0
	for scanner.Scan() {
		text := scanner.Text()
		if text[0] == '#' {
			continue
		}
		w, h := s.Size()
		style := tcell.StyleDefault.Bold(true).Reverse(true)
		print(s, w-12, h-1, style, fmt.Sprintf("loading...%02d", i))
		s.Show()
		f, err := rss.Fetch(scanner.Text())
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s could not be fetched", scanner.Text())
			continue
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
		i++
	}
	fatal(186, scanner.Err())
	err = db.Update(func(tx *bolt.Tx) error {
		unreadBucket, err := tx.CreateBucketIfNotExists([]byte("unread"))
		if err != nil {
			return err
		}
		readBucket, err := tx.CreateBucketIfNotExists([]byte("read"))
		if err != nil {
			return err
		}
		for _, i := range items {
			if leng(i.Title) > 20 {
				rs := []rune(i.Title)
				i.Title = string(rs[:20])
			}
			buf, err := json.Marshal(i)
			if err != nil {
				return err
			}
			if key := readBucket.Get([]byte(fmt.Sprintf("%d", i.I.Date.Local().Unix()) + i.I.Link)); key != nil {
				continue
			}
			if key := unreadBucket.Get([]byte(fmt.Sprintf("%d", i.I.Date.Local().Unix()) + i.I.Link)); key == nil {
				unreadBucket.Put([]byte(fmt.Sprintf("%d", i.I.Date.Local().Unix())+i.I.Link), buf)
			} else if key[READ_OFFSET] != ZERO {
				unreadBucket.Delete([]byte(fmt.Sprintf("%d", i.I.Date.Local().Unix()) + i.I.Link))
				readBucket.Put([]byte(fmt.Sprintf("%d", i.I.Date.Local().Unix())+i.I.Link), buf)
			}
		}
		return nil
	})
	fatal(204, err)
}

func openURL(db *bolt.DB, index int) {
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
	var url string
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("unread"))
		if b == nil {
			fatal(222, errors.New("no bucket"))
		}
		c := b.Cursor()
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			index--
			if k == nil {
				fatal(230, errors.New("no key for current item"))
			}
			if index == -1 {
				var item Item
				err := json.Unmarshal(v, &item)
				if err != nil {
					return err
				}
				url = item.I.Link
				item.Read = 1
				buf, err := json.Marshal(item)
				if err != nil {
					return err
				}
				b.Put(k, buf)
				break
			}
		}
		return nil
	})
	args = append(args, url)
	err = exec.Command(cmd, args...).Start()
	fatal(221, err)
}

func scroll(db *bolt.DB, s tcell.Screen, item *int) {
	w, _ := s.Size()
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("unread"))
		c := bucket.Cursor()
		y := 0
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			read := v[READ_OFFSET]
			var i Item
			err := json.Unmarshal(v, &i)
			fatal(242, err)
			var result string
			var style tcell.Style
			if read == ZERO {
				result = fmt.Sprintf("%6s│%[2]*s│%s", date(i.I.Date.Local()), 20, i.Title, i.I.Title)
				style = tcell.StyleDefault.Bold(true)
			} else {
				result = fmt.Sprintf("%6s│%[2]*s│%s", date(i.I.Date.Local()), 20, i.Title, i.I.Title)
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
			y++
		}
		return nil
	})
	fatal(264, err)
}

func print(s tcell.Screen, x, y int, style tcell.Style, text string) {
	for _, c := range text {
		s.SetCell(x, y, style, c)
		x += runewidth.RuneWidth(c)
	}
}

func fatal(line int, e error) {
	if e != nil {
		log.Fatal(line, e)
	}
}
