package main

import (
	"sync"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rnr-capital/newsfeed-backend/bot"
	"github.com/rnr-capital/newsfeed-backend/utils"
	"github.com/rnr-capital/newsfeed-backend/utils/dotenv"
	. "github.com/rnr-capital/newsfeed-backend/utils/flag"
	. "github.com/rnr-capital/newsfeed-backend/utils/log"
	gintrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/gin-gonic/gin"
)

func cleanup() {
	LogV2.Info("bot server shutdown")
}

func main() {
	ParseFlags()
	defer cleanup()

	if err := dotenv.LoadDotEnvs(); err != nil {
		panic(err)
	}

	// Default With the Logger and Recovery middleware already attached
	router := gin.Default()

	router.Use(cors.Default())
	router.Use(gintrace.Middleware(*ServiceName))

	db, err := utils.GetDBConnection()
	utils.BotDBSetupAndMigration(db)
	// post := &model.Post{}
	// db.Preload("SubSource").Preload("SharedFromPost").Preload("SharedFromPost.SubSource").Where("id='748423e8-9ea5-4036-89bd-611e37083560'").First(&post)
	// fmt.Println("post", *post)
	if err != nil {
		panic("failed to connect to database")
	}
	mu := &sync.Mutex{}

	router.POST("/bot/notifypost", bot.PostNotifyHandler(db))

	router.GET("/bot/auth", bot.AuthHandler(db))

	router.POST("/bot/cmd", bot.SlashCommandHandler(db, mu))

	router.POST("/bot/interaction", bot.InteractionHandler(db))

	router.POST("/bot/sharepost", bot.PostShareHandler(db))

	router.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"message": "Newsfeed server - API not found"})
	})

	LogV2.Info("bot server starts up")
	router.Run(":9090")
}
