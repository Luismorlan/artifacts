package collector_instances

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gocolly/colly"
	"github.com/rnr-capital/newsfeed-backend/collector"
	"github.com/rnr-capital/newsfeed-backend/collector/file_store"
	"github.com/rnr-capital/newsfeed-backend/collector/sink"
	"github.com/rnr-capital/newsfeed-backend/collector/working_context"
	"github.com/rnr-capital/newsfeed-backend/protocol"
	"github.com/rnr-capital/newsfeed-backend/utils"
	Logger "github.com/rnr-capital/newsfeed-backend/utils/log"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	ClsNewsStartUrl = "https://www.cls.cn/telegraph"
)

type ClsNewsCrawler struct {
	Sink       sink.CollectedDataSink
	ImageStore file_store.CollectedFileStore
}

func (cls ClsNewsCrawler) UpdateImageUrls(workingContext *working_context.CrawlerWorkingContext) error {
	// prepare imageUrls
	imageUrls := []string{}
	selection := workingContext.Element.DOM.Find(".telegraph-images-box > img")
	for i := 0; i < selection.Length(); i++ {
		img := selection.Eq(i)
		imageUrl := img.AttrOr("src", "")
		parts := strings.Split(imageUrl, "?")
		imageUrl = parts[0]
		if len(imageUrl) == 0 {
			Logger.LogV2.Errorf("image DOM exist but src not found at index ", i, " of selection")
			continue
		}
		imageUrls = append(imageUrls, imageUrl)
	}

	// fetch and upload imageUrks to S3
	s3OrOriginalUrls, err := collector.UploadImagesToS3(cls.ImageStore, imageUrls, nil)
	if err != nil {
		Logger.LogV2.Errorf("fail to get cls_news image, err:", err)
	}
	workingContext.Result.Post.ImageUrls = s3OrOriginalUrls
	return nil
}

func (j ClsNewsCrawler) UpdateFileUrls(workingContext *working_context.CrawlerWorkingContext) error {
	return errors.New("UpdateFileUrls not implemented, should not be called")
}

func (j ClsNewsCrawler) UpdateNewsType(workingContext *working_context.CrawlerWorkingContext) error {
	s := workingContext.Element.DOM.Find(".telegraph-content-box")
	selection := s.Find(":nth-child(2)")
	if len(selection.Nodes) > 0 && selection.HasClass("c-de0422") {
		workingContext.NewsType = protocol.PanopticSubSource_KEYNEWS
	} else {
		workingContext.NewsType = protocol.PanopticSubSource_FLASHNEWS
	}

	if !collector.IsRequestedNewsType(workingContext.Task.TaskParams.SubSources, workingContext.NewsType) {
		workingContext.IntentionallySkipped = true
	}

	return nil
}

func (j ClsNewsCrawler) UpdateContent(workingContext *working_context.CrawlerWorkingContext) error {
	html, _ := workingContext.Element.DOM.Find(".telegraph-content-box span:not(.telegraph-time-box)").Html()
	workingContext.Result.Post.Content = html
	title_selection := workingContext.Element.DOM.Find(".telegraph-content-box span:not(.telegraph-time-box) > strong")
	if title_selection.Length() == 0 {
		title_selection = workingContext.Element.DOM.Find(".telegraph-content-box span:not(.telegraph-time-box) > div > strong")
	}
	fmt.Println(title_selection)
	fmt.Println("here", workingContext.Result)
	workingContext.Result.Post.Content = strings.ReplaceAll(workingContext.Result.Post.Content, "<div>", "")
	workingContext.Result.Post.Content = strings.ReplaceAll(workingContext.Result.Post.Content, "</div>", "")
	workingContext.Result.Post.Content = strings.ReplaceAll(workingContext.Result.Post.Content, "<br/>", "\n")
	workingContext.Result.Post.Content = strings.ReplaceAll(workingContext.Result.Post.Content, "<strong>", "")
	workingContext.Result.Post.Content = strings.ReplaceAll(workingContext.Result.Post.Content, "</strong>", "")
	workingContext.Result.Post.Content = strings.ReplaceAll(workingContext.Result.Post.Content, "<nil>", "")
	workingContext.Result.Post.Content = strings.TrimPrefix(workingContext.Result.Post.Content, "\n")
	workingContext.Result.Post.Content = strings.TrimSuffix(workingContext.Result.Post.Content, "\n")
	if title_selection.Length() > 0 {
		replacer := strings.NewReplacer("【", "", "】", "")
		workingContext.Result.Post.Title = replacer.Replace(title_selection.Text())
		workingContext.Result.Post.Content = strings.ReplaceAll(workingContext.Result.Post.Content, title_selection.Text(), "")
	}
	return nil
}

func (j ClsNewsCrawler) UpdateURL(workingContext *working_context.CrawlerWorkingContext) error {
	url, _ := workingContext.Element.DOM.Find("a:contains('评论')").Attr("href")
	workingContext.Result.Post.OriginUrl = fmt.Sprintf("https://www.cls.cn%s", url)
	return nil
}

func (j ClsNewsCrawler) UpdateTags(workingContext *working_context.CrawlerWorkingContext) error {
	workingContext.Result.Post.Tags = []string{}
	selection := workingContext.Element.DOM.Find(".label-item")
	for i := 0; i < selection.Length(); i++ {
		tag := selection.Eq(i)
		workingContext.Result.Post.Tags = append(workingContext.Result.Post.Tags, tag.Text())
	}
	return nil
}

func (j ClsNewsCrawler) UpdateGeneratedTime(workingContext *working_context.CrawlerWorkingContext) error {
	workingContext.Result.Post.ContentGeneratedAt = timestamppb.Now()
	return nil
}

func (j ClsNewsCrawler) UpdateDedupId(workingContext *working_context.CrawlerWorkingContext) error {
	md5, err := utils.TextToMd5Hash(workingContext.Result.Post.Content)
	if err != nil {
		return err
	}
	workingContext.Result.Post.DeduplicateId = md5
	return nil
}

func (j ClsNewsCrawler) UpdateSubsourceName(workingContext *working_context.CrawlerWorkingContext) error {
	workingContext.Result.Post.SubSource.Name = collector.SubsourceTypeToName(workingContext.NewsType)
	return nil
}

func (j ClsNewsCrawler) UpdateSubsourceOriginUrl(workingContext *working_context.CrawlerWorkingContext) error {
	workingContext.Result.Post.SubSource.OriginUrl = ClsNewsStartUrl
	return nil
}

func (j ClsNewsCrawler) UpdateVipSkip(workingContext *working_context.CrawlerWorkingContext) error {
	selection := workingContext.Element.DOM.Find(".telegraph-vip-box")
	if selection.Length() > 0 {
		workingContext.IntentionallySkipped = true
	}
	return nil
}

func (j ClsNewsCrawler) GetMessage(workingContext *working_context.CrawlerWorkingContext) error {
	collector.InitializeCrawlerResult(workingContext)

	updaters := []func(workingContext *working_context.CrawlerWorkingContext) error{
		j.UpdateContent,
		j.UpdateImageUrls,
		j.UpdateURL,
		j.UpdateTags,
		j.UpdateDedupId,
		j.UpdateNewsType,
		j.UpdateGeneratedTime,
		j.UpdateSubsourceName,
		j.UpdateSubsourceOriginUrl,
		j.UpdateVipSkip,
	}
	for _, updater := range updaters {
		err := updater(workingContext)
		if err != nil {
			return err
		}
	}

	return nil
}

func (j ClsNewsCrawler) GetQueryPath() string {
	return `.telegraph-list`
}

func (j ClsNewsCrawler) GetStartUri() string {
	return ClsNewsStartUrl
}

// todo: mock http response and test end to end Collect()
func (j ClsNewsCrawler) CollectAndPublish(task *protocol.PanopticTask) {
	metadata := task.TaskMetadata

	c := colly.NewCollector()
	// each crawled card(news) will go to this
	// for each page loaded, there are multiple calls into this func
	c.OnHTML(j.GetQueryPath(), func(elem *colly.HTMLElement) {
		var err error
		workingContext := &working_context.CrawlerWorkingContext{
			SharedContext: working_context.SharedContext{Task: task, IntentionallySkipped: false}, Element: elem, OriginUrl: j.GetStartUri()}
		if err = j.GetMessage(workingContext); err != nil {
			metadata.TotalMessageFailed++
			collector.LogHtmlParsingError(task, elem, err)
			return
		}
		sink.PushResultToSinkAndRecordInTaskMetadata(j.Sink, workingContext)
	})

	// Set error handler
	c.OnError(func(r *colly.Response, err error) {
		task.TaskMetadata.ResultState = protocol.TaskMetadata_STATE_FAILURE
		Logger.LogV2.Errorf("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err, " path ", j.GetQueryPath())
	})

	c.OnScraped(func(_ *colly.Response) {
		// Set Fail/Success in task meta based on number of message succeeded
		collector.SetErrorBasedOnCounts(task, j.GetStartUri(), fmt.Sprintf(" path: %s", j.GetQueryPath()))
	})

	c.OnRequest(func(r *colly.Request) {
		for _, kv := range task.TaskParams.HeaderParams {
			r.Headers.Set(kv.Key, kv.Value)
		}
	})

	c.Visit(j.GetStartUri())
}
