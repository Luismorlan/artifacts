package collector_instances

import (
	"fmt"
	"math"
	"time"

	twitterscraper "github.com/n0madic/twitter-scraper"
	"github.com/rnr-capital/newsfeed-backend/collector"
	"github.com/rnr-capital/newsfeed-backend/collector/file_store"
	"github.com/rnr-capital/newsfeed-backend/collector/sink"
	"github.com/rnr-capital/newsfeed-backend/collector/working_context"
	"github.com/rnr-capital/newsfeed-backend/protocol"
	Logger "github.com/rnr-capital/newsfeed-backend/utils/log"
)

// Twitter's thread max lenth is 25, this number ensures that we capture 2 threads.
const TwitterBatchSize = 50

type TwitterApiCrawler struct {
	Sink sink.CollectedDataSink

	Scraper *twitterscraper.Scraper

	ImageStore file_store.CollectedFileStore
}

// MustLogin make sure that the scraper is logged in with 10 exponential backoff retries
func (t TwitterApiCrawler) MustLogin() {
	retries := 0
	for !t.Scraper.IsLoggedIn() {
		if retries > 10 {
			Logger.LogV2.Error("can't login to twitter")
			panic("can't login to twitter")
		}
		time.Sleep(time.Second * time.Duration(math.Pow(1.5, float64(retries))))
		fmt.Println("retrying login", retries)
		err := t.Scraper.LoginOpenAccount()
		if err != nil {
			Logger.LogV2.Error("can't login to twitter")
			panic(err)
		}
		retries += 1
	}
}

// Crawl and publish for a single Twitter user.
func (t TwitterApiCrawler) CollectOneSubsource(
	task *protocol.PanopticTask, subSource *protocol.PanopticSubSource) error {
	t.MustLogin()
	tweets, _, err := t.Scraper.FetchTweets(subSource.ExternalId, TwitterBatchSize, "")
	if err != nil {
		Logger.LogV2.Error(fmt.Sprintf("fail to collect tweeter user %s, %s", subSource.ExternalId, err))
		task.TaskMetadata.ResultState = protocol.TaskMetadata_STATE_FAILURE
		return err
	}
	for _, tweet := range FilterIncompleteTweet(tweets) {
		t.ProcessSingleTweet(tweet, task)
	}

	return nil
}

func (t TwitterApiCrawler) ProcessSingleTweet(tweet *twitterscraper.Tweet,
	task *protocol.PanopticTask) {
	t.MustLogin()
	workingContext := &working_context.ApiCollectorWorkingContext{
		SharedContext:   working_context.SharedContext{Task: task, IntentionallySkipped: false},
		ApiResponseItem: tweet,
	}
	if err := t.GetMessage(workingContext); err != nil {
		task.TaskMetadata.TotalMessageFailed++
		Logger.LogV2.Error(fmt.Sprintf("fail to collect twitter message from API response. message %s, err %s", collector.PrettyPrint(tweet), err))
		return
	}
	sink.PushResultToSinkAndRecordInTaskMetadata(t.Sink, workingContext)
}

func (t TwitterApiCrawler) GetMessage(workingContext *working_context.ApiCollectorWorkingContext) error {
	t.MustLogin()
	collector.InitializeApiCollectorResult(workingContext)
	tweet := workingContext.ApiResponseItem.(*twitterscraper.Tweet)
	post, err := ConvertTweetTreeToCrawledPost(tweet, t.Scraper, workingContext.Task, t.ImageStore)
	if err != nil {
		return err
	}
	workingContext.Result.Post = post
	return nil
}

func (t TwitterApiCrawler) CollectAndPublish(task *protocol.PanopticTask) {
	t.MustLogin()
	collector.ParallelSubsourceApiCollect(task, t)
	collector.SetErrorBasedOnCounts(task, "Twitter")
}
