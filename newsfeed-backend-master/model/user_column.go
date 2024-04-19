package model

import (
	"time"

	"gorm.io/gorm"
)

/*

UserColumnSubscription is a "many-to-many" relation of user's subscription to a column

UserID: user id
ColumnID: Column id
CreatedAt: time when relation is created

*/

type UserColumnSubscription struct {
	UserID    string `gorm:"primaryKey"`
	ColumnID  string `gorm:"primaryKey"`
	CreatedAt time.Time

	// order of this column in user's panel, from left to right marked as 0,1,2,3...
	OrderInPanel int `gorm:"default:0"`

	// whether the user will receive mobile notification on this Column
	MobileNotification bool `gorm:"default:FALSE"`

	// whether the use will receive notification in browser on this Column
	WebNotification bool `gorm:"default:FALSE"`

	// Show unread indicator on Column's icon in side/bottom bar
	ShowUnreadIndicatorOnIcon bool `gorm:"default:TRUE"`
}

func (UserColumnSubscription) BeforeCreate(db *gorm.DB) error {
	return nil
}
