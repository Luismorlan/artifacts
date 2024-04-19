package collector_instances

import (
	"fmt"
	"strings"

	"github.com/antchfx/xmlquery"
	"github.com/araddon/dateparse"
	"github.com/gocolly/colly"
	"golang.org/x/net/html"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rnr-capital/newsfeed-backend/collector"
	"github.com/rnr-capital/newsfeed-backend/collector/file_store"
	"github.com/rnr-capital/newsfeed-backend/collector/sink"
	"github.com/rnr-capital/newsfeed-backend/collector/working_context"
	"github.com/rnr-capital/newsfeed-backend/protocol"
	"github.com/rnr-capital/newsfeed-backend/utils"
	Logger "github.com/rnr-capital/newsfeed-backend/utils/log"
)

type CustomizedSubSourceCrawler struct {
	Sink       sink.CollectedDataSink
	ImageStore file_store.CollectedFileStore
}

func (crawler CustomizedSubSourceCrawler) UpdateTitle(workingContext *working_context.CrawlerWorkingContext) error {
	workingContext.Result.Post.Title = collector.CustomizedCrawlerExtractPlainText(workingContext.SubSource.CustomizedCrawlerParamsForSubSource.TitleRelativeSelector, workingContext.Element, "")
	return nil
}

func (crawler CustomizedSubSourceCrawler) UpdateContent(workingContext *working_context.CrawlerWorkingContext) error {
	workingContext.Result.Post.Content = collector.CustomizedCrawlerExtractPlainText(workingContext.SubSource.CustomizedCrawlerParamsForSubSource.ContentRelativeSelector, workingContext.Element, "")
	return nil
}

func (crawler CustomizedSubSourceCrawler) UpdateExternalId(workingContext *working_context.CrawlerWorkingContext) error {
	workingContext.Result.Post.SubSource.ExternalId = collector.CustomizedCrawlerExtractPlainText(workingContext.SubSource.CustomizedCrawlerParamsForSubSource.ExternalIdRelativeSelector, workingContext.Element, "")
	return nil
}

func (crawler CustomizedSubSourceCrawler) UpdateGeneratedTime(workingContext *working_context.CrawlerWorkingContext) error {
	dateString := collector.CustomizedCrawlerExtractPlainText(workingContext.SubSource.CustomizedCrawlerParamsForSubSource.TimeRelativeSelector, workingContext.Element, "")
	t, err := dateparse.ParseLocal(dateString)
	if err != nil {
		workingContext.Result.Post.ContentGeneratedAt = timestamppb.Now()
	} else {
		workingContext.Result.Post.ContentGeneratedAt = timestamppb.New(t)
	}
	return nil
}

// Dedup id in customized crawler is fixed logic, user don't have UI to modify it
func (crawler CustomizedSubSourceCrawler) UpdateDedupId(workingContext *working_context.CrawlerWorkingContext) error {
	md5, err := utils.TextToMd5Hash(workingContext.Result.Post.Content)
	if err != nil {
		return err
	}
	workingContext.Result.Post.DeduplicateId = md5
	return nil
}

// For subsource customized crawler, we don't use subsource jquery selector to get subsource name, we use the one specified in task params instead
func (crawler CustomizedSubSourceCrawler) UpdateSubsource(workingContext *working_context.CrawlerWorkingContext) error {
	if workingContext.SubSource != nil {
		workingContext.Result.Post.SubSource.Name = workingContext.SubSource.Name
	} else {
		return fmt.Errorf("subsource is nil")
	}

	workingContext.Result.Post.SubSource.AvatarUrl = *workingContext.SubSource.AvatarUrl
	return nil
}

func (crawler CustomizedSubSourceCrawler) UpdateImageUrls(workingContext *working_context.CrawlerWorkingContext) error {
	imageUrls := collector.CustomizedCrawlerExtractMultiAttribute(workingContext.
		SubSource.CustomizedCrawlerParamsForSubSource.ImageRelativeSelector, workingContext.Element, "src")

	s3OrOriginalUrls, err := collector.UploadImagesToS3(crawler.ImageStore, imageUrls, nil)

	if err != nil {
		Logger.LogV2.Errorf("fail to get customized_sub_source images, err:", err, "urls:", imageUrls)
	}
	workingContext.Result.Post.ImageUrls = s3OrOriginalUrls
	return nil
}

func (crawler CustomizedSubSourceCrawler) UpdateOriginUrl(workingContext *working_context.CrawlerWorkingContext) error {
	params := workingContext.SubSource.CustomizedCrawlerParamsForSubSource
	workingContext.Result.Post.OriginUrl = collector.CustomizedCrawlerExtractAttribute(params.OriginUrlRelativeSelector, workingContext.Element, params.CrawlUrl, "href")
	if params.OriginUrlIsRelativePath != nil && *params.OriginUrlIsRelativePath {
		base := params.CrawlUrl
		path := workingContext.Result.Post.OriginUrl
		Logger.LogV2.Info(fmt.Sprintf("crawed subsource: base: %s, path: %s", base, path))
		workingContext.Result.Post.OriginUrl = collector.ConcateUrlBaseAndRelativePath(base, path)
	}
	return nil
}

func (crawler CustomizedSubSourceCrawler) GetMessage(workingContext *working_context.CrawlerWorkingContext) error {
	collector.InitializeCrawlerResult(workingContext)

	updaters := []func(workingContext *working_context.CrawlerWorkingContext) error{
		crawler.UpdateTitle,
		crawler.UpdateContent,
		crawler.UpdateExternalId,
		crawler.UpdateGeneratedTime,
		crawler.UpdateSubsource,
		crawler.UpdateImageUrls,
		crawler.UpdateDedupId,
		crawler.UpdateOriginUrl,
	}
	for _, updater := range updaters {
		err := updater(workingContext)
		if err != nil {
			return err
		}
	}

	return nil
}

func (crawler CustomizedSubSourceCrawler) GetBaseSelector(subsource *protocol.PanopticSubSource) (string, error) {
	return subsource.CustomizedCrawlerParamsForSubSource.BaseSelector, nil
}

func (crawler CustomizedSubSourceCrawler) GetCrawlUrl(subsource *protocol.PanopticSubSource) (string, error) {
	return subsource.CustomizedCrawlerParamsForSubSource.CrawlUrl, nil
}

func (crawler CustomizedSubSourceCrawler) CollectOneSubsource(task *protocol.PanopticTask, subsource *protocol.PanopticSubSource) error {
	metadata := task.TaskMetadata

	startUrl, err := crawler.GetCrawlUrl(subsource)
	if err != nil {
		collector.MarkAndLogCrawlError(task, err, "")
		return err
	}

	baseSelector, err := crawler.GetBaseSelector(subsource)
	if err != nil {
		collector.MarkAndLogCrawlError(task, err, "")
		return err
	}

	c := colly.NewCollector()
	// each crawled card(news) will go to this
	// for each page loaded, there are multiple calls into this func
	c.OnHTML(baseSelector, func(elem *colly.HTMLElement) {
		var err error

		workingContext := &working_context.CrawlerWorkingContext{
			SharedContext: working_context.SharedContext{Task: task, IntentionallySkipped: false}, Element: elem, OriginUrl: startUrl,
			SubSource: subsource}
		if err = crawler.GetMessage(workingContext); err != nil {
			metadata.TotalMessageFailed++
			collector.LogHtmlParsingError(task, elem, err)
			return
		}
		sink.PushResultToSinkAndRecordInTaskMetadata(crawler.Sink, workingContext)
	})

	c.OnXML(baseSelector, func(elem *colly.XMLElement) {
		Logger.LogV2.Info("RSS source detected")
		switch elem.DOM.(type) {
		// html will be processed twice as xml, skip in this case
		case *html.Node:
			return
		}

		workingContext := &working_context.RssCollectorWorkingContext{
			SharedContext: working_context.SharedContext{
				Task: task, IntentionallySkipped: false},
			RssUrl:          startUrl,
			SubSource:       subsource,
			RssResponseItem: elem,
		}
		collector.InitializeRssCollectorResult(workingContext)

		node := elem.DOM.(*xmlquery.Node)
		workingContext.Result.Post.Title = collector.CustomizedCrawlerExtractPlainTextXml(workingContext.SubSource.CustomizedCrawlerParamsForSubSource.TitleRelativeSelector, node, "")
		workingContext.Result.Post.Content = collector.CustomizedCrawlerExtractPlainTextXml(workingContext.SubSource.CustomizedCrawlerParamsForSubSource.ContentRelativeSelector, node, "")
		workingContext.Result.Post.SubSource.ExternalId = collector.CustomizedCrawlerExtractPlainTextXml(workingContext.SubSource.CustomizedCrawlerParamsForSubSource.ExternalIdRelativeSelector, node, "")
		dateString := collector.CustomizedCrawlerExtractPlainTextXml(workingContext.SubSource.CustomizedCrawlerParamsForSubSource.TimeRelativeSelector, node, "")
		t, err := dateparse.ParseLocal(dateString)
		if err != nil {
			workingContext.Result.Post.ContentGeneratedAt = timestamppb.Now()
		} else {
			workingContext.Result.Post.ContentGeneratedAt = timestamppb.New(t)
		}
		workingContext.Result.Post.SubSource.Name = workingContext.SubSource.Name

		workingContext.Result.Post.SubSource.AvatarUrl = *workingContext.SubSource.AvatarUrl
		workingContext.Result.Post.SubSource.Name = workingContext.SubSource.Name

		// Wisburg's rss url is flaky, sometimes with www while somestimes not, which causes duplicate posts
		workingContext.Result.Post.OriginUrl = strings.Replace(collector.CustomizedCrawlerExtractPlainTextXml(workingContext.SubSource.CustomizedCrawlerParamsForSubSource.OriginUrlRelativeSelector, node, ""), "https://www.wisburg.com", "https://wisburg.com", 1)

		md5 := ""
		if workingContext.Result.Post.OriginUrl != "" {
			md5, _ = utils.TextToMd5Hash(workingContext.Result.Post.OriginUrl)
		} else {
			md5, _ = utils.TextToMd5Hash(workingContext.Result.Post.Title + workingContext.Result.Post.Content)
		}
		workingContext.Result.Post.DeduplicateId = md5
		Logger.LogV2.Info(fmt.Sprintf("crawled customized rss from url %s with %s", startUrl, workingContext.Result.Post.Content))

		sink.PushResultToSinkAndRecordInTaskMetadata(crawler.Sink, workingContext)
	})

	// Set error handler
	c.OnError(func(r *colly.Response, err error) {
		task.TaskMetadata.ResultState = protocol.TaskMetadata_STATE_FAILURE
		Logger.LogV2.Errorf("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err, " path ", baseSelector)
	})

	c.OnScraped(func(_ *colly.Response) {
		// Set Fail/Success in task meta based on number of message succeeded
		collector.SetErrorBasedOnCounts(task, startUrl, fmt.Sprintf(" path: %s", baseSelector))
	})

	c.OnRequest(func(r *colly.Request) {
		if len(task.TaskParams.HeaderParams) == 0 {
			// to avoid http 418
			task.TaskParams.HeaderParams = collector.GetDefautlCrawlerHeader()
		}
		for _, kv := range task.TaskParams.HeaderParams {
			r.Headers.Set(kv.Key, kv.Value)
		}
	})

	c.Visit(startUrl)

	return nil
}

func (crawler CustomizedSubSourceCrawler) CollectAndPublish(task *protocol.PanopticTask) {
	collector.ParallelSubsourceApiCollect(task, crawler)
	collector.SetErrorBasedOnCounts(task, "customized subsource crawler")
}
