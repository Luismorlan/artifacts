package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rnr-capital/newsfeed-backend/model"
	Logger "github.com/rnr-capital/newsfeed-backend/utils/log"
	"github.com/slack-go/slack"
)

func buildPostLink(post model.Post) string {
	if post.OriginUrl != "" {
		return post.OriginUrl
	}
	return fmt.Sprintf("https://rnr.capital/shared-posts/%s", post.Id)
}

func buildFromUserBlock(userName string, comment string) slack.Block {
	if comment == "" {
		return slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("%s shared a news", userName), false, false), nil, nil)
	}
	return slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("%s shared: %s", userName, comment), false, false), nil, nil)
}

func buildSubsourceBlock(post model.Post) slack.Block {
	return slack.NewContextBlock("",
		slack.NewImageBlockElement(post.SubSource.AvatarUrl, post.SubSource.Name),
		slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("<%s|%s>", buildPostLink(post), post.SubSource.Name), false, false))
}

func buildRetweetBlock(post model.Post, postLink string) slack.MixedElement {
	return slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("><%s|%s> %s", postLink, post.SubSource.Name, buildContentWithShowMore(post, postLink)), false, false)
}

// buildImageElements should be used only when we have 2+ images
func buildImageElements(post model.Post) []slack.MixedElement {
	elements := []slack.MixedElement{}
	for _, imageUrl := range post.ImageUrls {
		elements = append(elements, slack.NewImageBlockElement(imageUrl, "post image"))
	}
	return elements
}

func buildFileObject(post model.Post) *slack.TextBlockObject {
	fileBlockText := "```"
	for i, url := range post.FileUrls {
		if i > 0 {
			fileBlockText += "\n"
		}
		fileBlockText += fmt.Sprintf("<%s|📄 %s>", url, url[strings.LastIndex(url, "/")+1:])
	}
	fileBlockText += "```"
	return slack.NewTextBlockObject("mrkdwn", fileBlockText, false, false)
}

func buildContentWithShowMore(post model.Post, postLink string) string {
	contentRunes := []rune(post.Content)
	if len(contentRunes) > 400 {
		return fmt.Sprintf("%s...<%s|[查看全文]>", string(contentRunes[:400]), postLink)
	}
	return post.Content
}

func TimeBoundedPushPost(ctx context.Context, webhookUrl string, post model.Post) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		sharePost := SharePostPayload{
			Post:       post,
			WebhookUrl: webhookUrl,
		}
		postBytes, _ := json.Marshal(sharePost)
		_, err := http.Post(os.Getenv("BOT_SHARE_POST_URL"), "application/json", bytes.NewReader(postBytes))
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			Logger.LogV2.Error(fmt.Sprint("failed to push post to channel", err))
		}
		return
	case <-ctx.Done():
		Logger.LogV2.Error(fmt.Sprintf("push post via webhook timed out. post: %s, webhook url: %s", post.Id, webhookUrl))
		return
	}
}

func TimeBoundedNotifyPost(ctx context.Context, post model.Post, columns []*model.Column) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	done := make(chan error, 1)
	columnsDedup := []model.Column{}
	seenColumnIds := map[string]struct{}{}

	for _, c := range columns {
		if _, ok := seenColumnIds[c.Id]; !ok {
			columnsDedup = append(columnsDedup, *c)
		}
		seenColumnIds[c.Id] = struct{}{}
	}
	go func() {
		sharePost := PostNotifyPayload{
			Post:    post,
			Columns: columnsDedup,
		}
		postBytes, _ := json.Marshal(sharePost)
		_, err := http.Post(os.Getenv("BOT_NOTIFY_POST_URL"), "application/json", bytes.NewReader(postBytes))
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			Logger.LogV2.Error(fmt.Sprintf("failed to notify post to user %v", err))
		}
		return
	case <-ctx.Done():
		Logger.LogV2.Error(fmt.Sprintf("push post via webhook timed out. post: %s, columns: %v", post.Id, columns))
		return
	}
}

// PushPostViaWebhook is an async call to push a post to a channel
func PushPostViaWebhook(post model.Post, webhookUrl string, fromUser string, comment string) error {
	blocks := []slack.Block{}
	if fromUser != "" {
		blocks = append(blocks, buildFromUserBlock(fromUser, comment))
	}
	blocks = append(blocks, buildSubsourceBlock(post))
	// build subsource and post body blocks

	if post.SharedFromPost != nil {
		sharedFromPost := post.SharedFromPost
		if post.Content != "" {
			sharedFromContext := slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", buildContentWithShowMore(post, buildPostLink(post)), false, false), nil, nil)
			blocks = append(blocks, sharedFromContext)
		}
		sharedFromContextElements := []slack.MixedElement{buildRetweetBlock(*sharedFromPost, buildPostLink(post))}
		if len(sharedFromPost.ImageUrls) > 1 {
			sharedFromContextElements = append(sharedFromContextElements, buildImageElements(*sharedFromPost)...)
		}
		blocks = append(blocks, slack.NewContextBlock("", sharedFromContextElements...))
	} else {
		if post.Title != "" {
			blocks = append(blocks, slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*%s*", post.Title), false, false), nil, nil))
		}
		if post.Content != "" {
			blocks = append(blocks, slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", buildContentWithShowMore(post, buildPostLink(post)), false, false), nil, nil))
		}
		if len(post.ImageUrls) > 1 {
			blocks = append(blocks, slack.NewContextBlock("", buildImageElements(post)...))
		}
		if len(post.FileUrls) > 0 {
			blocks = append(blocks, slack.NewContextBlock("", buildFileObject(post)))
		}
	}

	if len(post.ImageUrls) == 1 {
		blocks = append(blocks, slack.NewImageBlock(post.ImageUrls[0], "post image", "", nil))
	}

	webhookMsg := &slack.WebhookMessage{
		Text:   fmt.Sprintf("%s: %s...", post.SubSource.Name, string([]rune(post.Content)[:30])),
		Blocks: &slack.Blocks{BlockSet: blocks},
	}

	if fromUser == "" {
		webhookMsg.Text = fmt.Sprintf("%s: %s...", post.SubSource.Name, string([]rune(post.Content)[:30]))
	} else {
		webhookMsg.Text = fmt.Sprintf("%s shared: %s...", fromUser, string([]rune(post.Content)[:30]))
	}

	err := slack.PostWebhook(webhookUrl, webhookMsg)
	if err != nil {
		Logger.LogV2.Error(fmt.Sprintf("failed to post to slack %s. %v", webhookUrl, err.Error()))
		return err
	}

	return nil
}
