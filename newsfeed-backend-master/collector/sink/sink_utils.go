package sink

import (
	"fmt"

	"github.com/rnr-capital/newsfeed-backend/collector/validation"
	"github.com/rnr-capital/newsfeed-backend/collector/working_context"
	"github.com/rnr-capital/newsfeed-backend/protocol"
	Logger "github.com/rnr-capital/newsfeed-backend/utils/log"
)

// Push a result into data sink and update task metadata on working context
// - On Success: Increment collected message counter on task.
// - On Failure: Log error and increment failure message counter on task.
func PushResultToSinkAndRecordInTaskMetadata(s CollectedDataSink, workingContext interface{}) {
	var sharedContext *working_context.SharedContext
	switch wc := workingContext.(type) {
	case *working_context.CrawlerWorkingContext:
		sharedContext = &wc.SharedContext
	case *working_context.ApiCollectorWorkingContext:
		sharedContext = &wc.SharedContext
	case *working_context.RssCollectorWorkingContext:
		sharedContext = &wc.SharedContext
	case *working_context.SharedContext:
		sharedContext = wc
	}

	if sharedContext.IntentionallySkipped {
		sharedContext.Task.TaskMetadata.TotalMessageSkipped++
		return
	}

	if err := validation.ValidateSharedContext(sharedContext); err != nil {
		sharedContext.Task.TaskMetadata.ResultState = protocol.TaskMetadata_STATE_FAILURE
		sharedContext.Task.TaskMetadata.TotalMessageFailed++
		switch wc := workingContext.(type) {
		case *working_context.CrawlerWorkingContext:
			html, _ := wc.Element.DOM.Html()
			Logger.LogV2.Error(fmt.Sprintf("crawled message failed validation, Error: %s \n, Html %s", err, html))
		default:
			Logger.LogV2.Error(fmt.Sprintf("crawled message failed validation, Error: %s", err))
		}
		return
	}

	if err := s.Push(sharedContext.Result); err != nil {
		sharedContext.Task.TaskMetadata.ResultState = protocol.TaskMetadata_STATE_FAILURE
		sharedContext.Task.TaskMetadata.TotalMessageFailed++
		Logger.LogV2.Error(fmt.Sprintf("fail to publish message %s to Sink. Task: %s, Error: %s", sharedContext.Result.String(), sharedContext.Task.String(), err))
		return
	}
	sharedContext.Task.TaskMetadata.TotalMessageCollected++
}
