package consumers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/rnr-capital/newsfeed-backend/notifier"
	"github.com/rnr-capital/newsfeed-backend/notifier/consumers/onesignal"
	Logger "github.com/rnr-capital/newsfeed-backend/utils/log"
)

// REST API DOC: https://documentation.onesignal.com/reference/view-devices

const (
	REST_API_KEY = "MzFkMTY2ZGItZTFmZi00NzFkLTlkMDAtNDk0NjQ3MWYwMWFj"
	// https://pkg.go.dev/github.com/tbalthazar/onesignal-go
	ONESIGNAL_APP_ID = "fb0cfb9c-c9c0-4c05-a004-b052acfbc463"
)

var (
	Log = Logger.LogV2
)

type OneSignalAdapter struct {
	client *onesignal.Client
}

var _ notifier.INotificationConsumer = &OneSignalAdapter{}

func NewOneSignalAdapter() *OneSignalAdapter {
	client := onesignal.NewClient(nil)
	// client.UserKey = REST_API_KEY // which is not needed sincee adapter is using AppKey
	client.AppKey = REST_API_KEY // here should not use ONESIGNAL_APP_ID
	return &OneSignalAdapter{
		client: client,
	}
}

func (o *OneSignalAdapter) PushNotification(job notifier.NotificationOutputJob) (*http.Response, error) {
	Logger.LogV2.Info(fmt.Sprint("Received push notificaion", job))
	if len(job.UserIds) == 0 {
		return nil, nil
	}

	imageAttachments := map[string]string{}
	for index, imageUrl := range job.Images {
		imageAttachments[strconv.Itoa(index)] = imageUrl // key doesn't matter
	}

	subsourceAvarUrlsStr := ""
	urlDelimiter := "**"
	for index, avatarUrl := range job.SubsourceAvatarUrls {
		if index == 0 {
			subsourceAvarUrlsStr = avatarUrl
		} else {
			subsourceAvarUrlsStr += (urlDelimiter + avatarUrl)
		}
	}

	notificationReq := &onesignal.NotificationRequest{
		AppID:    ONESIGNAL_APP_ID,
		Headings: map[string]string{"en": job.Title},
		Subtitle: map[string]string{"en": job.Subtitle},
		Contents: map[string]string{"en": job.Description},
		Data: map[string]string{
			"columnId":             job.ColumnId,
			"subsourceAvarUrlsStr": subsourceAvarUrlsStr,
			"urlDelimitter":        urlDelimiter,
		},
		IncludeExternalUserIDs:    job.UserIds,
		ChannelForExternalUserIds: "push",
		IOSAttachments:            imageAttachments,
	}
	createRes, res, err := o.client.Notifications.Create(notificationReq)
	if err != nil {
		Log.Errorf("Failed to create notification from one signal, res: ", res, ", createRes: ", createRes)
	}
	return res, err
}
