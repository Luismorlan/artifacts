package resolver

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
// Code generated by github.com/99designs/gqlgen version v0.17.40

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/rnr-capital/newsfeed-backend/collector"
	"github.com/rnr-capital/newsfeed-backend/model"
	"github.com/rnr-capital/newsfeed-backend/server/graph/generated"
	Logger "github.com/rnr-capital/newsfeed-backend/utils/log"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CreateUser is the resolver for the createUser field.
func (r *mutationResolver) CreateUser(ctx context.Context, input model.NewUserInput) (*model.User, error) {
	var user model.User
	res := r.DB.Model(&model.User{}).Where("id = ?", input.ID).First(&user)
	if res.RowsAffected == 0 {
		// if the user doesn't exist, create the user.
		t := model.User{
			Id:   input.ID,
			Name: input.Name,
			// TODO(chenweilunster): For now we set only default user avatar, later
			// we'll allow user to customize their avatar in the frontend.
			AvatarUrl:         "https://robohash.org/54a9068a8750731226a284514c01b0bb?set=set4&bgset=&size=400x400",
			CreatedAt:         time.Now(),
			SubscribedColumns: []*model.Column{},
		}
		r.DB.Create(&t)
		return &t, nil
	}

	// otherwise
	return &user, nil
}

// UpsertFeed is the resolver for the upsertFeed field.
func (r *mutationResolver) UpsertFeed(ctx context.Context, input model.UpsertFeedInput) (*model.Feed, error) {
	// Upsert a feed
	// return feed with updated posts
	var (
		feed           model.Feed
		updatedFeed    model.Feed
		column         model.Column
		needClearPosts = true
	)

	// get creator user
	userID := input.UserID

	if input.FeedID != nil {
		// adding a favorite feed to a new column
		if input.AddToColumn {
			queryResult := r.DB.Where("id = ?", *input.FeedID).Preload("Columns").Preload("SubSources").First(&feed)
			if queryResult.RowsAffected != 1 {
				return nil, errors.New("invalid feed id")
			}
			if input.ColumnID == nil {
				column = model.Column{
					Id:        uuid.New().String(),
					CreatorID: userID,
					// TODO(Boning) we might want column name here to avoid weird behaviour
					Name:  "new column",
					Feeds: []*model.Feed{&feed},
				}
			} else {
				queryRes := r.DB.Where("id = ?", *input.ColumnID).Find(&column)
				if queryRes.RowsAffected != 1 {
					return nil, fmt.Errorf("provided column doesn't exist: %s", *input.ColumnID)
				}
			}

			feed.Columns = append(feed.Columns, &column)
		} else {
			// If it is update:
			// 1. read from DB
			queryResult := r.DB.Where("id = ?", *input.FeedID).Preload("SubSources").First(&feed)
			if queryResult.RowsAffected != 1 {
				return nil, errors.New("invalid feed id")
			}

			if feed.CreatorID != input.UserID && feed.Visibility == model.VisibilityPrivate {
				return nil, errors.New("only feed owner can make change to private feed")
			}

			if input.ColumnID != nil {
				queryRes := r.DB.Where("id = ?", *input.ColumnID).Find(&column)
				if queryRes.RowsAffected != 1 {
					return nil, fmt.Errorf("provided column doesn't exist: %s", *input.ColumnID)
				}
			}
			// 2. check if dropping posts is needed
			var err error
			needClearPosts, err = isClearPostsNeededForFeedsUpsert(&feed, &input)
			if err != nil {
				return nil, err
			}

			// Update feed object
			feed.Name = input.Name
			feed.FilterDataExpression = datatypes.JSON(input.FilterDataExpression)
		}
	} else {
		// If it is insert, create feed object
		feed = model.Feed{
			Id:                   uuid.New().String(),
			Name:                 input.Name,
			CreatorID:            userID,
			FilterDataExpression: datatypes.JSON(input.FilterDataExpression),
		}
		// If it doesn't have columnId, create a column for it
		if input.ColumnID == nil {
			column = model.Column{
				Id:        uuid.New().String(),
				CreatorID: userID,
				// TODO(Boning) we might want column name here to avoid weird behaviour
				Name:  "new column",
				Feeds: []*model.Feed{&feed},
			}
			feed.Columns = []*model.Column{&column}
			r.DB.Save(column)
		} else {
			queryRes := r.DB.Where("id = ?", *input.ColumnID).Find(&column)
			if queryRes.RowsAffected != 1 {
				return nil, fmt.Errorf("provided column doesn't exist: %s", *input.ColumnID)
			}
		}
	}

	// One caveat on gorm: if we don't specify a createdAt
	// gorm will automatically update its created time after Save is called
	// even though DB is not udpated (this is a hell of debugging)

	// Upsert DB
	// err := r.DB.Transaction(func(tx *gorm.DB) error {
	// Update all columns, except primary keys and subscribers to new value, on conflict
	// fmt.Println("2")
	queryResult := r.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: false,
		DoUpdates: clause.AssignmentColumns([]string{"name", "updated_at", "creator_id", "filter_data_expression", "visibility"}),
	}).Create(&feed)
	// fmt.Println("3")

	if queryResult.RowsAffected != 1 {
		return nil, fmt.Errorf("can't upsert %s", queryResult.Error)
	}

	// Update subsources
	var subSources []*model.SubSource
	r.DB.Where("id IN ?", input.SubSourceIds).Find(&subSources)

	r.DB.Model(&feed).Association("SubSources").Replace(&subSources)

	// Update column-feed
	if column.Id != "" {
		if e := r.DB.Model(&column).Association("Feeds").Append(&feed); e != nil {
			return nil, e
		}
	}
	// If user upsert the feed's visibility to be PRIVATE, delete all
	// subscription other than the user himself.

	r.DB.Preload("Columns").Preload("SubSources").Preload("Creator").First(&updatedFeed, "id = ?", feed.Id)

	columnIds := []string{}
	for _, c := range updatedFeed.Columns {
		columnIds = append(columnIds, c.Id)
	}
	r.DB.Model(&model.Column{}).Where("id IN ?", columnIds).Update("updated_at", updatedFeed.UpdatedAt)

	// If no data expression or subsources changed, skip, otherwise clear the feed's posts
	if !needClearPosts {
		// get posts
		Logger.LogV2.Info("update feed metadata without clear published posts")
		return &updatedFeed, nil
	}

	// Clear the feed's posts
	Logger.LogV2.Info("changed feed clear all posts published")
	r.DB.Where("feed_id = ?", updatedFeed.Id).Delete(&model.PostFeedPublish{})
	updatedFeed.Posts = []*model.Post{}

	return &updatedFeed, nil
}

// UpsertColumn is the resolver for the upsertColumn field.
func (r *mutationResolver) UpsertColumn(ctx context.Context, input model.UpsertColumnInput) (*model.Column, error) {
	if input.ColumnID == nil {
		// input.ColumnId shoule not be nil as feeds need to be created which creates column as well
		return nil, errors.New("columnId should not be nil")
	}
	var (
		column model.Column
		user   model.User
	)
	upsertColumnStartTime := time.Now()
	queryResult := r.DB.Where("id = ?", input.UserID).First(&user)
	if queryResult.RowsAffected != 1 {
		return nil, errors.New("invalid user id")
	}

	queryResult = r.DB.Where("id = ?", *input.ColumnID).Preload("Feeds").First(&column)
	if queryResult.RowsAffected != 1 {
		return nil, errors.New("invalid column id")
	}

	// Allow user to update its own column and public column
	if column.CreatorID != user.Id && column.Visibility == model.VisibilityPrivate {
		return nil, errors.New("column is private and not owned by this user")
	}

	column.Name = input.Name
	column.Creator = user
	column.Visibility = input.Visibility

	queryResult = r.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: false,
		DoUpdates: clause.AssignmentColumns([]string{"name", "updated_at", "creator_id", "visibility"}),
	}).Create(&column)

	if queryResult.RowsAffected != 1 {
		return nil, fmt.Errorf("can't upsert %s", queryResult.Error)
	}

	feedIds := []string{}
	for _, f := range column.Feeds {
		feedIds = append(feedIds, f.Id)
	}
	sort.Strings(feedIds)

	inputFeedIds := []string{}
	inputFeedIds = append(inputFeedIds, input.FeedIds...)
	sort.Strings(inputFeedIds)

	if len(feedIds) != 0 && !Equal(feedIds, inputFeedIds) {
		var feeds []*model.Feed

		r.DB.Where("id IN ?", input.FeedIds).Find(&feeds)
		for _, feed := range feeds {
			feed.Visibility = column.Visibility
			r.DB.Model(&feed).Update("visibility", column.Visibility)
		}
		r.DB.Model(&column).Association("Feeds").Replace(&feeds)
	}

	if column.Visibility == model.VisibilityPrivate {
		if err := r.DB.Model(&model.UserColumnSubscription{}).
			Where("user_id != ? AND column_id = ?", column.Creator.Id, column.Id).
			Delete(model.UserColumnSubscription{}).Error; err != nil {
			return nil, err
		}
	}

	var updatedColumn model.Column
	r.DB.First(&updatedColumn, "id = ?", column.Id)
	elapsedQueryTime := time.Since(upsertColumnStartTime)
	fmt.Println("upsertColumns time:  ", elapsedQueryTime, "userId is: ", user.Id)
	Logger.LogV2.Info(fmt.Sprintf("upsertColumns time:  %v, userId is: %v", elapsedQueryTime, user.Id))

	return &updatedColumn, nil
}

// DeleteColumn is the resolver for the deleteColumn field.
func (r *mutationResolver) DeleteColumn(ctx context.Context, input model.DeleteColumnInput) (*model.Column, error) {
	userId := input.UserID
	columnId := input.ColumnID

	var column model.Column
	result := r.DB.First(&column, "id = ?", columnId)
	if result.RowsAffected != 1 {
		return nil, errors.New("no valid column found")
	}
	if result.Error != nil {
		return nil, result.Error
	}

	// Check ownership, if the deletion operation is not initiated from the Feed
	// owner, this is just unsubscribe. This is used in Feed sharing, where the
	// non-owner unsubscribes a feed.
	if column.CreatorID != userId {
		sub := model.UserColumnSubscription{}
		if err := r.DB.Model(&model.UserColumnSubscription{}).
			Where("user_id = ? AND column_id = ?", userId, columnId).
			Delete(&sub).Error; err != nil {
			return nil, err
		}
		return &column, nil
	}

	// Delete automatically cascade to join tables according to the schema.
	if err := r.DB.Delete(&column).Error; err != nil {
		return nil, err
	}

	go func() {
		r.SignalChans.PushSignalToUser(&model.Signal{
			SignalType: model.SignalTypeSeedState}, input.UserID)
	}()
	return &column, nil
}

// CreatePost is the resolver for the createPost field.
func (r *mutationResolver) CreatePost(ctx context.Context, input model.NewPostInput) (*model.Post, error) {
	var (
		subSource      model.SubSource
		sharedFromPost *model.Post
	)

	result := r.DB.Where("id = ?", input.SubSourceID).First(&subSource)
	if result.RowsAffected != 1 {
		return nil, errors.New("SubSource not found")
	}

	if input.SharedFromPostID != nil {
		var res model.Post
		result := r.DB.Where("id = ?", input.SharedFromPostID).First(&res)
		if result.RowsAffected != 1 {
			return nil, errors.New("SharedFromPost not found")
		}
		sharedFromPost = &res
	}

	post := model.Post{
		Id:                 uuid.New().String(),
		Title:              input.Title,
		Content:            input.Content,
		CreatedAt:          time.Now(),
		ContentGeneratedAt: time.Now(),
		SubSource:          subSource,
		SubSourceID:        input.SubSourceID,
		SharedFromPost:     sharedFromPost,
		ReadByUser:         []*model.User{},
		PublishedFeeds:     []*model.Feed{},
	}
	r.DB.Create(&post)

	for _, feedId := range input.FeedsIDPublishTo {
		err := r.DB.Transaction(func(tx *gorm.DB) error {
			var feed model.Feed
			result := tx.Where("id = ?", feedId).First(&feed)
			if result.RowsAffected != 1 {
				return errors.New("Feed not found")
			}

			if e := tx.Model(&post).Association("PublishedFeeds").Append(&feed); e != nil {
				return e
			}
			// return nil will commit the whole transaction
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return &post, nil
}

// Subscribe is the resolver for the subscribe field.
func (r *mutationResolver) Subscribe(ctx context.Context, input model.SubscribeInput) (*model.User, error) {
	userId := input.UserID
	columnId := input.ColumnID

	var user model.User
	var column model.Column

	result := r.DB.First(&user, "id = ?", userId)
	if result.RowsAffected != 1 {
		return nil, fmt.Errorf("no valid user found %s", userId)
	}
	if result.Error != nil {
		return nil, result.Error
	}

	result = r.DB.First(&column, "id = ?", columnId)
	if result.RowsAffected != 1 {
		return nil, fmt.Errorf("no valid column found %s", columnId)
	}
	if result.Error != nil {
		return nil, result.Error
	}

	count := r.DB.Model(&user).Association("SubscribedColumns").Count()

	if err := r.DB.Transaction(
		func(tx *gorm.DB) error {
			// The join table is ready after this associate, do not need to do for
			// feed model. Doing that will change the UpdateTime, which is not
			// expected and breaks when feed setting is updated
			if err := tx.Model(&user).
				Association("SubscribedColumns").
				Append(&column); err != nil {
				return err
			}
			// The newly subscribed feed must be at the last order, instead of using
			// order_in_panel == 0
			if err := tx.Model(&model.UserColumnSubscription{}).
				Where("user_id = ? AND column_id = ?", userId, columnId).
				Update("order_in_panel", count).Error; err != nil {
				return err
			}
			// return nil will commit the whole transaction
			return nil
		}); err != nil {
		return nil, err
	}

	return &user, nil
}

// CreateSource is the resolver for the createSource field.
func (r *mutationResolver) CreateSource(ctx context.Context, input model.NewSourceInput) (*model.Source, error) {
	// get creator user
	var user model.User
	queryResult := r.DB.Where("id = ?", input.UserID).First(&user)
	if queryResult.RowsAffected != 1 {
		return nil, errors.New("invalid user id")
	}
	newSourceId := uuid.New().String()
	source := model.Source{
		Id:        newSourceId,
		Name:      input.Name,
		Domain:    input.Domain,
		CreatedAt: time.Now(),
		Creator:   user,
	}

	err := r.DB.Transaction(func(tx *gorm.DB) error {
		tx.Create(&source)
		// Create default sub source, this subsource have no creator, no external id

		UpsertSubsourceImpl(tx, model.UpsertSubSourceInput{
			Name:               DefaultSubSourceName,
			ExternalIdentifier: "",
			SourceID:           source.Id,
		})
		return nil
	})

	return &source, err
}

// UpsertSubSource is the resolver for the upsertSubSource field.
func (r *mutationResolver) UpsertSubSource(ctx context.Context, input model.UpsertSubSourceInput) (*model.SubSource, error) {
	return UpsertSubsourceImpl(r.DB, input)
}

// AddWeiboSubSource is the resolver for the addWeiboSubSource field.
func (r *mutationResolver) AddWeiboSubSource(ctx context.Context, input model.AddWeiboSubSourceInput) (*model.SubSource, error) {
	return AddWeiboSubsourceImp(r.DB, ctx, input)
}

// AddSubSource is the resolver for the addSubSource field.
func (r *mutationResolver) AddSubSource(ctx context.Context, input model.AddSubSourceInput) (*model.SubSource, error) {
	return AddSubSourceImp(r.DB, ctx, input)
}

// DeleteSubSource is the resolver for the deleteSubSource field.
func (r *mutationResolver) DeleteSubSource(ctx context.Context, input *model.DeleteSubSourceInput) (*model.SubSource, error) {
	var subSource model.SubSource
	queryResult := r.DB.Where("id = ?", input.SubsourceID).First(&subSource)
	if queryResult.RowsAffected == 0 {
		return nil, fmt.Errorf("DeleteSubSource subsource with id %s not exist", input.SubsourceID)
	}
	subSource.IsFromSharedPost = true
	r.DB.Save(&subSource)
	return &subSource, nil
}

// SyncUp is the resolver for the syncUp field.
func (r *mutationResolver) SyncUp(ctx context.Context, input *model.SeedStateInput) (*model.SeedState, error) {
	if err := r.DB.Transaction(syncUpTransaction(input)); err != nil {
		return nil, err
	}

	ss, err := getSeedStateById(r.DB, input.UserSeedState.ID)
	if err != nil {
		return nil, err
	}

	// Asynchronously push to user's all other channels.
	// Feed deletion updates seed state.
	go func() {
		r.SignalChans.PushSignalToUser(&model.Signal{
			SignalType: model.SignalTypeSeedState}, input.UserSeedState.ID)
	}()

	return ss, err
}

// SetItemsReadStatus is the resolver for the setItemsReadStatus field.
func (r *mutationResolver) SetItemsReadStatus(ctx context.Context, input model.SetItemsReadStatusInput) (bool, error) {
	go func() { r.RedisStatusStore.SetItemsReadStatus(input.ItemNodeIds, input.UserID, input.Read) }()

	readPosts := []model.UserPostRead{}

	for i := 0; i < len(input.ItemNodeIds); i++ {
		readPosts = append(readPosts, model.UserPostRead{PostID: input.ItemNodeIds[i], UserID: input.UserID})
	}

	if input.Read {
		err := r.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(readPosts).Error
		if err != nil {
			return false, err
		}
	} else {
		err := r.DB.Clauses(clause.OnConflict{DoNothing: true}).Delete(readPosts).Error
		if err != nil {
			return false, err
		}
	}

	go func() {
		payload := ReadStatusPayload{
			read:        input.Read,
			itemNodeIds: input.ItemNodeIds,
			delimiter:   "__",
			itemType:    input.Type,
		}
		ser, _ := payload.Marshal()
		err := r.SignalChans.PushSignalToUser(&model.Signal{
			SignalType:    model.SignalTypeSetItemsReadStatus,
			SignalPayload: ser,
		}, input.UserID)
		if err != nil {
			Logger.LogV2.Error(fmt.Sprintf("failed to push read status signal to user %s: %v", input.UserID, err))
		}
	}()
	return true, nil
}

// SetFeedFavorite is the resolver for the setFeedFavorite field.
func (r *mutationResolver) SetFeedFavorite(ctx context.Context, input model.SetFeedFavoriteInput) (bool, error) {
	fmt.Println("input", input.FeedID, input.UserID, input.IsFavorite)
	fmt.Println(r.DB.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&model.UserFeedFavorite{UserID: input.UserID, FeedID: input.FeedID, Favorite: input.IsFavorite}))
	return true, nil
}

// SetNotificationSetting is the resolver for the setNotificationSetting field.
func (r *mutationResolver) SetNotificationSetting(ctx context.Context, input model.NotificationSettingInput) (bool, error) {
	r.DB.Model(&model.UserColumnSubscription{UserID: input.UserID, ColumnID: input.ColumnID}).Update("mobile_notification", input.Mobile).Update("web_notification", input.Web).Update("show_unread_indicator_on_icon", input.UnreadIndicatorOnIcon)
	return true, nil
}

// AllVisibleColumns is the resolver for the allVisibleColumns field.
func (r *queryResolver) AllVisibleColumns(ctx context.Context) ([]*model.Column, error) {
	var columns []*model.Column
	if err := r.DB.
		Preload("Creator").
		Preload("Feeds").
		Preload("Feeds.SubSources").
		Preload("Feeds.Creator").
		Where("visibility = 'GLOBAL'").
		Find(&columns).Error; err != nil {
		return nil, err
	}
	return columns, nil
}

// FavoriteFeeds is the resolver for the favoriteFeeds field.
func (r *queryResolver) FavoriteFeeds(ctx context.Context, input *model.UserIDInput) ([]*model.Feed, error) {
	var feeds []*model.Feed
	if err := r.DB.Model(model.Feed{}).
		Preload("SubSources").
		Preload("Creator").
		Joins("INNER JOIN user_feed_favorites ON user_feed_favorites.feed_id = feeds.id").
		Where("user_feed_favorites.user_id = ? AND user_feed_favorites.favorite is TRUE", input.UserID).
		Find(&feeds).Error; err != nil {
		return nil, err
	}
	return feeds, nil
}

// Post is the resolver for the post field.
func (r *queryResolver) Post(ctx context.Context, input *model.PostInput) (*model.Post, error) {
	var post *model.Post
	result := r.DB.
		Model(&model.Post{}).
		Preload("SharedFromPost.SubSource").
		Preload(clause.Associations).
		// Maintain a chronological order of reply thread.
		Preload("ReplyThread", func(db *gorm.DB) *gorm.DB {
			return db.Order("posts.created_at ASC")
		}).
		Preload("ReplyThread.SubSource").
		Preload("ReplyThread.SharedFromPost").
		Preload("ReplyThread.SharedFromPost.SubSource").
		Where("id=?", input.ID).First(&post)
	return post, result.Error
}

// Posts is the resolver for the posts field.
func (r *queryResolver) Posts(ctx context.Context, input *model.SearchPostsInput) ([]*model.Post, error) {
	return searchPostsInDB(r, input)
}

// Users is the resolver for the users field.
func (r *queryResolver) Users(ctx context.Context) ([]*model.User, error) {
	var users []*model.User
	result := r.DB.Preload(clause.Associations).Find(&users)
	return users, result.Error
}

// UserState is the resolver for the userState field.
func (r *queryResolver) UserState(ctx context.Context, input model.UserStateInput) (*model.UserState, error) {
	var user model.User
	res := r.DB.Model(&model.User{}).Where("id=?", input.UserID).First(&user)
	if res.RowsAffected != 1 {
		return nil, errors.New("user not found or duplicate user")
	}

	var columns []model.Column
	r.DB.Model(&model.UserColumnSubscription{}).
		Select("columns.id", "columns.name").
		Joins("INNER JOIN columns ON columns.id = user_column_subscriptions.column_id").
		Where("user_column_subscriptions.user_id = ?", input.UserID).
		Order("order_in_panel").
		Find(&columns)

	for idx := range columns {
		user.SubscribedColumns = append(user.SubscribedColumns, &columns[idx])
	}

	return &model.UserState{User: &user}, nil
}

// Feeds is the resolver for the feeds field.
func (r *queryResolver) Feeds(ctx context.Context, input *model.FeedsGetPostsInput) ([]*model.Feed, error) {
	feedRefreshInputs := input.FeedRefreshInputs
	return getRefreshFeedPosts(r, feedRefreshInputs, input.UserID)
}

// Columns is the resolver for the columns field.
func (r *queryResolver) Columns(ctx context.Context, input *model.ColumnsGetPostsInput) ([]*model.Column, error) {
	columnRefreshInputs := input.ColumnsRefreshInputs
	if len(columnRefreshInputs) == 0 {
		columns, err := getUserColumnSubscriptions(r, input.UserID)
		if err != nil {
			return nil, err
		}
		for _, column := range columns {
			columnRefreshInputs = append(columnRefreshInputs, &model.ColumnRefreshInput{
				ColumnID:  column.Id,
				Limit:     feedRefreshLimit,
				Direction: defaultFeedsQueryDirection,
			})
		}
	}

	res, err := getRefreshColumnPosts(r, columnRefreshInputs, input.UserID)
	if err != nil {
		return res, err
	}
	for _, c := range res {
		for _, f := range c.Feeds {
			Logger.LogV2.Info(fmt.Sprintf("setting column posts: %v", r.RedisStatusStore.SetColumnPosts(c.Id, f.Posts)))
		}
	}
	return res, err
}

// SubSources is the resolver for the subSources field.
func (r *queryResolver) SubSources(ctx context.Context, input *model.SubsourcesInput) ([]*model.SubSource, error) {
	var subSources []*model.SubSource
	filterCustomizedClause := ""
	if input.IsCustomized != nil {
		if *input.IsCustomized {
			filterCustomizedClause = "AND customized_crawler_params IS NOT NULL"
		} else {
			filterCustomizedClause = "AND customized_crawler_params IS NULL"
		}
	}
	result := r.DB.Preload(clause.Associations).Where("is_from_shared_post = ? "+filterCustomizedClause, input.IsFromSharedPost).Order("created_at").Find(&subSources)
	return subSources, result.Error
}

// Sources is the resolver for the sources field.
func (r *queryResolver) Sources(ctx context.Context, input *model.SourcesInput) ([]*model.Source, error) {
	var sources []*model.Source
	result := r.DB.Preload("SubSources", "is_from_shared_post = ?", input.SubSourceFromSharedPost).Find(&sources)
	return sources, result.Error
}

// TryCustomizedCrawler is the resolver for the tryCustomizedCrawler field.
func (r *queryResolver) TryCustomizedCrawler(ctx context.Context, input *model.CustomizedCrawlerParams) ([]*model.CustomizedCrawlerTestResponse, error) {
	return collector.TryCustomizedCrawler(input)
}

// Signal is the resolver for the signal field.
func (r *subscriptionResolver) Signal(ctx context.Context, userID string) (<-chan *model.Signal, error) {
	ch, chId := r.SignalChans.AddNewConnection(ctx, userID)
	// Initially, user by default will receive SeedState signal.
	r.SignalChans.PushSignalToSingleChannelForUser(
		&model.Signal{SignalType: model.SignalTypeSeedState},
		chId,
		userID)
	return ch, nil
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

// Subscription returns generated.SubscriptionResolver implementation.
func (r *Resolver) Subscription() generated.SubscriptionResolver { return &subscriptionResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
type subscriptionResolver struct{ *Resolver }

// !!! WARNING !!!
// The code below was going to be deleted when updating resolvers. It has been copied here so you have
// one last chance to move it out of harms way if you want. There are two reasons this happens:
//   - When renaming or deleting a resolver the old code will be put in here. You can safely delete
//     it when you're done.
//   - You have helper methods in this file. Move them out to keep these resolver files clean.
func Equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}