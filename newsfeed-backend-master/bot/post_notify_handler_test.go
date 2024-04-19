package bot_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/rnr-capital/newsfeed-backend/bot"
	"github.com/rnr-capital/newsfeed-backend/model"
	"github.com/stretchr/testify/assert"
)

func TestEncodeAndDecodePostNotifyPayload(t *testing.T) {
	users := []*model.User{
		{
			Id: "userId-1",
		},
		{
			Id: "userId-2",
		},
		{
			Id: "userId-3",
		},
		{
			Id: "userId-4",
		},
		{
			Id: "userId-5",
		},
		{
			Id: "userId-6",
		},
	}
	post := model.Post{
		Id:          "294a136e-416f-4bf5-865e-9a26484e3c4f",
		Title:       "",
		Content:     "如果这样的风气不能被遏制，法律不能保护好人，和谐社会就是一句空话，敲诈勒索，寻衅滋事，这两条我看都符合。//@响马:操",
		SubSourceID: "c2bdedc6-671d-47f3-8505-ec3091808f29",
		SubSource: model.SubSource{
			Name: "caoz",
			Id:   "c2bdedc6-671d-47f3-8505-ec3091808f29",
		},
		ImageUrls: []string{},
		Tag:       "",
	}
	columns := []model.Column{
		{
			Id:          "ColumnId-1",
			CreatedAt:   time.Date(2022, 1, 25, 12, 30, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2022, 1, 25, 12, 30, 0, 0, time.UTC),
			CreatorID:   "CreatorID-1",
			Subscribers: users[0:3],
		},
		{
			Id:          "ColumnId-2",
			CreatedAt:   time.Date(2022, 1, 25, 12, 30, 1, 0, time.UTC),
			UpdatedAt:   time.Date(2022, 1, 25, 12, 30, 1, 0, time.UTC),
			CreatorID:   "CreatorID-2",
			Subscribers: users[2:5],
		},
		{
			Id:          "ColumnId-3",
			CreatedAt:   time.Date(2022, 1, 25, 12, 30, 2, 0, time.UTC),
			UpdatedAt:   time.Date(2022, 1, 25, 12, 30, 2, 0, time.UTC),
			CreatorID:   "CreatorID-3",
			Subscribers: users[5:6],
		},
	}

	sharePost := bot.PostNotifyPayload{
		Post:    post,
		Columns: columns,
	}
	encodeBytes, err := json.Marshal(sharePost)
	assert.Nil(t, err)
	assert.NotEmpty(t, encodeBytes)

	decoded := bot.PostNotifyPayload{}

	err = json.Unmarshal(encodeBytes, &decoded)
	assert.Nil(t, err)
	assert.Equal(t, len(decoded.Columns), 3)
	assert.Equal(t, len(decoded.Columns[0].Subscribers), 3)
	assert.Equal(t, decoded.Post.SubSource.Name, "caoz")
}
