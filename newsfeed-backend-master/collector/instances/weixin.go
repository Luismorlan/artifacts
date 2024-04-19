package collector_instances

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rnr-capital/newsfeed-backend/collector"
	clients "github.com/rnr-capital/newsfeed-backend/collector/clients"
	"github.com/rnr-capital/newsfeed-backend/collector/file_store"
	"github.com/rnr-capital/newsfeed-backend/collector/sink"
	"github.com/rnr-capital/newsfeed-backend/collector/working_context"
	"github.com/rnr-capital/newsfeed-backend/protocol"
	"github.com/rnr-capital/newsfeed-backend/utils"
	Logger "github.com/rnr-capital/newsfeed-backend/utils/log"
)

func GetWeixinS3ImageStore(t *protocol.PanopticTask, isProd bool) (*file_store.S3FileStore, error) {
	bucketName := file_store.TestS3Bucket
	if isProd {
		bucketName = file_store.ProdS3ImageBucket
	}
	zsxqFileStore, err := file_store.NewS3FileStore(bucketName)
	if err != nil {
		return nil, err
	}
	zsxqFileStore.SetCustomizeFileExtFunc(GetWeixinImgExtMethod())
	return zsxqFileStore, nil
}

func GetWeixinImgExtMethod() file_store.CustomizeFileExtFuncType {
	return func(url string, fileName string) string {
		str := url
		str = strings.Replace(url, "wx_fmt%3D", "", -1)
		str = strings.Replace(str, "%22", "", -1)
		return "." + str
	}
}

const (
	WeixinArticleDateFormat = time.RFC1123Z
)

type WeixinArticleRssCollector struct {
	Sink       sink.CollectedDataSink
	ImageStore file_store.CollectedFileStore
}

func (w WeixinArticleRssCollector) UpdateFileUrls(workingContext *working_context.CrawlerWorkingContext) error {
	return errors.New("UpdateFileUrls not implemented, should not be called")
}

func (w WeixinArticleRssCollector) UpdateExternalPostId(workingContext *working_context.CrawlerWorkingContext) error {
	id := workingContext.Element.DOM.AttrOr("id", "")
	if len(id) == 0 {
		return errors.New("can't get external post id for the news")
	}
	workingContext.ExternalPostId = id
	return nil
}

func (w WeixinArticleRssCollector) UpdateDedupId(workingContext *working_context.RssCollectorWorkingContext) error {
	md5, err := utils.TextToMd5Hash(workingContext.Task.TaskParams.SourceId + workingContext.Result.Post.OriginUrl)
	if err != nil {
		return err
	}
	workingContext.Result.Post.DeduplicateId = md5
	return nil
}

func (w WeixinArticleRssCollector) UpdateAvatarUrl(post *protocol.CrawlerMessage_CrawledPost, res *gofeed.Feed) error {
	return nil
}

func (w WeixinArticleRssCollector) ConstructUrl(task *protocol.PanopticTask, subsource *protocol.PanopticSubSource) string {
	return "https://zapier.com/engine/rss/14662062/weixin"
}

func (w WeixinArticleRssCollector) UpdateResultFromArticle(
	article *gofeed.Item,
	res *gofeed.Feed,
	workingContext *working_context.RssCollectorWorkingContext,
) error {
	post := workingContext.Result.Post
	// date
	generatedTime, err := time.Parse(WeixinArticleDateFormat, article.Published)
	if err != nil {
		return utils.ImmediatePrintError(err)
	}
	post.ContentGeneratedAt = timestamppb.New(generatedTime)
	post.SubSource.Name = article.Author.Name
	// post.SubSource.AvatarUrl = res.Image.URL
	// w.UpdateAvatarUrl(post, res)
	post.OriginUrl = article.Link
	post.Title = article.Title

	if err != nil {
		return utils.ImmediatePrintError(err)
	}

	err = w.UpdateDedupId(workingContext)
	if err != nil {
		return utils.ImmediatePrintError(err)
	}

	return nil
}

func (w WeixinArticleRssCollector) CollectOneSubsourceOnePage(
	task *protocol.PanopticTask,
	subsource *protocol.PanopticSubSource,
) error {
	client := clients.NewHttpClientFromTaskParams(task)
	url := w.ConstructUrl(task, subsource)
	resp, err := client.Get(url)
	if err != nil {
		return utils.ImmediatePrintError(err)
	}

	fp := gofeed.NewParser()
	feed, err := fp.Parse(resp.Body)

	if err != nil {
		return utils.ImmediatePrintError(err)
	}

	for _, article := range feed.Items {
		fmt.Println("article", article)
		// working context for each message
		workingContext := &working_context.RssCollectorWorkingContext{
			SharedContext:   working_context.SharedContext{Task: task, Result: &protocol.CrawlerMessage{}, IntentionallySkipped: false},
			RssUrl:          url,
			SubSource:       subsource,
			RssResponseItem: article,
		}
		collector.InitializeRssCollectorResult(workingContext)
		err := w.UpdateResultFromArticle(article, feed, workingContext)
		if err != nil {
			task.TaskMetadata.TotalMessageFailed++
			return utils.ImmediatePrintError(err)
		}

		if workingContext.SharedContext.Result != nil {
			Logger.LogV2.Info("Weixin collected post: %s" + workingContext.Result.Post.Title)
			fmt.Println("wc", workingContext)
			sink.PushResultToSinkAndRecordInTaskMetadata(w.Sink, workingContext)
		}
	}

	return nil
}

// Support configable multi-page API call
func (w WeixinArticleRssCollector) CollectOneSubsource(task *protocol.PanopticTask, subsource *protocol.PanopticSubSource) error {
	return w.CollectOneSubsourceOnePage(task, subsource)
}

func (w WeixinArticleRssCollector) CollectAndPublish(task *protocol.PanopticTask) {
	collector.ParallelSubsourceApiCollect(task, w)
	collector.SetErrorBasedOnCounts(task, "weixin")
}
