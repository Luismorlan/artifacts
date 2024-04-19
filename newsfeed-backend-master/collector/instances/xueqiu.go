package collector_instances

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
	"time"

	"github.com/rnr-capital/newsfeed-backend/collector"
	"github.com/rnr-capital/newsfeed-backend/collector/clients"
	"github.com/rnr-capital/newsfeed-backend/collector/file_store"
	sink "github.com/rnr-capital/newsfeed-backend/collector/sink"
	"github.com/rnr-capital/newsfeed-backend/collector/working_context"
	"github.com/rnr-capital/newsfeed-backend/protocol"
	"github.com/rnr-capital/newsfeed-backend/utils"
	Logger "github.com/rnr-capital/newsfeed-backend/utils/log"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	baseurl = "https://xueqiu.com/v4/statuses/user_timeline.json?page=1&user_id=%s&type=0"
)

type XueqiuApiCollector struct {
	Sink       sink.CollectedDataSink
	ImageStore file_store.CollectedFileStore
}

type XueqiuUser struct {
	Name            string `json:"screen_name"`
	ExternalId      int    `json:"id"`
	Profile         string `json:"profile"`
	ProfileImageUrl string `json:"profile_image_url"`
	PhotoDomain     string `json:"photo_domain"`
}

type XueqiuItem struct {
	Title       string      `json:"title"`
	ContentText string      `json:"text"`
	DisplayTime int         `json:"created_at"`
	ID          int         `json:"id"`
	User        *XueqiuUser `json:"user"`
	Pic         string      `json:"pic"`
}

type XueqiuApiResponse struct {
	Count    int           `json:"count"`
	Statuses []*XueqiuItem `json:"statuses"`
}

func (w XueqiuApiResponse) UpdateFileUrls(workingContext *working_context.ApiCollectorWorkingContext) error {
	return errors.New("UpdateFileUrls not implemented, should not be called")
}

func (w XueqiuApiCollector) ConstructUrl(task *protocol.PanopticTask, subsource *protocol.PanopticSubSource, paginationInfo *working_context.PaginationInfo) string {
	return fmt.Sprintf(baseurl, subsource.ExternalId)
}

func (w XueqiuApiCollector) UpdateResultFromItem(item *XueqiuItem, workingContext *working_context.ApiCollectorWorkingContext) error {
	generatedTime := time.UnixMilli(int64(item.DisplayTime))
	workingContext.Result.Post.ContentGeneratedAt = timestamppb.New(generatedTime)
	workingContext.Result.Post.DeduplicateId = fmt.Sprintf("xueqiu-%d", item.ID)
	if err := w.UpdateImages(item, workingContext.Result.Post); err != nil {
		return utils.ImmediatePrintError(err)
	}
	contentText := strings.ReplaceAll(item.ContentText, "<br/><br/>", "\n")
	contentText = strings.ReplaceAll(contentText, "&nbsp;", " ")
	allBracket := regexp.MustCompile(`<[^>]*>`)
	workingContext.Result.Post.Content = allBracket.ReplaceAllString(contentText, "")
	if item.Title != "" {
		workingContext.Result.Post.Title = item.Title
	}
	workingContext.Result.Post.OriginUrl = fmt.Sprintf("https://xueqiu.com/%d/%d", item.User.ExternalId, item.ID)
	workingContext.NewsType = protocol.PanopticSubSource_USERS
	workingContext.Result.Post.SubSource.Name = item.User.Name
	workingContext.Result.Post.SubSource.ExternalId = fmt.Sprint(item.User.ExternalId)
	avatarUrls := strings.Split(item.User.ProfileImageUrl, ",")
	if (len(avatarUrls)) == 0 {
		return errors.New("Fatal Error: invalid profile image: " + item.User.ProfileImageUrl)
	}
	workingContext.Result.Post.SubSource.AvatarUrl = fmt.Sprintf("https:%s%s!240x240.jpg", item.User.PhotoDomain, avatarUrls[0])
	workingContext.Result.Post.SubSource.OriginUrl = fmt.Sprintf("https://xueqiu.com/u/%d", item.User.ExternalId)
	return nil
}

func (w XueqiuApiCollector) UpdateImages(item *XueqiuItem, post *protocol.CrawlerMessage_CrawledPost) error {
	imageUrls := []string{}
	if len(item.Pic) == 0 {
		return nil
	}
	for _, pic := range strings.Split(item.Pic, ",") {
		imageUrls = append(imageUrls, strings.TrimSuffix(pic, "!thumb.jpg"))
	}
	s3OrOriginalUrls, err := collector.UploadImagesToS3(w.ImageStore, imageUrls, nil)
	if err != nil {
		Logger.LogV2.Error("fail to upload xueqiu image, err:" + err.Error())
	}
	post.ImageUrls = s3OrOriginalUrls
	return nil
}

func (w XueqiuApiCollector) CollectOneSubsourceOnePage(
	task *protocol.PanopticTask,
	subsource *protocol.PanopticSubSource,
	paginationInfo *working_context.PaginationInfo,
) error {
	client, err := clients.NewXueqiuHttpClient()
	if err != nil {
		return err
	}

	url := w.ConstructUrl(task, subsource, paginationInfo)
	resp, err := client.Get(url)
	if err != nil {
		return utils.ImmediatePrintError(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	res := &XueqiuApiResponse{}
	err = json.Unmarshal(body, res)
	if err != nil {
		return utils.ImmediatePrintError(err)
	}

	if res.Count == 0 {
		return fmt.Errorf("no posts found %v", res)
	}

	// We travese from the end for a better time order of posts crawled(oldest cralwed first)
	for i := len(res.Statuses) - 1; i >= 0; i -= 1 {
		workingContext := &working_context.ApiCollectorWorkingContext{
			SharedContext:  working_context.SharedContext{Task: task, Result: &protocol.CrawlerMessage{}, IntentionallySkipped: false},
			PaginationInfo: paginationInfo,
			ApiUrl:         url,
			SubSource:      subsource,
		}

		collector.InitializeApiCollectorResult(workingContext)
		err := w.UpdateResultFromItem(res.Statuses[i], workingContext)

		if err != nil {
			task.TaskMetadata.TotalMessageFailed++
			return utils.ImmediatePrintError(err)
		}

		if workingContext.SharedContext.Result != nil {
			sink.PushResultToSinkAndRecordInTaskMetadata(w.Sink, workingContext)
		}
	}
	return nil
}

// Support configable multi-page API call
// Iterate on each channel
func (w XueqiuApiCollector) CollectOneSubsource(task *protocol.PanopticTask, subsource *protocol.PanopticSubSource) error {
	// Xueqiu uses channels and only know subsource after each message if fetched
	w.CollectOneSubsourceOnePage(task, subsource, nil)

	collector.SetErrorBasedOnCounts(task, "Xueqiu kuaixun")
	return nil
}

func (w XueqiuApiCollector) CollectAndPublish(task *protocol.PanopticTask) {
	collector.ParallelSubsourceApiCollect(task, w)
	collector.SetErrorBasedOnCounts(task, "xueqiu")
}
