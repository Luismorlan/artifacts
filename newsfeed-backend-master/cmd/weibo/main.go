package main

import (
	"fmt"

	"github.com/rnr-capital/newsfeed-backend/bot/articlesaver"
	"github.com/rnr-capital/newsfeed-backend/utils/dotenv"
)

const url = "https://weibo.com/ttarticle/p/show?id=2309404916648472871468"

func main() {
	if err := dotenv.LoadDotEnvs(); err != nil {
		panic(err)
	}
	doc, err := articlesaver.GetWeiboArticle(url)
	if err != nil {
		fmt.Println("failed to get weibo article", err)
		return
	}
	link, err := articlesaver.SaveWeiboDocToNotion(doc, url)
	if err != nil {
		fmt.Println("failed to save weibo article", err)
		return
	}
	fmt.Println(link)
}
