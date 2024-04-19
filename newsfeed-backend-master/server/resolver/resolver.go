package resolver

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/rnr-capital/newsfeed-backend/utils"
	"gorm.io/gorm"
)

const (
	DefaultSubSourceName = "default"
)

// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	DB               *gorm.DB
	RedisStatusStore *utils.RedisStatusStore
	SignalChans      *SignalChannels
}

func GetGinContextFromContext(ctx context.Context) (*gin.Context, error) {
	ginContext := ctx.Value("GinContextKey")
	if ginContext == nil {
		err := fmt.Errorf("count not retrieve gin.Context")
		return nil, err
	}
	gc, ok := ginContext.(*gin.Context)
	if !ok {
		err := fmt.Errorf("gin.Context has wrong type")
		return nil, err
	}
	return gc, nil
}
