package bot

// This handler is for slack slash commands
// https://api.slack.com/interactivity/slash-commands

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	govector "github.com/drewlanenga/govector"
	"github.com/rnr-capital/newsfeed-backend/model"
	Logger "github.com/rnr-capital/newsfeed-backend/utils/log"
)

const (
	SimilarityThreshold   = 0.4
	SimilarityWindowHours = 1
)

var PostsSent map[string][]postMeta
var Mutex sync.Mutex

func init() {
	PostsSent = map[string][]postMeta{}
}

type postMeta struct {
	Id                 string `json:"id"`
	Embedding          govector.Vector
	ContentGeneratedAt time.Time
}

type SharePostPayload struct {
	Post       model.Post
	FromUser   string `json:"from_user"`
	WebhookUrl string `json:"webhook_url"`
	Comment    string `json:"comment"`
}

func isEmbeddingSimilar(e1 govector.Vector, e2 govector.Vector) bool {
	// If the hashing is invalid, or not of same length, they cannot be considered
	// as the semantically identical.
	diff, _ := e1.Subtract(e2)
	norm := govector.Norm(diff, 2)
	return math.Sqrt(norm) <= SimilarityThreshold
}

func isPostDuplicated(
	post model.Post,
	channelId string,
) bool {
	Mutex.Lock()
	defer Mutex.Unlock()

	Logger.LogV2.Info(fmt.Sprintf("Got a dedup request for %s in channel %s", post.Id, channelId))
	_, ok := PostsSent[channelId]
	if !ok {
		return false
	}
	Logger.LogV2.Info(fmt.Sprintf("Current channel %s, posts: %v, size %d", channelId, PostsSent[channelId], len(PostsSent[channelId])))

	for i := len(PostsSent[channelId]) - 1; i >= 0; i-- {
		p := PostsSent[channelId][i]
		// the collector has some interval(up to 12 hours for zsxq) to collect the data
		// we will keep the cache for two days
		if math.Abs(time.Since(p.ContentGeneratedAt).Hours()) > 48 {
			PostsSent[channelId] = append(PostsSent[channelId][:i], PostsSent[channelId][i+1:]...)
		}

		if post.Embedding == nil || p.Embedding == nil {
			return false
		}

		if len(post.Embedding.Slice()) == 0 ||
			len(p.Embedding) == 0 {
			return false
		}

		if (math.Abs(post.ContentGeneratedAt.Sub(p.ContentGeneratedAt).Hours())) < SimilarityWindowHours {
			vec, _ := govector.AsVector(post.Embedding.Slice())
			Logger.LogV2.Info(fmt.Sprintf("Comparing %s with %s, %v, %v", post.Id, p.Id, vec, p.Embedding))
			if isEmbeddingSimilar(vec, p.Embedding) {
				return true
			}
		}
	}

	return false
}

func parsePostSharePayload(body io.ReadCloser, db *gorm.DB) (*SharePostPayload, error) {
	bodybytes, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, err
	}

	payload := SharePostPayload{}

	err = json.Unmarshal(bodybytes, &payload)
	if err != nil {
		return nil, err
	}

	if payload.FromUser != "" {
		var post model.Post
		db.Preload("SubSource").Preload("SharedFromPost").Preload("SharedFromPost.SubSource").Where("id=?", payload.Post.Id).First(&post)
		Logger.LogV2.Info(fmt.Sprintf("post %s embedding %v, size %d", post.Id, post.Embedding, len(post.Embedding.Slice())))
		payload.Post = post
	}

	return &payload, nil
}

func PostShareHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		payload, err := parsePostSharePayload(c.Request.Body, db)
		if err != nil {
			bodybytes, _ := ioutil.ReadAll(c.Request.Body)
			Logger.LogV2.Error(fmt.Sprint("invalid post share payload", err, string(bodybytes)))
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}

		if payload.FromUser == "" {
			if isPostDuplicated(payload.Post, payload.WebhookUrl) {
				c.Data(202, "application/json; charset=utf-8", []byte("Post duplicated"))
				return
			}
		}

		if err := PushPostViaWebhook(payload.Post, payload.WebhookUrl, payload.FromUser, payload.Comment); err != nil {
			Logger.LogV2.Error(fmt.Sprint("Fail to post via webhook", payload.WebhookUrl, err))
		}

		Mutex.Lock()
		defer Mutex.Unlock()

		vec, _ := govector.AsVector([]float32{})
		if payload.Post.Embedding != nil {
			vec, _ = govector.AsVector(payload.Post.Embedding.Slice())
		}
		if posts, ok := PostsSent[payload.WebhookUrl]; ok {
			PostsSent[payload.WebhookUrl] = append(posts,
				postMeta{
					Id:                 payload.Post.Id,
					Embedding:          vec,
					ContentGeneratedAt: payload.Post.ContentGeneratedAt,
				})
		} else {
			PostsSent[payload.WebhookUrl] = []postMeta{
				{
					Id:                 payload.Post.Id,
					Embedding:          vec,
					ContentGeneratedAt: payload.Post.ContentGeneratedAt,
				},
			}
		}

		c.Data(200, "application/json; charset=utf-8", []byte("Post sent"))
	}
}
