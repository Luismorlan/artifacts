package collector_instances

import (
	"fmt"

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

type CustomizedSourceCrawler struct {
	Sink       sink.CollectedDataSink
	ImageStore file_store.CollectedFileStore
}

func (j CustomizedSourceCrawler) UpdateTitle(workingContext *working_context.CrawlerWorkingContext) error {
	workingContext.Result.Post.Title = collector.CustomizedCrawlerExtractPlainText(workingContext.Task.TaskParams.
		GetCustomizedSourceCrawlerTaskParams().TitleRelativeSelector, workingContext.Element, "")
	return nil
}

func (j CustomizedSourceCrawler) UpdateContent(workingContext *working_context.CrawlerWorkingContext) error {
	workingContext.Result.Post.Content = collector.CustomizedCrawlerExtractPlainText(workingContext.Task.TaskParams.
		GetCustomizedSourceCrawlerTaskParams().ContentRelativeSelector, workingContext.Element, "")
	return nil
}

func (j CustomizedSourceCrawler) UpdateExternalId(workingContext *working_context.CrawlerWorkingContext) error {
	workingContext.Result.Post.SubSource.ExternalId = collector.CustomizedCrawlerExtractPlainText(workingContext.Task.TaskParams.
		GetCustomizedSourceCrawlerTaskParams().ExternalIdRelativeSelector, workingContext.Element, "")
	return nil
}

func (j CustomizedSourceCrawler) UpdateGeneratedTime(workingContext *working_context.CrawlerWorkingContext) error {
	dateString := collector.CustomizedCrawlerExtractPlainText(workingContext.Task.TaskParams.
		GetCustomizedSourceCrawlerTaskParams().TimeRelativeSelector, workingContext.Element, "")
	t, err := dateparse.ParseLocal(dateString)
	if err != nil {
		workingContext.Result.Post.ContentGeneratedAt = timestamppb.Now()
	} else {
		workingContext.Result.Post.ContentGeneratedAt = timestamppb.New(t)
	}
	return nil
}

// Dedup id in customized crawler is fixed logic, user don't have UI to modify it
func (j CustomizedSourceCrawler) UpdateDedupId(workingContext *working_context.CrawlerWorkingContext) error {
	md5, err := utils.TextToMd5Hash(workingContext.Result.Post.Content)
	if err != nil {
		return err
	}
	workingContext.Result.Post.DeduplicateId = md5
	return nil
}

func (j CustomizedSourceCrawler) UpdateSubsource(workingContext *working_context.CrawlerWorkingContext) error {
	if workingContext.SubSource != nil {
		workingContext.Result.Post.SubSource.Name = workingContext.SubSource.Name
	} else {
		return fmt.Errorf("subsource is nil")
	}
	workingContext.Result.Post.SubSource.OriginUrl = workingContext.OriginUrl
	workingContext.Result.Post.SubSource.AvatarUrl = *workingContext.SubSource.AvatarUrl
	return nil
}

func (j CustomizedSourceCrawler) UpdateImageUrls(workingContext *working_context.CrawlerWorkingContext) error {
	imageUrls := collector.CustomizedCrawlerExtractMultiAttribute(workingContext.Task.TaskParams.
		GetCustomizedSourceCrawlerTaskParams().ImageRelativeSelector, workingContext.Element, "src")

	s3OrOriginalUrls, err := collector.UploadImagesToS3(j.ImageStore, imageUrls, nil)

	if err != nil {
		Logger.LogV2.Errorf("fail to get customized_source images, err:", err, "urls:", imageUrls)
	}
	workingContext.Result.Post.ImageUrls = s3OrOriginalUrls
	return nil
}

func (j CustomizedSourceCrawler) GetMessage(workingContext *working_context.CrawlerWorkingContext) error {
	collector.InitializeCrawlerResult(workingContext)

	updaters := []func(workingContext *working_context.CrawlerWorkingContext) error{
		j.UpdateTitle,
		j.UpdateContent,
		j.UpdateExternalId,
		j.UpdateGeneratedTime,
		j.UpdateSubsource,
		j.UpdateImageUrls,
		j.UpdateDedupId,
	}
	for _, updater := range updaters {
		err := updater(workingContext)
		if err != nil {
			return err
		}
	}

	return nil
}

func (j CustomizedSourceCrawler) GetBaseSelector(task *protocol.PanopticTask) (string, error) {
	return task.TaskParams.GetCustomizedSourceCrawlerTaskParams().BaseSelector, nil
}

func (j CustomizedSourceCrawler) GetCrawlUrl(task *protocol.PanopticTask) (string, error) {
	return task.TaskParams.GetCustomizedSourceCrawlerTaskParams().CrawlUrl, nil
}

func (j CustomizedSourceCrawler) CollectAndPublish(task *protocol.PanopticTask) {
	metadata := task.TaskMetadata

	startUrl, err := j.GetCrawlUrl(task)
	if err != nil {
		collector.MarkAndLogCrawlError(task, err, "")
		return
	}

	baseSelector, err := j.GetBaseSelector(task)
	if err != nil {
		collector.MarkAndLogCrawlError(task, err, "")
		return
	}

	if len(task.TaskParams.SubSources) != 1 {
		collector.MarkAndLogCrawlError(task, err, "Source level customized crawler should have exact 1 subsource ")
		return
	}

	c := colly.NewCollector()
	// each crawled card(news) will go to this
	// for each page loaded, there are multiple calls into this func
	c.OnHTML(baseSelector, func(elem *colly.HTMLElement) {
		var err error

		workingContext := &working_context.CrawlerWorkingContext{
			SharedContext: working_context.SharedContext{Task: task, IntentionallySkipped: false}, SubSource: task.TaskParams.SubSources[0], Element: elem, OriginUrl: startUrl}
		if err = j.GetMessage(workingContext); err != nil {
			metadata.TotalMessageFailed++
			collector.LogHtmlParsingError(task, elem, err)
			return
		}
		sink.PushResultToSinkAndRecordInTaskMetadata(j.Sink, workingContext)
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
			SubSource:       task.TaskParams.SubSources[0],
			RssResponseItem: elem,
		}
		collector.InitializeRssCollectorResult(workingContext)

		node := elem.DOM.(*xmlquery.Node)
		workingContext.Result.Post.Title = collector.CustomizedCrawlerExtractPlainTextXml(workingContext.Task.TaskParams.
			GetCustomizedSourceCrawlerTaskParams().TitleRelativeSelector, node, "")
		workingContext.Result.Post.Content = collector.CustomizedCrawlerExtractPlainTextXml(workingContext.Task.TaskParams.
			GetCustomizedSourceCrawlerTaskParams().ContentRelativeSelector, node, "")
		workingContext.Result.Post.SubSource.ExternalId = collector.CustomizedCrawlerExtractPlainTextXml(workingContext.Task.TaskParams.
			GetCustomizedSourceCrawlerTaskParams().ExternalIdRelativeSelector, node, "")
		dateString := collector.CustomizedCrawlerExtractPlainTextXml(workingContext.Task.TaskParams.
			GetCustomizedSourceCrawlerTaskParams().TimeRelativeSelector, node, "")
		t, err := dateparse.ParseLocal(dateString)
		if err != nil {
			workingContext.Result.Post.ContentGeneratedAt = timestamppb.Now()
		} else {
			workingContext.Result.Post.ContentGeneratedAt = timestamppb.New(t)
		}
		workingContext.Result.Post.SubSource.Name = workingContext.SubSource.Name

		workingContext.Result.Post.SubSource.AvatarUrl = *workingContext.SubSource.AvatarUrl
		workingContext.Result.Post.SubSource.Name = workingContext.SubSource.Name
		workingContext.Result.Post.OriginUrl = collector.CustomizedCrawlerExtractPlainTextXml(workingContext.Task.TaskParams.
			GetCustomizedSourceCrawlerTaskParams().OriginUrlRelativeSelector, node, "")

		md5 := ""
		if workingContext.Result.Post.OriginUrl != "" {
			md5, _ = utils.TextToMd5Hash(workingContext.Result.Post.OriginUrl)
		} else {
			md5, _ = utils.TextToMd5Hash(workingContext.Result.Post.Title + workingContext.Result.Post.Content)
		}
		workingContext.Result.Post.DeduplicateId = md5

		sink.PushResultToSinkAndRecordInTaskMetadata(j.Sink, workingContext)
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
}
