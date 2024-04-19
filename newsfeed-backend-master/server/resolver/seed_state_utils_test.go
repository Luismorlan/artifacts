package resolver

import (
	"testing"

	"github.com/rnr-capital/newsfeed-backend/model"
	"github.com/rnr-capital/newsfeed-backend/utils"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestConstructSeedStateFromUser(t *testing.T) {
	ss := constructSeedStateFromUser(&model.User{
		Id:        "user_id",
		Name:      "user_name",
		AvatarUrl: "user_avatar_url",
		SubscribedColumns: []*model.Column{
			{Id: "column_id_1", Name: "column_name_1"},
			{Id: "column_id_2", Name: "column_name_2"},
		},
	})

	assert.Equal(t, ss, &model.SeedState{
		UserSeedState: &model.UserSeedState{
			ID:        "user_id",
			Name:      "user_name",
			AvatarURL: "user_avatar_url",
		},
		// Order dependent comparison.
		ColumnSeedState: []*model.ColumnSeedState{
			{ID: "column_id_1", Name: "column_name_1"},
			{ID: "column_id_2", Name: "column_name_2"},
		},
	})
}

func TestUpdateUserSeedState(t *testing.T) {

	db, _ := utils.CreateTempDB(t)

	assert.Nil(t, db.Create(&model.User{
		Id:                "id",
		Name:              "name",
		AvatarUrl:         "avatar_url",
		SubscribedColumns: []*model.Column{},
	}).Error)

	db.Transaction(func(tx *gorm.DB) error {
		if err := updateUserSeedState(tx, &model.SeedStateInput{
			UserSeedState: &model.UserSeedStateInput{
				ID:        "id",
				Name:      "new_name",
				AvatarURL: "new_avatar_url",
			},
		}); err != nil {
			// return error will rollback
			return err
		}

		return nil
	})

	var user model.User
	assert.Nil(t, db.Model(&model.User{}).Select("id", "name", "avatar_url").Where("id=?", "id").First(&user).Error)
	assert.Equal(t, &model.User{
		Id:        "id",
		Name:      "new_name",
		AvatarUrl: "new_avatar_url",
	}, &user)
}

func TestUpdateUserSeedState_UserNotFound(t *testing.T) {
	db, _ := utils.CreateTempDB(t)

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := updateUserSeedState(tx, &model.SeedStateInput{
			UserSeedState: &model.UserSeedStateInput{
				ID:        "id",
				Name:      "new_name",
				AvatarURL: "new_avatar_url",
			},
		}); err != nil {
			// return error will rollback
			return err
		}

		return nil
	})
	assert.NotNil(t, err)
}

func TestUpdateFeedState(t *testing.T) {
	db, _ := utils.CreateTempDB(t)

	assert.Nil(t, db.Select("id", "name").Create(&[]model.Column{
		{
			Id:   "id_1",
			Name: "name_1",
		},
		{
			Id:   "id_2",
			Name: "name_2",
		},
	}).Error)

	db.Transaction(func(tx *gorm.DB) error {
		if err := updateColumnSeedState(tx, &model.SeedStateInput{
			ColumnSeedState: []*model.ColumnSeedStateInput{
				{ID: "id_1", Name: "new_name_1"},
				{ID: "id_2", Name: "new_name_2"},
			},
		}); err != nil {
			// return error will rollback
			return err
		}
		return nil
	})

	var columns []model.Column
	db.Select("id", "name").
		Find(&columns, []string{"id_1", "id_2"}).
		Order("id")

	assert.Equal(t, 2, len(columns))
	assert.Equal(t, []model.Column{
		{Id: "id_1", Name: "new_name_1"},
		{Id: "id_2", Name: "new_name_2"}},
		columns)
}

func TestUpdateUserFeedSubscription_ChangeOrder(t *testing.T) {
	db, _ := utils.CreateTempDB(t)

	assert.Nil(t, db.Create(&model.User{
		Id:                "id",
		Name:              "name",
		AvatarUrl:         "avatar_url",
		SubscribedColumns: []*model.Column{},
	}).Error)

	assert.Nil(t, db.Select("id", "name").Create(&[]model.Column{
		{
			Id:   "id_1",
			Name: "name_1",
		},
		{
			Id:   "id_2",
			Name: "name_2",
		},
	}).Error)

	assert.Nil(t, db.Create(&[]model.UserColumnSubscription{
		{UserID: "id", ColumnID: "id_1", OrderInPanel: 0},
		{UserID: "id", ColumnID: "id_2", OrderInPanel: 1},
	}).Error)

	db.Transaction(func(tx *gorm.DB) error {
		if err := updateUserColumnSubscription(tx, &model.SeedStateInput{
			UserSeedState: &model.UserSeedStateInput{
				ID: "id",
			},
			ColumnSeedState: []*model.ColumnSeedStateInput{
				{ID: "id_2", Name: "name_2"},
				{ID: "id_1", Name: "name_1"},
			},
		}); err != nil {
			// return error will rollback
			return err
		}
		return nil
	})

	var userToColumns []model.UserColumnSubscription
	assert.Nil(t, db.Model(&model.UserColumnSubscription{}).
		Select("user_id, column_id", "order_in_panel").
		Where("user_id = ?", "id").
		Order("order_in_panel").
		Find(&userToColumns).Error)
	assert.Equal(t, []model.UserColumnSubscription{
		{UserID: "id", ColumnID: "id_2", OrderInPanel: 0},
		{UserID: "id", ColumnID: "id_1", OrderInPanel: 1},
	}, userToColumns)
}

func TestGetSeedStateById(t *testing.T) {
	db, _ := utils.CreateTempDB(t)

	assert.Nil(t, db.Create(&model.User{
		Id:                "id",
		Name:              "name",
		AvatarUrl:         "avatar_url",
		SubscribedColumns: []*model.Column{},
	}).Error)

	assert.Nil(t, db.Select("id", "name").Create(&[]model.Column{
		{
			Id:   "id_1",
			Name: "name_1",
		},
		{
			Id:   "id_3",
			Name: "name_3",
		},

		{
			Id:   "id_2",
			Name: "name_2",
		},
	}).Error)

	assert.Nil(t, db.Create(&[]model.UserColumnSubscription{
		{UserID: "id", ColumnID: "id_1", OrderInPanel: 1},
		{UserID: "id", ColumnID: "id_3", OrderInPanel: 2},
		{UserID: "id", ColumnID: "id_2", OrderInPanel: 0},
	}).Error)

	ss, err := getSeedStateById(db, "id")

	assert.Nil(t, err)
	assert.EqualValues(t, model.UserSeedState{
		ID:        "id",
		Name:      "name",
		AvatarURL: "avatar_url",
	}, *ss.UserSeedState)

	assert.EqualValues(t, []*model.ColumnSeedState{
		{ID: "id_2", Name: "name_2"},
		{ID: "id_1", Name: "name_1"},
		{ID: "id_3", Name: "name_3"},
	}, ss.ColumnSeedState)
}
