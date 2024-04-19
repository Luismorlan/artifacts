package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	Twitter "github.com/rnr-capital/newsfeed-backend/collector/webhook/twitter"
	Flag "github.com/rnr-capital/newsfeed-backend/utils/flag"
	Logger "github.com/rnr-capital/newsfeed-backend/utils/log"
)

func main() {
	Flag.ParseFlags()

	router := gin.Default()

	// Add a debug route for testing and health check
	router.GET("/webhook/ping", func(c *gin.Context) {
		c.JSON(http.StatusAccepted, "pong")
	})

	AddTwitterWebhook(router.Group("/webhook"))
	// Additional webhooks should be added below this line

	Logger.LogV2.Info(fmt.Sprint("===== Webhook Server Started ====="))
	router.Run(":7070")
}

func AddTwitterWebhook(rg *gin.RouterGroup) {
	twitter := rg.Group("/twitter")

	twitter.GET("/", Twitter.HandleTwitterCRC)
	twitter.POST("/", Twitter.HandleTwitterMessage)
}
