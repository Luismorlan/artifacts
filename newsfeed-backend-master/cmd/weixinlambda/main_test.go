package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseWeixinArticleFromJson(t *testing.T) {
	expected := []WeixinArticle{
		{
			Source: "爱范儿",
			Title:  "你以后喝的星巴克，将会有点不一样",
			Url:    "https://mp.weixin.qq.com/s/EvYwEC_xGSPohZRPOLCRVQ",
		},
		{
			Source: "爱范儿",
			Title:  "欧洲车企，需要中国电池",
			Url:    "https://mp.weixin.qq.com/s/AWKDKBuOhLFsWJu8FJD-PQ",
		},
	}

	articles, err := ParseWeixinArticleFromJson(`{"robotId": "651d064cc88da92ae75e4aa9",
	"articles":[
	{
		"source":"爱范儿",
		"title":"你以后喝的星巴克，将会有点不一样",
		"url":"https://mp.weixin.qq.com/s/EvYwEC_xGSPohZRPOLCRVQ"
	},
	{
		"source":"爱范儿",
		"title":"欧洲车企，需要中国电池",
		"url":"https://mp.weixin.qq.com/s/AWKDKBuOhLFsWJu8FJD-PQ"
	}]}`)
	assert.Nil(t, err)
	assert.Equal(t, articles, expected)
}
