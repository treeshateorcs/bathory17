package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"

	"github.com/SlyMarbo/rss"
	"github.com/gdamore/tcell"
	"github.com/mattn/go-runewidth"
)

func main() {
	var url string
	if len(os.Args) == 1 {
		url = "http://carlgene.com/blog/feed/atom/"
	} else {
		url = os.Args[1]
	}
	feed, err := rss.Fetch(url)
	fatal(err)
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
					openURL(feed.Items[currentItem].Link)
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
		scroll(s, currentItem, feed)
		s.Sync()
	}
}

func openURL(url string) error {
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
	return exec.Command(cmd, args...).Start()
}

func scroll(s tcell.Screen, item int, feed *rss.Feed) {
	//w, h := s.Size()
	for y, f := range feed.Items {
		result := fmt.Sprintf("%4d%2s%s", y, "N", f.Title)
		if y == item {
			print(s, 0, y, tcell.StyleDefault.Reverse(true), result)
		} else {
			print(s, 0, y, tcell.StyleDefault, result)
		}
	}
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
