package model

import (
	"gorm.io/gorm"
)

/*

UserFeedFavorite is a "many-to-many" relation of user save a post

UserID: user id
PostID: post id
CreatedAt: time when relation is created
DeletedAt: time when relation is deleted

*/

type UserFeedFavorite struct {
	UserID   string `gorm:"primaryKey"`
	FeedID   string `gorm:"primaryKey"`
	Favorite bool   `gorm:"default:FALSE"`
}

func (UserFeedFavorite) BeforeCreate(db *gorm.DB) error {
	return nil
}
