package main

import (
	"fmt"
	"log"

	"github.com/SlyMarbo/rss"
)

func main() {
	feed, err := rss.Fetch("https://github.com/qutebrowser/qutebrowser/releases.atom")
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range feed.Items {
		fmt.Println(f.Title)
	}
}
