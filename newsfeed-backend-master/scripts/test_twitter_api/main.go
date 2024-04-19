package main

import (
	"fmt"

	twitterscraper "github.com/n0madic/twitter-scraper"
	"github.com/rnr-capital/newsfeed-backend/collector"
)

func getAllTweets(name string) {
	tweets, _, _ := twitterscraper.New().FetchTweets(name, 50, "")
	// fmt.Println(collector.PrettyPrint(tweets[2]))
	for _, t := range tweets {
		fmt.Println(collector.PrettyPrint(t))
	}
}

func main() {
	name := "elonmusk"
	getAllTweets(name)
}
