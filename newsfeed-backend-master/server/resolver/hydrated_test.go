package resolver

import (
	"testing"
	"time"

	"github.com/rnr-capital/newsfeed-backend/model"
	"github.com/rnr-capital/newsfeed-backend/utils"
	"github.com/stretchr/testify/require"
)

const BoningUserId = "2ef1afbc-6ce3-4493-9a6b-e03e421d5066"
const TestingUserId = "0b799c87-589a-4d71-855e-f1fa05011d15"
const WisburgFeedId = "a1ae74b4-ce65-4c07-b7b8-c812d2d3e9d0"
const SocialMediaFeedId = "d5493c7d-3fff-422a-a8a9-64011583ecb1"

func createQueryResolver(t *testing.T) *queryResolver {
	redis, err := utils.GetRedisStatusStore()
	require.NoError(t, err)
	db, err := utils.GetTestingDBConnection()
	utils.DatabaseSetupAndMigration(db)
	require.NoError(t, err)
	return &queryResolver{&Resolver{DB: db, RedisStatusStore: redis, SignalChans: nil}}
}

func TestBasicOneFeed(t *testing.T) {
	r := createQueryResolver(t)
	input := &model.FeedRefreshInput{
		FeedID:    WisburgFeedId,
		Limit:     10,
		Cursor:    0,
		Direction: model.FeedRefreshDirectionNew,
	}

	// owner
	feeds, err := getRefreshFeedPosts(r, []*model.FeedRefreshInput{input}, TestingUserId)
	require.NoError(t, err)
	require.Len(t, feeds, 1)
	require.Len(t, feeds[0].Posts, 10)

	// didn't subscribe
	feeds, err = getRefreshFeedPosts(r, []*model.FeedRefreshInput{input}, BoningUserId)
	require.NoError(t, err)
	require.Len(t, feeds, 1)
	require.Len(t, feeds[0].Posts, 10)
}

func TestMultiFeeds(t *testing.T) {
	r := createQueryResolver(t)
	wisburgInput := &model.FeedRefreshInput{
		FeedID:    WisburgFeedId,
		Limit:     10,
		Cursor:    0,
		Direction: model.FeedRefreshDirectionNew,
	}
	socialMediaInput := &model.FeedRefreshInput{
		FeedID:    SocialMediaFeedId,
		Limit:     10,
		Cursor:    0,
		Direction: model.FeedRefreshDirectionNew,
	}

	feeds, err := getRefreshFeedPosts(r, []*model.FeedRefreshInput{wisburgInput, socialMediaInput}, TestingUserId)
	require.NoError(t, err)
	require.Len(t, feeds, 2)
	require.Len(t, feeds[0].Posts, 10)
	require.Len(t, feeds[1].Posts, 10)
}

func TestColumn(t *testing.T) {
	// normal case
	r := createQueryResolver(t)
	quick := &model.ColumnRefreshInput{
		ColumnID:  "3469498d-830a-4fc9-b7b0-8f9ababf1854",
		Limit:     10,
		Cursor:    0,
		Direction: model.FeedRefreshDirectionNew,
	}

	col, err := getRefreshColumnPosts(r, []*model.ColumnRefreshInput{quick}, TestingUserId)
	require.NoError(t, err)
	require.Len(t, col, 1)
	require.Len(t, col[0].Feeds, 2)
	if col[0].Feeds[0].Id == "ec5a316e-264d-4f02-a696-1264b5d52f35" {
		require.Len(t, col[0].Feeds[0].Posts, 0)

		timeString := "2023-06-27T15:04:58-07:00"
		// Define the layout format of the time string
		layout := "2006-01-02T15:04:05-07:00"

		// Parse the time string into a time.Time value
		timeValue, err := time.Parse(layout, timeString)
		require.NoError(t, err)
		require.Len(t, col[0].Feeds[0].Posts, 0)
		require.Len(t, col[0].Feeds[1].Posts, 10)
		require.Equal(t, col[0].Feeds[1].Posts[0].ContentGeneratedAt, timeValue)
	} else {
		require.Len(t, col[0].Feeds[0].Posts, 10)
		require.Len(t, col[0].Feeds[1].Posts, 0)
	}
}
