package main

import (
	"fmt"
	"log"

	"time"

	"github.com/pgvector/pgvector-go"
	"github.com/rnr-capital/newsfeed-backend/model"
	"github.com/rnr-capital/newsfeed-backend/publisher"
	"github.com/rnr-capital/newsfeed-backend/utils"
	"github.com/rnr-capital/newsfeed-backend/utils/dotenv"
	. "github.com/rnr-capital/newsfeed-backend/utils/flag"
	. "github.com/rnr-capital/newsfeed-backend/utils/log"
)

func cleanup() {
	LogV2.Info("bot server shutdown")
}

func main() {
	defer cleanup()
	ParseFlags()

	if err := dotenv.LoadDotEnvs(); err != nil {
		panic(err)
	}

	db, err := utils.GetDBConnection()
	utils.BotDBSetupAndMigration(db)
	if err != nil {
		panic("failed to connect to database")
	}

	var posts []model.Post
	err = db.Select("id, content, title").Where("embedding IS NULL").Order("cursor desc").Limit(100000).Find(&posts).Error
	if err != nil {
		log.Fatal(err)
	}

	print(len(posts))
	for i, p := range posts {
		try := 1
	RETRY:
		if i%100 == 0 {
			time.Sleep(5 * time.Second)
		}
		try += 1
		if p.Title != "" || p.Content != "" {
			text := p.Title + p.Content
			if len(text) > 15000 {
				text = text[0:15000]
			}

			embedding, err := publisher.CalculateEmbedding(text)
			if err != nil || len(embedding) != 100 {
				log.Print(err)
				time.Sleep(time.Duration(10*try) * time.Second)
				goto RETRY
			}
			r := db.Model(&p).Update("embedding", pgvector.NewVector(embedding))
			if r.RowsAffected != 1 {
				log.Print(r.Error)
				time.Sleep(time.Duration(10*try) * time.Second)
				goto RETRY
			}
		}
	}
	fmt.Println("done")
}
