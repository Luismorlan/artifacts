package model

import (
	"time"

	"gorm.io/gorm"
)

/*

ChannelColumnSubscription is a "many-to-many" relation of channel's subscription to a feed

ChannelId: channel id
ColumnId: column id
CreatedAt: time when relation is created

*/

type ChannelColumnSubscription struct {
	ChannelID string `gorm:"primaryKey"`
	ColumnID  string `gorm:"primaryKey"`
	CreatedAt time.Time
}

func (ChannelColumnSubscription) BeforeCreate(db *gorm.DB) error {
	return nil
}
