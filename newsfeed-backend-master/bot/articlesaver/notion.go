package articlesaver

import (
	"net/http"
	"os"
	"time"

	"github.com/dstotijn/go-notion"
)

func NewNotionClient() *notion.Client {
	apiKey := os.Getenv("NOTION_TOKEN")
	httpClient := &http.Client{Timeout: 20 * time.Second}
	client := notion.NewClient(apiKey, notion.WithHTTPClient(httpClient))
	return client
}
