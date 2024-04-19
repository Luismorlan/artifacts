package collector_instances

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly"
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
	Wublock123ChannelUrl   = "https://www.wublock123.com/html/"
	Wublock123SearchUrl    = "https://api.wublock123.com/api/site/getSearchArticleList"
	Wublock123HitsUrl      = "https://api.wublock123.com/api/site/getHitsList"
	Wublock123DepthHitsUrl = "https://api.wublock123.com/api/site/getDepthHitsList"
	HITS_CHANNEL           = "hits"      // 阅读最多
	DEPTH_HITS_CHANNEL     = "depthhits" // 深度阅读
	PAGE_SIZE              = 15
	// Select all the news item in a channel
	Wublock123ChannelItemSelector = "div.list li"
)

type Wublock123SubsourceType int

const (
	subsourceTypeChannel Wublock123SubsourceType = iota
	subsourceTypeSearch
	subsourceTypeHits
	subsourceTypeDepthHits
)

type Wublock123Collector struct {
	Sink       sink.CollectedDataSink
	ImageStore file_store.CollectedFileStore
}

type Wublock123SubSource struct {
	protocol.PanopticSubSource
	SubSourceType Wublock123SubsourceType
}

type Wublock123Item struct {
	Title      string `json:"title"`
	Thumb      string `json:"thumb"` // evidently mostly empty
	Date       string `json:"date"`
	ID         int    `json:"id"`
	CategoryId int    `json:"catid"`
	Url        string `json:"url"`
	Views      int    `json:"views"`
	InputTime  int    `json:"inputtime"`
	HitsId     string `json:"hitsid"`
}

type Wublock123ApiResponse struct {
	Code       int              `json:"code"`
	Success    bool             `json:"success"`
	Data       []Wublock123Item `json:"data"`
	PageIndex  int              `json:"pageIndex"`
	TotalPages int              `json:"totalPages"`
	Total      int              `json:"total"`
}

var idToNameMap = map[string]string{
	"exchange":  "交易所",
	"defi":      "DeFi",
	"dao":       "DAO",
	"pay":       "支付",
	"mining":    "挖矿",
	"supervise": "监管",
	"gamefi":    "GameFi",
	"jigou":     "机构",
	"gl":        "L1",
	"l2":        "L2",
	"kepu":      "科普",
	"BTC":       "比特币",
	"ETH":       "以太坊",
	"hits":      "阅读最多",
	"depthhits": "深度阅读",
	"NFT":       "NFT",
	"aq":        "安全",
	"香港":        "香港",
	"新加坡":       "新加坡",
	"比特大陆":      "比特大陆",
	"币安":        "币安",
}

func GetWublock123NameById(id string) string {
	if name, ok := idToNameMap[id]; ok {
		return name
	}
	return id
}

func (w Wublock123Collector) GetIDFromUrl(Url string) string {
	// https://www.wublock123.com/index.php?m=content&c=index&a=show&catid=10&id=22079
	// id = 22079
	m, err := url.ParseQuery(Url)
	if err != nil {
		return ""
	}
	if id, ok := m["id"]; ok {
		return id[0]
	}
	return ""
}

func (w Wublock123Collector) UpdateResultFromPageAndPush(r *working_context.SharedContext, subsource *protocol.PanopticSubSource, datum *Wublock123Item) error {

	// Visit the URL and update the Result object
	c := colly.NewCollector()

	c.OnHTML("body", func(e *colly.HTMLElement) {
		r.Result.Post.Title = utils.FallbackString(e.ChildText("div.title"), datum.Title)
		r.Result.Post.Content = utils.FallbackString(e.ChildText("div.entry-content"), datum.Title)
		r.Result.Post.OriginUrl = datum.Url
		// timestamp is from InputTime
		if datum.InputTime != 0 {
			r.Result.Post.ContentGeneratedAt = timestamppb.New(time.Unix(int64(datum.InputTime), 0))
		} else { // timestamp is from Date
			t, err := utils.ParseDate(datum.Date)
			if err != nil {
				r.Result.Post.ContentGeneratedAt = timestamppb.New(t)
			} else { // can't parse time, use crawl time
				r.Result.Post.ContentGeneratedAt = timestamppb.Now()
			}
		}
		if datum.ID != 0 {
			r.Result.Post.DeduplicateId = fmt.Sprintf("wublock123-%d", datum.ID)
		} else {
			r.Result.Post.DeduplicateId = fmt.Sprintf("wublock123-%s", w.GetIDFromUrl(datum.Url))
		}

		// we don't have individual subsource logo for wublock123
		r.Result.Post.SubSource.AvatarUrl = collector.GetSourceLogoUrl(r.Task.TaskParams.SourceId)
		if imageUrls := e.ChildAttrs("div.entry-content img", "src"); len(imageUrls) > 0 {
			r.Result.Post.ImageUrls = imageUrls
		}

		r.Result.Post.SubSource.Name = subsource.Name
		r.Result.Post.SubSource.ExternalId = subsource.ExternalId
		r.Result.Post.SubSource.OriginUrl = subsource.Link
		r.Result.Post.SubSource.SourceId = r.Task.TaskParams.SourceId
	})

	err := c.Visit(datum.Url)
	switch {
	case err != nil:
		r.Task.TaskMetadata.TotalMessageFailed++
		return utils.ImmediatePrintError(err)
	case r.Result == nil:
		r.Task.TaskMetadata.TotalMessageFailed++
		return utils.ImmediatePrintError(errors.New("Wublock123 result is nil"))
	default:
		sink.PushResultToSinkAndRecordInTaskMetadata(w.Sink, r)
		return nil
	}
}

// ProcessSingleDatum goes to the detail page of the datum and extracts the content. This is from Wublock123's API endpoint
func (w Wublock123Collector) ProcessSingleDatum(workingContext *working_context.ApiCollectorWorkingContext, datum *Wublock123Item) error {
	collector.InitializeApiCollectorResult(workingContext)
	return w.UpdateResultFromPageAndPush(&workingContext.SharedContext, workingContext.SubSource, datum)
}

// ProcessSingleUrl goes to the detail page of the url and extracts the content. This is from Wublock123's html endpoint
func (w Wublock123Collector) ProcessSingleUrl(workingContext *working_context.CrawlerWorkingContext, datum *Wublock123Item) error {
	collector.InitializeCrawlerResult(workingContext)
	return w.UpdateResultFromPageAndPush(&workingContext.SharedContext, workingContext.SubSource, datum)
}

// CollectByAPI collects data from the API using the provided working context and body function.
// It iterates over the specified number of pages and processes each datum.
// Returns an error if any error occurs during the collection process.
func (w Wublock123Collector) CollectByAPI(ctx *working_context.ApiCollectorWorkingContext, bodyFunc func(int) string) error {
	client := clients.NewHttpClientFromTaskParams(ctx.Task)
	for pageIndex := 1; pageIndex <= int(ctx.Task.TaskParams.GetWublock123TaskParams().Pages); pageIndex++ {
		resp, err := client.Post(ctx.ApiUrl, strings.NewReader(bodyFunc(pageIndex)))
		if err != nil {
			return utils.ImmediatePrintError(err)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return utils.ImmediatePrintError(err)
		}
		res := &Wublock123ApiResponse{}
		err = json.Unmarshal(body, res)
		if err != nil {
			return utils.ImmediatePrintError(err)
		}
		for _, datum := range res.Data {
			err := w.ProcessSingleDatum(ctx, &datum)
			if err != nil {
				return utils.ImmediatePrintError(err)
			}
		}
	}
	fmt.Printf("Collected %d items from %s\n", ctx.Task.TaskMetadata.TotalMessageCollected, ctx.ApiUrl)
	return nil
}

func (w Wublock123Collector) CollectHits(task *protocol.PanopticTask, subsource *Wublock123SubSource) error {
	var workingContext working_context.ApiCollectorWorkingContext = working_context.ApiCollectorWorkingContext{
		SharedContext: working_context.SharedContext{
			Task:                 task,
			IntentionallySkipped: false},
		ApiUrl:    Wublock123HitsUrl,
		SubSource: &subsource.PanopticSubSource,
		PaginationInfo: &working_context.PaginationInfo{
			CurrentPageCount: 0,
			NextPageId:       "0"},
		NewsType: protocol.PanopticSubSource_CHANNEL,
	}

	return w.CollectByAPI(&workingContext, func(pageIndex int) string {
		return fmt.Sprintf(`pageIndex=%d&pageSize=%d`, pageIndex, PAGE_SIZE)
	})
}

func (w Wublock123Collector) CollectDepthHits(task *protocol.PanopticTask, subsource *Wublock123SubSource) error {
	var workingContext working_context.ApiCollectorWorkingContext = working_context.ApiCollectorWorkingContext{
		SharedContext: working_context.SharedContext{
			Task:                 task,
			IntentionallySkipped: false},
		ApiUrl:    Wublock123DepthHitsUrl,
		SubSource: &subsource.PanopticSubSource,
		PaginationInfo: &working_context.PaginationInfo{
			CurrentPageCount: 0,
			NextPageId:       "0"},
		NewsType: protocol.PanopticSubSource_CHANNEL,
	}

	return w.CollectByAPI(&workingContext, func(pageIndex int) string {
		return fmt.Sprintf(`pageIndex=%d&pageSize=%d`, pageIndex, PAGE_SIZE)
	})
}

func (w Wublock123Collector) CollectByChannel(task *protocol.PanopticTask, subsource *Wublock123SubSource) error {
	c := colly.NewCollector()
	wg := sync.WaitGroup{}
	c.OnHTML(Wublock123ChannelItemSelector, func(e *colly.HTMLElement) {
		datum := &Wublock123Item{
			Title: e.ChildText("a"),
			Url:   e.ChildAttr("a", "href"),
			Date:  e.ChildText("span")}
		wg.Add(1)
		go func() {
			defer wg.Done()

			ctx := &working_context.CrawlerWorkingContext{
				SharedContext: working_context.SharedContext{
					Task:                 task,
					IntentionallySkipped: false},
				SubSource: &subsource.PanopticSubSource,
				NewsType:  protocol.PanopticSubSource_CHANNEL,
				Element:   e,
				OriginUrl: datum.Url,
			}

			err := w.ProcessSingleUrl(ctx, datum)
			if err != nil {
				Logger.LogV2.Error(fmt.Sprintf("Wublock123 %v", err))
				ctx.SharedContext.Task.TaskMetadata.TotalMessageFailed++
				return
			}
		}()
	})

	c.Visit(subsource.Link)
	wg.Wait()
	fmt.Printf("Collected %d items from %s\n", task.TaskMetadata.TotalMessageCollected, subsource.Link)
	return nil
}

func (w Wublock123Collector) CollectBySearchKeyword(task *protocol.PanopticTask, subsource *Wublock123SubSource) error {
	var workingContext working_context.ApiCollectorWorkingContext = working_context.ApiCollectorWorkingContext{
		SharedContext: working_context.SharedContext{
			Task:                 task,
			IntentionallySkipped: false},
		ApiUrl:    Wublock123SearchUrl,
		SubSource: &subsource.PanopticSubSource,
		PaginationInfo: &working_context.PaginationInfo{
			CurrentPageCount: 0,
			NextPageId:       "0"},
		NewsType: protocol.PanopticSubSource_CHANNEL,
	}

	return w.CollectByAPI(&workingContext, func(pageIndex int) string {
		return fmt.Sprintf(`pageIndex=%d&keyword=%s&pageSize=%d`, pageIndex, subsource.ExternalId, PAGE_SIZE)
	})
}

func (w Wublock123Collector) CollectOneSubsource(task *protocol.PanopticTask, subsource *Wublock123SubSource) error {
	switch subsource.SubSourceType {
	case subsourceTypeChannel:
		return w.CollectByChannel(task, subsource)
	case subsourceTypeSearch:
		return w.CollectBySearchKeyword(task, subsource)
	case subsourceTypeHits:
		return w.CollectHits(task, subsource)
	case subsourceTypeDepthHits:
		return w.CollectDepthHits(task, subsource)
	default:
		return fmt.Errorf("unknown subsource type %v", subsource.SubSourceType)
	}
}

func (w Wublock123Collector) GetSubsourcesFromTaskParams(task *protocol.PanopticTask) ([]*Wublock123SubSource, error) {
	var subsources []*Wublock123SubSource

	if len(task.TaskParams.GetWublock123TaskParams().Channels) == 0 && len(task.TaskParams.GetWublock123TaskParams().SearchKeywords) == 0 {
		return nil, errors.New("Wublock123 must specify either channel or search_keyword")
	}

	if len(task.TaskParams.GetWublock123TaskParams().Channels) > 0 {
		for _, channel := range task.TaskParams.GetWublock123TaskParams().Channels {
			var subsource Wublock123SubSource
			if channel == HITS_CHANNEL {
				subsource = Wublock123SubSource{
					PanopticSubSource: protocol.PanopticSubSource{
						Name:       GetWublock123NameById(channel),
						ExternalId: channel,
						Link:       Wublock123HitsUrl},
					SubSourceType: subsourceTypeHits,
				}
			} else if channel == DEPTH_HITS_CHANNEL {
				subsource = Wublock123SubSource{
					PanopticSubSource: protocol.PanopticSubSource{
						Name:       GetWublock123NameById(channel),
						ExternalId: channel,
						Link:       Wublock123DepthHitsUrl},
					SubSourceType: subsourceTypeDepthHits,
				}
			} else {
				subsource = Wublock123SubSource{
					PanopticSubSource: protocol.PanopticSubSource{
						Name:       GetWublock123NameById(channel),
						ExternalId: channel,
						Link:       Wublock123ChannelUrl + channel},
					SubSourceType: subsourceTypeChannel,
				}
			}
			subsources = append(subsources, &subsource)
		}
	} else if len(task.TaskParams.GetWublock123TaskParams().SearchKeywords) > 0 {
		for _, keyword := range task.TaskParams.GetWublock123TaskParams().SearchKeywords {
			subsource := Wublock123SubSource{
				PanopticSubSource: protocol.PanopticSubSource{
					Name:       GetWublock123NameById(keyword),
					ExternalId: keyword,
					Link:       Wublock123SearchUrl},
				SubSourceType: subsourceTypeSearch,
			}
			subsources = append(subsources, &subsource)
		}
	}
	return subsources, nil
}

func (w Wublock123Collector) CollectAndPublish(task *protocol.PanopticTask) {
	subsources, err := w.GetSubsourcesFromTaskParams(task)
	if err != nil {
		Logger.LogV2.Error(fmt.Sprintf("Wublock123 %v", err))
		return
	}

	var wg sync.WaitGroup
	for _, ss := range subsources {
		wg.Add(1)
		go func(ss *Wublock123SubSource) {
			defer wg.Done()
			if err := w.CollectOneSubsource(task, ss); err != nil {
				Logger.LogV2.Error(fmt.Sprintf("Wublock123 %v", err))
				task.TaskMetadata.ResultState = protocol.TaskMetadata_STATE_FAILURE
			}
		}(ss)
	}

	wg.Wait()
	Logger.LogV2.Info(fmt.Sprint("Finished collecting wublock123 subsources, Task", task))
}
