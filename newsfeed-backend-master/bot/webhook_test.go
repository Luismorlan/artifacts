package bot

import (
	"encoding/json"
	"testing"

	"github.com/pgvector/pgvector-go"
	"github.com/rnr-capital/newsfeed-backend/model"
	"github.com/stretchr/testify/assert"
)

func TestTimeBoundedPushPost(t *testing.T) {

	t.Run("pushes post within timeout", func(t *testing.T) {
		webhookUrl := "http://example.com/webhook"

		// Mock post
		vec := pgvector.NewVector([]float32{1, 2, 3})

		post := model.Post{Id: "aaa", Embedding: &vec}
		sharePost := SharePostPayload{
			Post:       post,
			WebhookUrl: webhookUrl,
			FromUser:   "Test",
		}
		postBytes, err := json.Marshal(sharePost)
		assert.Nil(t, err)
		p2 := SharePostPayload{}
		err = json.Unmarshal(postBytes, &p2)
		assert.Nil(t, err)
		assert.EqualValues(t, p2.Post.Embedding.Slice(), []float32{1, 2, 3})
		assert.Equal(t, p2.FromUser, "Test")

		notifPost := PostNotifyPayload{
			Post:    post,
			Columns: []model.Column{{Id: "TEST1"}},
		}
		notifPostBytes, err := json.Marshal(notifPost)
		assert.Nil(t, err)

		p3 := PostNotifyPayload{}
		err = json.Unmarshal(notifPostBytes, &p3)
		assert.Nil(t, err)

		assert.EqualValues(t, p2.Post.Embedding.Slice(), []float32{1, 2, 3})

	})

	t.Run("nil embedding shouldn't throw an error", func(t *testing.T) {
		webhookUrl := "http://example.com/webhook"

		post := model.Post{Id: "aaa", Embedding: nil}
		sharePost := SharePostPayload{
			Post:       post,
			WebhookUrl: webhookUrl,
			FromUser:   "Test",
		}
		postBytes, err := json.Marshal(sharePost)
		assert.Nil(t, err)
		p2 := SharePostPayload{}
		err = json.Unmarshal(postBytes, &p2)
		assert.Nil(t, err)
		assert.Nil(t, p2.Post.Embedding)
		assert.Equal(t, p2.FromUser, "Test")
	})


}
