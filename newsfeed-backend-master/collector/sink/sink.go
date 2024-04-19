package sink

import "github.com/rnr-capital/newsfeed-backend/protocol"

type CollectedDataSink interface {
	Push(msg *protocol.CrawlerMessage) error
}
