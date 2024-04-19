package collector_builder

import (
	twitterscraper "github.com/n0madic/twitter-scraper"
	. "github.com/rnr-capital/newsfeed-backend/collector"
	"github.com/rnr-capital/newsfeed-backend/collector/file_store"
	. "github.com/rnr-capital/newsfeed-backend/collector/instances"
	"github.com/rnr-capital/newsfeed-backend/collector/sink"
)

type CollectorBuilder struct{}

func (CollectorBuilder) NewCaUsArticleCrawlerCollector(s sink.CollectedDataSink, imageStore file_store.CollectedFileStore) CrawlerCollector {
	return &CaUsArticleCrawler{Sink: s, ImageStore: imageStore}
}

// Crawler Collectors
func (CollectorBuilder) NewJin10Crawler(s sink.CollectedDataSink, imageStore file_store.CollectedFileStore) CrawlerCollector {
	return &Jin10Crawler{Sink: s, ImageStore: imageStore}
}

func (CollectorBuilder) NewZsxqApiCollector(s sink.CollectedDataSink, imageStore file_store.CollectedFileStore, fileStore file_store.CollectedFileStore) ApiCollector {
	return &ZsxqApiCollector{Sink: s, ImageStore: imageStore, FileStore: fileStore}
}

func (CollectorBuilder) NewWeiboApiCollector(s sink.CollectedDataSink, store file_store.CollectedFileStore) ApiCollector {
	return &WeiboApiCollector{Sink: s, ImageStore: store}
}

func (CollectorBuilder) NewWallstreetNewsApiCollector(s sink.CollectedDataSink) ApiCollector {
	return &WallstreetApiCollector{Sink: s}
}

func (CollectorBuilder) NewKuailansiApiCollector(s sink.CollectedDataSink) DataCollector {
	return &KuailansiApiCrawler{Sink: s}
}

func (CollectorBuilder) NewJinseApiCollector(s sink.CollectedDataSink) DataCollector {
	return &JinseApiCrawler{Sink: s}
}

func (CollectorBuilder) NewWeixinRssCollector(s sink.CollectedDataSink, imageStore file_store.CollectedFileStore) DataCollector {
	return &WeixinArticleRssCollector{Sink: s, ImageStore: imageStore}
}

func (CollectorBuilder) NewWisburgCrawler(s sink.CollectedDataSink, imageStore file_store.CollectedFileStore) DataCollector {
	return &WisburgCrawler{Sink: s, ImageStore: imageStore}
}

func (CollectorBuilder) NewKe36ApiCollector(s sink.CollectedDataSink) DataCollector {
	return &Kr36ApiCollector{Sink: s}
}

func (CollectorBuilder) NewXueqiuCollector(s sink.CollectedDataSink, imageStore file_store.CollectedFileStore) DataCollector {
	return &XueqiuApiCollector{Sink: s, ImageStore: imageStore}
}

func (CollectorBuilder) NewWallstreetNewsArticleCollector(s sink.CollectedDataSink, imageStore file_store.CollectedFileStore) DataCollector {
	return &WallstreetArticleCollector{Sink: s, ImageStore: imageStore}
}

func (CollectorBuilder) NewCaUsNewsCrawlerCollector(s sink.CollectedDataSink, imageStore file_store.CollectedFileStore) DataCollector {
	return &CaUsNewsCrawler{Sink: s, ImageStore: imageStore}
}

func (CollectorBuilder) NewCaixinCrawler(s sink.CollectedDataSink) DataCollector {
	return &CaixinCollector{Sink: s}
}

func (CollectorBuilder) NewGelonghuiCrawler(s sink.CollectedDataSink) DataCollector {
	return &GelonghuiCrawler{Sink: s}
}

func (CollectorBuilder) NewClsNewsCrawlerCollector(s sink.CollectedDataSink, imageStore file_store.CollectedFileStore) DataCollector {
	return &ClsNewsCrawler{Sink: s, ImageStore: imageStore}
}

func (CollectorBuilder) NewCustomizedSourceCrawlerCollector(s sink.CollectedDataSink, imageStore file_store.CollectedFileStore) DataCollector {
	return &CustomizedSourceCrawler{Sink: s, ImageStore: imageStore}
}

func (CollectorBuilder) NewCustomizedSubSourceCollector(s sink.CollectedDataSink, imageStore file_store.CollectedFileStore) DataCollector {
	return &CustomizedSubSourceCrawler{Sink: s, ImageStore: imageStore}
}

func (CollectorBuilder) NewTwitterCollector(s sink.CollectedDataSink, imageStore file_store.CollectedFileStore) DataCollector {
	return &TwitterApiCrawler{Sink: s, Scraper: twitterscraper.New(), ImageStore: imageStore}
}

func (CollectorBuilder) NewWublock123Collector(s sink.CollectedDataSink, imageStore file_store.CollectedFileStore) DataCollector {
	return &Wublock123Collector{Sink: s, ImageStore: imageStore}
}
