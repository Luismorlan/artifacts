package resolver

import (
	"errors"

	"github.com/rnr-capital/newsfeed-backend/model"
	"github.com/rnr-capital/newsfeed-backend/utils"
	"gorm.io/gorm"
)

// constructSeedStateFromUser constructs SeedState with model.User with
// pre-populated SubscribedFeeds.
func constructSeedStateFromUser(user *model.User) *model.SeedState {
	res := &model.SeedState{
		UserSeedState: &model.UserSeedState{
			ID:        user.Id,
			Name:      user.Name,
			AvatarURL: user.AvatarUrl,
		},
		ColumnSeedState: columnToSeedState(user.SubscribedColumns),
	}

	return res
}

// feedToSeedState converts from Feed to FeedSeedState.
func feedToSeedState(feeds []*model.Feed) []*model.FeedSeedState {
	res := []*model.FeedSeedState{}

	for _, feed := range feeds {
		res = append(res, &model.FeedSeedState{
			ID:   feed.Id,
			Name: feed.Name,
		})
	}

	return res
}

// feedToSeedState converts from Feed to FeedSeedState.
func columnToSeedState(columns []*model.Column) []*model.ColumnSeedState {
	res := []*model.ColumnSeedState{}

	for _, column := range columns {
		res = append(res, &model.ColumnSeedState{
			ID:   column.Id,
			Name: column.Name,
		})
	}

	return res
}

func updateUserSeedState(tx *gorm.DB, input *model.SeedStateInput) error {
	var user model.User
	res := tx.Model(&model.User{}).Where("id=?", input.UserSeedState.ID).First(&user)
	if res.RowsAffected != 1 {
		return errors.New("user not found")
	}

	user.AvatarUrl = input.UserSeedState.AvatarURL
	user.Name = input.UserSeedState.Name

	if err := tx.Save(&user).Error; err != nil {
		return err
	}

	return nil
}

func updateColumnSeedState(tx *gorm.DB, input *model.SeedStateInput) error {
	for _, columnSeedStateInput := range input.ColumnSeedState {
		// Handler error in a soft way. If the feed doesn't exist, continue.
		var tmp model.Column
		res := tx.Model(&model.Column{}).Where("id = ?", columnSeedStateInput.ID).First(&tmp)
		if res.RowsAffected != 1 {
			continue
		}
		res = tx.Model(&model.Column{}).Where("id = ?", columnSeedStateInput.ID).
			Updates(model.Feed{Name: columnSeedStateInput.Name})
		if res.Error != nil {
			// Return error will rollback
			return res.Error
		}
	}

	return nil
}

// updateUserFeedSubscription will do 2 things:
// 1. remove/add unnecessary user feed subscription.
// 2. reorder Feed subscriptions.
func updateUserColumnSubscription(tx *gorm.DB, input *model.SeedStateInput) error {
	columnIdToPos := make(map[string]int)
	for idx, columnSeedStateInput := range input.ColumnSeedState {
		columnIdToPos[columnSeedStateInput.ID] = idx
	}

	var userToColumns []model.UserColumnSubscription
	if err := tx.Model(&model.UserColumnSubscription{}).
		Where("user_id = ?", input.UserSeedState.ID).
		Find(&userToColumns).Error; err != nil {
		return err
	}

	for _, userToColumn := range userToColumns {
		pos, ok := columnIdToPos[userToColumn.ColumnID]
		if !ok {
			continue
		}

		// Otherwise we should just update the position. We use map to update the
		// field order_in_panel due to zero-like value will be ignored during
		// structural update. See https://gorm.io/docs/update.html for details.
		if err := tx.Model(&model.UserColumnSubscription{}).
			Where("user_id = ? AND column_id = ?", userToColumn.UserID, userToColumn.ColumnID).
			Updates(map[string]interface{}{
				"order_in_panel": pos,
			}).Error; err != nil {
			return err
		}
	}

	// return nil will commit the whole transaction
	return nil
}

// create a syncUp transaction callback that performs the core business logic
func syncUpTransaction(input *model.SeedStateInput) utils.GormTransaction {
	return func(tx *gorm.DB) error {
		if err := updateUserSeedState(tx, input); err != nil {
			// return error will rollback
			return err
		}

		if err := updateColumnSeedState(tx, input); err != nil {
			return err
		}

		if err := updateUserColumnSubscription(tx, input); err != nil {
			return err
		}

		// return nil will commit the whole transaction
		return nil
	}
}

// getting the latest SeedState from the DB
func getSeedStateById(db *gorm.DB, userId string) (*model.SeedState, error) {
	var user model.User
	res := db.Model(&model.User{}).Where("id=?", userId).First(&user)
	if res.RowsAffected != 1 {
		return nil, errors.New("user not found or duplicate user")
	}

	var columns []model.Column
	db.Model(&model.UserColumnSubscription{}).
		Select("columns.id", "columns.name").
		Joins("INNER JOIN columns ON columns.id = user_column_subscriptions.column_id").
		Where("user_column_subscriptions.user_id = ?", userId).
		Order("order_in_panel").
		Find(&columns)

	for idx := range columns {
		user.SubscribedColumns = append(user.SubscribedColumns, &columns[idx])
	}

	ss := constructSeedStateFromUser(&user)

	return ss, nil
}
