package main

import (
	"context"
	"time"

	"github.com/rnr-capital/newsfeed-backend/bot"
	"github.com/rnr-capital/newsfeed-backend/model"
)

func main() {
	users := []*model.User{
		{
			Id: "2ef1afbc-6ce3-4493-9a6b-e03e421d5066",
		},
		{
			Id: "24f075ad-7f0c-4347-a8ef-e34b1e0204dd",
		},
		{
			Id: "userId-3",
		},
		{
			Id: "userId-4",
		},
		{
			Id: "userId-5",
		},
		{
			Id: "userId-6",
		},
	}
	bot.TimeBoundedNotifyPost(context.Background(), model.Post{
		Id:          "40a29c76-d483-4043-84a7-35030f1cf9ba",
		Title:       "财联社1月28日互动平台精选",
		Content:     "今日大盘波动较大，沪指收跌0.97%，深成指跌0.53%，创业板指涨0.07%。旅游、教育、猪肉、三胎等板块涨幅居前。互动平台上，威海广泰表示，公司目前订单饱满，生产情况良好；爱旭股份表示，近期公司大尺寸电池订单充足，相关产线处于满产状态；玉马遮阳表示，公司产品的市场需求仍持续旺盛，在手订单3-5个月左右；中天科技表示，得益于电子铜箔市场需求增长，公司进一步投建产线；可孚医疗表示，公司新冠抗原家用自测检测试剂正在申请欧盟CE认证和美国FDA认证；铜冠铜箔表示，铜价上涨将导致铜箔价格相应增加，对公司营业收入的增加产生积极作用。",
		SubSourceID: "0e036627-5e7b-4a46-b46d-375502d0d69c",
		SubSource: model.SubSource{
			Name: "快讯",
			Id:   "0e036627-5e7b-4a46-b46d-375502d0d69c",
		},
		ImageUrls: []string{"https://d20uffqoe1h0vv.cloudfront.net/3f7fdb3b9673b3ab6a8d1f8c10adf4b0.png"},
		Tag:       "互动平台精选",
	}, []*model.Column{
		{
			Id:          "FeedId-1",
			CreatedAt:   time.Date(2022, 1, 25, 12, 30, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2022, 1, 25, 12, 30, 0, 0, time.UTC),
			CreatorID:   "CreatorID-1",
			Subscribers: users[0:3],
		},
	})
}
