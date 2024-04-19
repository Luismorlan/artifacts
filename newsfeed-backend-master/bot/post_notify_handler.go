package bot

// This handler is for slack slash commands
// https://api.slack.com/interactivity/slash-commands

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/rnr-capital/newsfeed-backend/model"
	"github.com/rnr-capital/newsfeed-backend/notifier"
	"github.com/rnr-capital/newsfeed-backend/notifier/consumers"

	Logger "github.com/rnr-capital/newsfeed-backend/utils/log"
)

func init() {
	PostsSent = map[string][]postMeta{}
}

type PostNotifyPayload struct {
	Post    model.Post
	Columns []model.Column
}

func parsePostNotifyPayload(body io.ReadCloser) (*PostNotifyPayload, error) {
	bodybytes, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, err
	}

	payload := PostNotifyPayload{}

	err = json.Unmarshal(bodybytes, &payload)
	if err != nil {
		return nil, err
	}
	return &payload, nil
}

func PostNotifyHandler(db *gorm.DB) gin.HandlerFunc {
	notifier := notifier.NewNotifier(consumers.NewOneSignalAdapter(), 90*time.Second, 100, 100, time.Hour*12)
	go notifier.Start()
	return func(c *gin.Context) {
		payload, err := parsePostNotifyPayload(c.Request.Body)
		if err != nil {
			bodybytes, _ := ioutil.ReadAll(c.Request.Body)
			Logger.LogV2.Error(fmt.Sprint("invalid post share payload", err, string(bodybytes)))
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}
		notifier.AddIntakeJob(payload.Post, payload.Columns)
		c.Data(200, "application/json; charset=utf-8", []byte("Post sent"))
	}
}
