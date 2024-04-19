package model

import (
	"time"

	"gorm.io/gorm"
)

/*

User is a data model for a newsfeed user

Id: primary key, use to identify a user
CreatedAt: time when entity is created
DeletedAt: time when entity is deleted

Name: name of a user, can be changed, don't need to be unique
AvatarUrl: User's icon URL.
SubscribedColumns: columns that this user subscribed, "many-to-many" relation
Posts: posts that this user saved or read, "many-to-many" relation
SharedPosts: posts that this user shared, "many-to-many" relation

*/

type User struct {
	Id                string `gorm:"primaryKey"`
	CreatedAt         time.Time
	DeletedAt         gorm.DeletedAt
	Name              string
	AvatarUrl         string
	Email             string
	SubscribedColumns []*Column `json:"subscribed_columns" gorm:"many2many:user_column_subscriptions;constraint:OnDelete:CASCADE;"`
	PostsRead         []*Post   `json:"posts_read" gorm:"many2many:user_post_reads;"`
	FeedsFavorite     []*Feed   `json:"feeds_favorite" gorm:"many2many:user_feed_favorites;"`
}

func (User) IsUserSeedStateInterface() {}

func (u User) GetID() string        { return u.Id }
func (u User) GetName() string      { return u.Name }
func (u User) GetAvatarURL() string { return u.AvatarUrl }

var _ UserSeedStateInterface = User{}
