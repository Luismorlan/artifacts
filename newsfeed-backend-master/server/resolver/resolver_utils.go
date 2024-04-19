package resolver

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/prototext"
	"gorm.io/gorm"

	"github.com/rnr-capital/newsfeed-backend/model"
	"github.com/rnr-capital/newsfeed-backend/protocol"
	"github.com/rnr-capital/newsfeed-backend/utils"
	Logger "github.com/rnr-capital/newsfeed-backend/utils/log"
)

const (
	feedRefreshLimit           = 300
	defaultFeedsQueryCursor    = math.MaxInt32
	defaultFeedsQueryDirection = model.FeedRefreshDirectionOld
)

// Given a list of FeedRefreshInput, get posts for the requested feeds
// Do it by iterating through feeds
func getRefreshFeedPosts(r *queryResolver, queries []*model.FeedRefreshInput, userId string) ([]*model.Feed, error) {
	results := []*model.Feed{}
	var wg sync.WaitGroup
	var res sync.Map
	for idx := range queries {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			q := queries[i]
			if q == nil {
				// This is not expected since gqlgen guarantees it is not nil
				return
			}
			// Prepare feed basic info
			var feed model.Feed
			queryResult := r.DB.Preload("SubSources").Where("id = ?", q.FeedID).First(&feed)
			if queryResult.RowsAffected != 1 {
				Logger.LogV2.Error(fmt.Sprintf("invalid feed id %s", q.FeedID))
				return
			}
			if err := sanitizeFeedRefreshInput(q, &feed); err != nil {
				Logger.LogV2.Error(fmt.Sprintf("feed q invalid %v", q))
				return
			}
			if err := getFeedPostsOrRePublish(r.DB, r.RedisStatusStore, &feed, q, userId); err != nil {
				Logger.LogV2.Error(fmt.Sprintf("failure when get posts for feed id %s", feed.Id))
				return
			}
			res.Store(i, &feed)
		}(idx)
	}
	wg.Wait()
	for idx := range queries {
		if feed, ok := res.Load(idx); ok {
			results = append(results, feed.(*model.Feed))
		}
	}
	return results, nil
}

// Given a list of FeedRefreshInput, get posts for the requested feeds
// Do it by iterating through feeds
func getRefreshColumnPosts(r *queryResolver, queries []*model.ColumnRefreshInput, userId string) ([]*model.Column, error) {
	results := []*model.Column{}
	var wg sync.WaitGroup
	var res sync.Map
	for idx := range queries {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			query := queries[i]
			if query == nil {
				// This is not expected since gqlgen guarantees it is not nil
				return
			}
			// Prepare feed basic info
			var column model.Column
			queryResult := r.DB.Preload("Feeds").Where("id = ?", query.ColumnID).First(&column)
			if queryResult.RowsAffected != 1 {
				Logger.LogV2.Error(fmt.Sprintf("invalid column id %s", query.ColumnID))
				return
			}
			if err := sanitizeColumnRefreshInput(query, &column); err != nil {
				Logger.LogV2.Error(fmt.Sprintf("column query invalid %v", query))
				return
			}
			feedRefreshInputs := []*model.FeedRefreshInput{}
			for i := range query.FeedIds {
				feedRefreshInputs = append(feedRefreshInputs, &model.FeedRefreshInput{
					FeedID:          query.FeedIds[i],
					Query:           query.Query,
					Direction:       query.Direction,
					FeedUpdatedTime: query.FeedUpdatedTimes[i],
					Cursor:          query.Cursor,
					Limit:           query.Limit,
					Filter:          query.Filter,
				})
			}
			feeds, err := getRefreshFeedPosts(r, feedRefreshInputs, userId)
			if err != nil {
				Logger.LogV2.Error(fmt.Sprint("feed query failed", query))
				return
			}
			if query.OtherEndCursor != nil {
				var smallCursor, largeCursor int
				if *query.OtherEndCursor > query.Cursor {
					smallCursor = query.Cursor
					largeCursor = *query.OtherEndCursor
				} else {
					smallCursor = *query.OtherEndCursor
					largeCursor = query.Cursor
				}
				var allPosts []string
				allPosts, err = r.RedisStatusStore.GetColumnPosts(query.ColumnID, int32(smallCursor), int32(largeCursor))
				if err != nil {
					Logger.LogV2.Error("failed to get postIds of column: " + err.Error())
				}
				res, err := r.RedisStatusStore.GetItemsReadStatus(allPosts, userId)
				if err != nil {
					Logger.LogV2.Error(fmt.Sprintf("failed to get read status. %v", err))
				} else if len(res) != len(allPosts) {
					Logger.LogV2.Error(fmt.Sprintf("read status has different length then posts. res length: %d, posts length: %d", len(res), len(allPosts)))
				} else {
					readed := []string{}
					for i := 0; i < len(res); i++ {
						if res[i] {
							readed = append(readed, allPosts[i])
						}
					}
					column.Readed = readed
				}
			}
			cursors := map[int]*model.Post{}
			selected := map[int]bool{}
			for _, f := range feeds {
				for _, p := range f.Posts {
					cursors[int(p.Cursor)] = p
				}
			}
			sorted := []*model.Post{}
			if len(sorted) == 0 {
				Logger.LogV2.Error("Something went wrong, sorted is empty. ColumnID: " + query.ColumnID)
			}
			for _, p := range cursors {
				sorted = append(sorted, p)
			}
			sort.Slice(sorted, func(i, j int) bool {
				return sorted[i].ContentGeneratedAt.After(sorted[j].ContentGeneratedAt)
			})
			for i := 0; i < query.Limit; i++ {
				if i >= len(sorted) {
					break
				}
				selected[int(sorted[i].Cursor)] = true
			}
			for i := range feeds {
				newPosts := []*model.Post{}
				for _, p := range feeds[i].Posts {
					if reserve := selected[int(p.Cursor)]; reserve {
						newPosts = append(newPosts, p)
					}
				}
				feeds[i].Posts = newPosts
			}
			column.Feeds = feeds
			res.Store(i, &column)
		}(idx)
	}
	wg.Wait()
	for idx := range queries {
		if column, ok := res.Load(idx); ok {
			results = append(results, column.(*model.Column))
		}
	}
	return results, nil
}

// TODO: add unit test
func getFeedPostsOrRePublish(db *gorm.DB, r *utils.RedisStatusStore, feed *model.Feed, query *model.FeedRefreshInput, userId string) error {
	var posts []*model.Post
	// try to read published posts
	var cursorPost *model.Post
	var found int64
	var cursorPublishedTime time.Time

	if query.Direction == model.FeedRefreshDirectionNew {
		cursorPublishedTime = time.Unix(0, 0)
	} else {
		cursorPublishedTime = time.Now()
	}
	startQuery := time.Now()
	if query.Cursor != 2147483647 {
		db.Model(&model.Post{}).Where("cursor = ?", query.Cursor).First(&cursorPost).Count(&found)
		if found > 0 {
			cursorPublishedTime = cursorPost.ContentGeneratedAt
		} else {
			if query.Direction == model.FeedRefreshDirectionNew && query.Cursor != 0 {
				return errors.New("got an empty post using cursor which should never happen")
			}
		}
	}

	dbQuery := db.Model(&model.Post{}).
		Preload("SubSource").
		Preload("SharedFromPost").
		Preload("SharedFromPost.SubSource").
		// Preload("ReadByUser").
		// Maintain a chronological order of reply thread.
		Preload("ReplyThread", func(db *gorm.DB) *gorm.DB {
			return db.Order("posts.created_at ASC")
		}).
		Preload("ReplyThread.SubSource").
		Preload("ReplyThread.SharedFromPost").
		Preload("ReplyThread.SharedFromPost.SubSource").
		Joins("LEFT JOIN post_feed_publishes ON post_feed_publishes.post_id = posts.id").
		Joins("LEFT JOIN feeds ON post_feed_publishes.feed_id = feeds.id").
		Joins("LEFT JOIN sub_sources ON posts.sub_source_id = sub_sources.id").
		Joins("LEFT JOIN user_post_reads ON user_post_reads.post_id = posts.id AND user_post_reads.user_id = ?", userId)

	if query.Filter != nil && *query.Filter.Unread {
		dbQuery.Where("user_post_reads.post_id IS NULL")
	}

	dbQuery.Where("feed_id = ?", feed.Id)

	if query.Direction == model.FeedRefreshDirectionNew {
		dbQuery.Where("posts.cursor > ?", query.Cursor)
	} else {
		dbQuery.Where("posts.content_generated_at < ?", cursorPublishedTime)
	}

	if query.Query != nil && len(*query.Query) != 0 {
		dbQuery.Where("COALESCE(posts.content, '') ILIKE '%' || ? || '%' OR COALESCE(posts.title, '') ILIKE '%' || ? || '%' OR COALESCE(sub_sources.name, '') ILIKE '%' || ? || '%'",
			*query.Query, *query.Query, *query.Query)
	}

	dbQuery.Order("posts.content_generated_at desc").Limit(query.Limit).Find(&posts)
	if query.Direction == model.FeedRefreshDirectionNew {
		for _, post := range posts {
			if post.ContentGeneratedAt.Before(cursorPublishedTime) {
				post.Delayed = true
			}
		}
	}
	feed.Posts = posts

	if query.Direction == model.FeedRefreshDirectionOld {
		if len(posts) < query.Limit {
			// query OLD but can't satisfy the limit, republish in this case
			lastPublished := cursorPublishedTime
			if len(posts) > 0 {
				lastPublished = posts[len(posts)-1].ContentGeneratedAt
			}
			Logger.LogV2.Info(fmt.Sprintf("run ondemand publish posts to feed: %s. triggered by OLD in {feeds} API from lastPublished %v try to republish %d more posts",
				feed.Id, lastPublished, query.Limit-len(posts)))
			before := len(posts)
			rePublishPostsBefore(db, feed, query.Limit-len(posts), lastPublished, query.Query)
			Logger.LogV2.Info(fmt.Sprintf("republished %d posts for feed %s", len(posts)-before, feed.Id))
		}
	}
	elapsedQueryTime := time.Since(startQuery)
	Logger.LogV2.Info(fmt.Sprintf("Query columns execution time:  %v, userId is: %v", elapsedQueryTime, userId))
	sortPostsByCreationTime(feed.Posts)
	// update feed read status from redis

	postIds := []string{}
	for _, post := range feed.Posts {
		postIds = append(postIds, post.Id)
	}

	status, err := r.GetItemsReadStatus(postIds, userId)
	if err != nil {
		return errors.Wrap(err, "failure when gettting posts read status")
	}

	for idx := range feed.Posts {
		feed.Posts[idx].IsRead = status[idx]
	}

	if query.Filter != nil && *query.Filter.Unread {
		for i := len(feed.Posts) - 1; i >= 0; i -= 1 {
			if feed.Posts[i].IsRead {
				feed.Posts = append(feed.Posts[:i], feed.Posts[i+1:]...)
			}
		}
	}
	return nil
}

// Sort a batch by content_generated_at (instead of by cursor) so that
// we guarantee this batch is chronologically descreasing. Frontend should
// process the entire batch to find max/min cursor instead of relying only
// on the first and the last returned item. Same for below.
func sortPostsByCreationTime(posts []*model.Post) {
	// Maintain chronological order
	sort.SliceStable(posts, func(i, j int) bool {
		return posts[i].ContentGeneratedAt.After(posts[j].ContentGeneratedAt)
	})
}

// Redo posts publish to feeds
// From a particular cursor down
// If cursor is -1, republish from NEWest
func rePublishPostsBefore(db *gorm.DB, feed *model.Feed, limit int, contentGeneratedTime time.Time, query *string) {
	var subsourceIds []string
	for _, subsource := range feed.SubSources {
		subsourceIds = append(subsourceIds, subsource.Id)
	}

	dataExpression, err := utils.ParseDataExpression(string(feed.FilterDataExpression))
	if err != nil {
		Logger.LogV2.Error(fmt.Sprintf("Failed to parse data expression. %v. %v", feed.FilterDataExpression, err))
	}
	sql, err := utils.DataExpressionToSql(dataExpression)

	Logger.LogV2.Info(fmt.Sprintf("Sql Generated: %s", sql))
	if err != nil {
		Logger.LogV2.Error(fmt.Sprintf("Failed to convert data expression to sql. %v. %v", feed.FilterDataExpression, err))
	}

	var postsToPublish []*model.Post

	// 1. Read subsources' most recent posts
	// 2. skip if post is shared by another one, this used to handle case as retweet
	// 	  this will also work, if in future we will support user generate comments on other user posts
	//    the shared post creation and publish is in one transaction, so the sharing can only happen
	//    after the shared one is published.
	//    however for re-publish,
	q := db.Model(&model.Post{}).
		Preload("SubSource").
		Preload("SharedFromPost").
		Preload("SharedFromPost.SubSource").
		Preload("ReplyThread", func(db *gorm.DB) *gorm.DB {
			return db.Order("posts.created_at ASC")
		}).
		Preload("ReplyThread.SubSource").
		Joins(`LEFT JOIN posts "shared_from_post" ON shared_from_post.id = posts.shared_from_post_id`).
		Where("posts.sub_source_id IN ? AND posts.content_generated_at < ? AND (NOT posts.in_sharing_chain)", subsourceIds, contentGeneratedTime)

	if query != nil && len(*query) != 0 {
		q.Where("COALESCE(posts.content, '') ILIKE '%' || ? || '%' OR COALESCE(posts.title, '') ILIKE '%' || ? || '%' OR COALESCE(sub_sources.name, '') ILIKE '%' || ? || '%'",
			*query, *query, *query)
	}

	q.Where(sql).
		Order("posts.content_generated_at desc").
		Limit(limit).
		Find(&postsToPublish)

	// This call will also update feed object with posts, no need to append
	if query == nil || len(*query) == 0 {
		if err := db.Model(feed).UpdateColumns(model.Feed{UpdatedAt: feed.UpdatedAt}).Association("Posts").Append(postsToPublish); err != nil {
			Logger.LogV2.Error(fmt.Sprintf("Failed to publish feeds %v", err))
		}
	} else {
		feed.Posts = append(feed.Posts, postsToPublish...)
	}
}

func getUserColumnSubscriptions(r *queryResolver, userID string) ([]*model.Column, error) {
	var user model.User
	queryResult := r.DB.Where("id = ?", userID).Preload("SubscribedColumns").First(&user)
	if queryResult.RowsAffected != 1 {
		return nil, errors.New("User not found")
	}
	return user.SubscribedColumns, nil
}

func sanitizeFeedRefreshInput(query *model.FeedRefreshInput, feed *model.Feed) error {
	if query.Cursor < 0 {
		return errors.New("query.Cursor should be >= 0")
	}

	if query.Limit <= 0 {
		return errors.New("query.Limit should be > 0")
	}

	// Check if requested cursors are out of sync from last feed update
	// If out of sync, default to query latest posts
	// Use unix() to avoid accuracy loss due to gqlgen serialization impacting matching
	if query.FeedUpdatedTime == nil || query.FeedUpdatedTime.Unix() != feed.UpdatedAt.Unix() {
		Logger.LogV2.Info(
			fmt.Sprintf("requested with outdated feed updated time, feed_id=%s query updated time=%v feed updated at=%v",
				feed.Id, query.FeedUpdatedTime, feed.UpdatedAt))
		query.Cursor = defaultFeedsQueryCursor
		query.Direction = defaultFeedsQueryDirection
	}

	// Cap query limit
	if query.Limit > feedRefreshLimit {
		query.Limit = feedRefreshLimit
	}

	return nil
}

func sanitizeColumnRefreshInput(query *model.ColumnRefreshInput, column *model.Column) error {
	if query.Cursor < 0 {
		return errors.New("query.Cursor should be >= 0")
	}

	if len(query.FeedIds) != len(query.FeedUpdatedTimes) {
		return errors.New("cursors and feedIds and feedUpdatedTimes should have same length")
	}

	if query.Limit <= 0 {
		return errors.New("query.Limit should be > 0")
	}

	// Check if requested cursors are out of sync from last feed update
	// If out of sync, default to query latest posts
	// Use unix() to avoid accuracy loss due to gqlgen serialization impacting matching
	if query.ColumnUpdatedTime == nil || query.ColumnUpdatedTime.Unix() != column.UpdatedAt.Unix() || len(query.FeedIds) != len(column.Feeds) {
		Logger.LogV2.Info(fmt.Sprintf(
			"requested with outdated column updated time, column_id=%s query updated time=%v column updated at=%v",
			column.Id, query.ColumnUpdatedTime, column.UpdatedAt))
		query.FeedIds = []string{}
		query.Cursor = defaultFeedsQueryCursor
		query.FeedUpdatedTimes = []*time.Time{}
		for _, f := range column.Feeds {
			query.FeedIds = append(query.FeedIds, f.Id)
			query.Direction = defaultFeedsQueryDirection
			query.FeedUpdatedTimes = append(query.FeedUpdatedTimes, &f.UpdatedAt)
		}
	}

	// Cap query limit
	if query.Limit > feedRefreshLimit {
		query.Limit = feedRefreshLimit
	}

	return nil
}

func isClearPostsNeededForFeedsUpsert(feed *model.Feed, input *model.UpsertFeedInput) (bool, error) {
	var subsourceIds []string
	for _, subsource := range feed.SubSources {
		subsourceIds = append(subsourceIds, subsource.Id)
	}
	dataExpressionMatched, err := utils.AreJSONsEqual(feed.FilterDataExpression.String(), input.FilterDataExpression)
	if err != nil {
		return false, err
	}

	if !dataExpressionMatched || !utils.StringSlicesContainSameElements(subsourceIds, input.SubSourceIds) {
		return true, nil
	}

	return false, nil
}

func UpsertSubsourceImpl(db *gorm.DB, input model.UpsertSubSourceInput) (*model.SubSource, error) {
	var subSource model.SubSource
	// TECHDEBT
	// TODO: change the mobile_notification logic to subscribers logic when we have other notification settings
	queryResult := db.Preload("Feeds").Preload("Feeds.Columns").Preload("Feeds.Columns.SubscribedChannels").
		Where("(name = ? OR (external_identifier = ? AND external_identifier IS NOT NULL AND external_identifier != '')) AND source_id = ?", input.Name, input.ExternalIdentifier, input.SourceID).
		First(&subSource)

	for i, feed := range subSource.Feeds {
		for j, column := range feed.Columns {
			var users []*model.User
			db.Model(&model.UserColumnSubscription{}).
				Joins("INNER JOIN users ON users.id = user_column_subscriptions.user_id").
				Where("user_column_subscriptions.column_id = ? AND mobile_notification = TRUE", column.Id).
				Select("Name", "Id").
				Find(&users)
			subSource.Feeds[i].Columns[j].Subscribers = users
		}
	}

	var customizedCrawlerParams *string
	if input.CustomizedCrawlerParams != nil {
		config, err := ConstructCustomizedCrawlerParams(*input.CustomizedCrawlerParams)
		if err != nil {
			return nil, err
		}
		bytes, err := prototext.Marshal(config)
		if err != nil {
			return nil, err
		}
		str := string(bytes)
		customizedCrawlerParams = &str
	}

	if queryResult.RowsAffected == 0 {
		var customizedCrawlerParams *string
		if input.CustomizedCrawlerParams != nil {
			config, err := ConstructCustomizedCrawlerParams(*input.CustomizedCrawlerParams)
			if err != nil {
				return nil, err
			}
			bytes, err := prototext.Marshal(config)
			if err != nil {
				return nil, err
			}
			str := string(bytes)
			customizedCrawlerParams = &str
		}

		// Create new SubSource
		subSource = model.SubSource{
			Id:                      uuid.New().String(),
			Name:                    input.Name,
			ExternalIdentifier:      input.ExternalIdentifier,
			SourceID:                input.SourceID,
			AvatarUrl:               input.AvatarURL,
			OriginUrl:               input.OriginURL,
			IsFromSharedPost:        input.IsFromSharedPost,
			CustomizedCrawlerParams: customizedCrawlerParams,
		}
		db.Create(&subSource)
		return &subSource, nil
	}
	// Update existing SubSource
	// Don't change subsource name
	// subSource.Name = input.Name
	// subSource.ExternalIdentifier = input.ExternalIdentifier
	if input.AvatarURL != "" {
		subSource.AvatarUrl = input.AvatarURL
	}
	if input.OriginURL != "" {
		subSource.OriginUrl = input.OriginURL
	}

	// Do not set it back to nil
	if customizedCrawlerParams != nil {
		subSource.CustomizedCrawlerParams = customizedCrawlerParams
	}
	if !input.IsFromSharedPost {
		// can only update IsFromSharedPost from true to false
		// meaning from hidden to display
		// to prevent an already needed subsource got shared, and become IsFromSharedPost = true
		subSource.IsFromSharedPost = false
	}
	db.Save(&subSource)

	return &subSource, nil
}

// For Customized SubSource
// Transform user provided form into CustomizedCrawlerParams in panoptic.proto
func ConstructCustomizedCrawlerParams(input model.CustomizedCrawlerParams) (*protocol.CustomizedCrawlerParams, error) {
	customizedCrawlerParams := &protocol.CustomizedCrawlerParams{
		CrawlUrl:                   input.CrawlURL,
		BaseSelector:               input.BaseSelector,
		TitleRelativeSelector:      input.TitleRelativeSelector,
		ContentRelativeSelector:    input.ContentRelativeSelector,
		ExternalIdRelativeSelector: input.ExternalIDRelativeSelector,
		TimeRelativeSelector:       input.TimeRelativeSelector,
		ImageRelativeSelector:      input.ImageRelativeSelector,
		SubsourceRelativeSelector:  input.SubsourceRelativeSelector,
		OriginUrlRelativeSelector:  input.OriginURLRelativeSelector,
		OriginUrlIsRelativePath:    input.OriginURLIsRelativePath,
	}
	return customizedCrawlerParams, nil
}

type ByContentGeneratedAtDesc []*model.Post

func (a ByContentGeneratedAtDesc) Len() int           { return len(a) }
func (a ByContentGeneratedAtDesc) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByContentGeneratedAtDesc) Less(i, j int) bool { return a[i].ContentGeneratedAt.After(a[j].ContentGeneratedAt) }


/**
 * Search posts in DB with 2 phase, search posts then posts shared them.
 * @param r
 * @param input
 * @return posts with keyword in title or content, the size does not strictly respect the limit, it could
 * be larger than limit but no more than 2 * limit
 */
func searchPostsInDB(r *queryResolver, input *model.SearchPostsInput) ([]*model.Post, error) {
	var start time.Time = time.Now()
	userId := input.UserID
	searchPostsRefreshInput := input.SearchPostsRefreshInput
	limit := searchPostsRefreshInput.Limit
	direction := searchPostsRefreshInput.Direction
	cursor := searchPostsRefreshInput.Cursor // oldest one, smaller cursor
	otherEndCursor := searchPostsRefreshInput.OtherEndCursor // latest one, larger cursor
	query := *searchPostsRefreshInput.Query
	unread := searchPostsRefreshInput.Filter.Unread
	Logger.LogV2.Info(fmt.Sprintf("searchPostsInDB quert=%s, limit=%d, cursor=%d, otherCursor=%d dir=%s\n",
		query, limit, cursor, *otherEndCursor, direction))

	queryValue := "%" + query + "%"
	cursorCondition := "posts.cursor > ?"
	cursorConditionValue := *otherEndCursor
	if cursor == 0 || *otherEndCursor == 0 {
		cursorConditionValue = 0
	} else {
		if direction == model.FeedRefreshDirectionOld {
			cursorCondition = "posts.cursor < ?"
			cursorConditionValue = cursor
		} else {
			// keep the default
		}
	}

	// First, find all posts that match the keyword in their title or content
	var matchingPosts []*model.Post
	matchQuery := r.DB.Model(&model.Post{}).
		Preload("SubSource").
		Joins(`LEFT JOIN sub_sources ON posts.sub_source_id = sub_sources.id`).
		Joins("LEFT JOIN user_post_reads ON user_post_reads.post_id = posts.id AND user_post_reads.user_id = ?", userId).
		Select("posts.*, CASE WHEN user_post_reads.post_id IS NOT NULL THEN true ELSE false END as is_read").
		Where(`(posts.title ILIKE ? OR posts.content ILIKE ?)`, queryValue, queryValue)

	if *unread {
		// matchQuery.Where("user_post_reads IS NOT NULL")
		matchQuery.
        Where(`NOT EXISTS (
            SELECT 1 FROM user_post_reads 
            WHERE user_post_reads.post_id = posts.id 
            AND user_post_reads.user_id = ?
        )`, userId)
	}
	if cursor != 0 && *otherEndCursor != 0 {
		matchQuery.Where(cursorCondition, cursorConditionValue)
	}
		
	matchQuery.Order("content_generated_at DESC").
		Limit(limit).
		Find(&matchingPosts)
	var matchTime time.Time = time.Now()
	Logger.LogV2.Info(fmt.Sprintf("query for matched posts took %d ms\n", matchTime.Sub(start).Milliseconds()))
	
	
	// Check for an error after the first query
	if matchQuery.Error != nil {
		return nil, matchQuery.Error
	}

	// early return if no match
	if len(matchingPosts) == 0 {
		return matchingPosts, nil
	}

	// Use a map to track the post IDs and remove duplicates
	postMap := make(map[string]*model.Post)

	// matchingPosts should be sorted and not empty, get the content_generated_at of the last post
	var lastMatchPostTime time.Time  = matchingPosts[len(matchingPosts)-1].ContentGeneratedAt

	// Add the matching posts to the map
	for _, post := range matchingPosts {
		postMap[post.Id] = post
	}

	// Extract the IDs of the matching posts for the next query
	var postIDs []string
	for id := range postMap {
		postIDs = append(postIDs, id)
	}

	// Now find all posts that are shares from the posts matched in the first query
	var sharedPosts []*model.Post
	shareQuery := r.DB.
		Preload("SharedFromPost").
		Preload("SharedFromPost.SubSource").
		Joins(`LEFT JOIN sub_sources ON posts.sub_source_id = sub_sources.id`).
		Joins(`LEFT JOIN posts "shared_from_post" ON shared_from_post.id = posts.shared_from_post_id`).
		Joins("LEFT JOIN user_post_reads ON user_post_reads.post_id = posts.id AND user_post_reads.user_id = ?", userId).
		Select("posts.*, CASE WHEN user_post_reads.post_id IS NOT NULL THEN true ELSE false END as is_read").
		Where(`posts.shared_from_post_id IN (?)`, postIDs)
	if *unread {
		shareQuery.
        Where(`NOT EXISTS (
            SELECT 1 FROM user_post_reads 
            WHERE user_post_reads.post_id = posts.id 
            AND user_post_reads.user_id = ?
        )`, userId)
	}
	if cursor != 0 && *otherEndCursor != 0 {
		shareQuery.Where(cursorCondition, cursorConditionValue)
	}
	shareQuery.Order("content_generated_at DESC").Limit(limit).Find(&sharedPosts)
	var sharePostQueryTime time.Time = time.Now()
	Logger.LogV2.Info(fmt.Sprintf("query for share posts took %d ms\n", sharePostQueryTime.Sub(matchTime).Milliseconds()))

	// Check for an error after the second query
	if shareQuery.Error != nil {
		return nil, shareQuery.Error
	}

	// Add the shared posts to the map, avoiding duplicates
	for _, post := range sharedPosts {
		if _, exists := postMap[post.Id]; !exists {
			postMap[post.Id] = post
		}
	}

	// Convert the map back to a slice and sort them by content generated time
	var posts []*model.Post
	for _, post := range postMap {
		if post.ContentGeneratedAt.Equal(lastMatchPostTime) || post.ContentGeneratedAt.After(lastMatchPostTime) {
			posts = append(posts, post)
		}
	}

	// Sort posts by content generated time in descending order
	// Assuming your Post model has a field named 'ContentGeneratedAt' which is of type time.Time
	sort.Sort(ByContentGeneratedAtDesc(posts))

	Logger.LogV2.Info(fmt.Sprintf("searchPostsInDB took %d ms in total for query %s\n",
		time.Now().Sub(start).Milliseconds(), queryValue))
	return posts, nil
}
