package consumers

import (
	"testing"

	"github.com/rnr-capital/newsfeed-backend/notifier"
	"github.com/stretchr/testify/require"
)

const (
	// MockNotificationPush   = true // it will actually send notification if set to false
	external_user_id_yifan = "24f075ad-7f0c-4347-a8ef-e34b1e0204dd"
)

// type MockNotifications struct {
// 	onesignal.NotificationsService
// }

// func (m *MockNotifications) Create(opt *onesignal.NotificationRequest) (*onesignal.NotificationCreateResponse, *http.Response, error) {
// 	createRes := &onesignal.NotificationCreateResponse{}
// 	resp := &http.Response{}
// 	return createRes, resp, nil
// }

func TestOneSignalAdapter(t *testing.T) {
	t.Run("Test push notification", func(t *testing.T) {
		onesignalClient := NewOneSignalAdapter()
		// if MockNotificationPush {
		// 	mockNotification := &MockNotifications{}
		// 	onesignalClient.client.Notifications = mockNotification
		// }
		job := notifier.NotificationOutputJob{
			Title:       "title - unit test",
			Description: "description-line1\ndescription-line2\ndescription-line3\ndescription-line4",
			Subtitle:    "subtitle - unit test",
			UserIds:     []string{external_user_id_yifan},
			Images: []string{
				"https://d20uffqoe1h0vv.cloudfront.net/16b50947517f65e0ab4f9eb847fb21f5.jpg",
				"https://d20uffqoe1h0vv.cloudfront.net/7f5c6bb0fd4df4bf0c6cff5eb182e880.jpg",
				// "https://d2cana2fc4gv86.cloudfront.net/eyJidWNrZXQiOiJuZXdzZmVlZC1jcmF3bGVyLWltYWdlLW91dHB1dCIsImtleSI6ImE5ODE0OGQyZmE2N2E1MmFiNTAzZDQwNmU3YWE5Nzc5LmpwZyIsImVkaXRzIjp7InJlc2l6ZSI6eyJ3aWR0aCI6NjAsImhlaWdodCI6NjAsImZpdCI6ImNvdmVyIiwiYmFja2dyb3VuZCI6eyJyIjoyNTUsImciOjI1NSwiYiI6MjU1LCJhbHBoYSI6MjU1fX19fQ==",
				// "https://d2cana2fc4gv86.cloudfront.net/eyJidWNrZXQiOiJuZXdzZmVlZC1jcmF3bGVyLWltYWdlLW91dHB1dCIsImtleSI6IjJmNzRhYjZmYzM5NDhkYWRjZmI0YmM2MDdiNjQ2YmYyLmpwZyIsImVkaXRzIjp7InJlc2l6ZSI6eyJ3aWR0aCI6NjAsImhlaWdodCI6NjAsImZpdCI6ImNvdmVyIiwiYmFja2dyb3VuZCI6eyJyIjoyNTUsImciOjI1NSwiYiI6MjU1LCJhbHBoYSI6MjU1fX19fQ==",
				// "https://d20uffqoe1h0vv.cloudfront.net/abc2de17114f4476d9c7c2061bc9f362.jpg",
				// "https://cdn.pixabay.com/photo/2018/01/21/01/46/architecture-3095716_960_720.jpg",
			},
			SubsourceAvatarUrls: []string{
				"https://newsfeed-logo.s3.us-west-1.amazonaws.com/gelonghui.png",
				"https://newsfeed-logo.s3.us-west-1.amazonaws.com/cls.png",
			},
			ColumnId: "Unit test Column id",
		}
		_, error := onesignalClient.PushNotification(job)
		require.True(t, error == nil)
	})
}
