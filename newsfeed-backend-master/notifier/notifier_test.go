package notifier

import (
	"net/http"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/rnr-capital/newsfeed-backend/model"
	"github.com/rnr-capital/newsfeed-backend/utils/dotenv"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

const (
	TickerCycle                   = 500 * time.Millisecond
	NotifierProcessingWaitingTime = 50 * time.Millisecond
	NotifierIntakeSize            = 100
	NotifierOutputSize            = 100
	CycleCount                    = 5
	PostDedupTTL                  = time.Hour
)

var (
	users []*model.User = []*model.User{
		{
			Id: "userId-1",
		},
		{
			Id: "userId-2",
		},
		{
			Id: "userId-3",
		},
		{
			Id: "userId-4",
		},
		{
			Id: "userId-5",
		},
		{
			Id: "userId-6",
		},
	}
	posts []model.Post = []model.Post{
		{ // 0
			Id:                 "40a29c76-d483-4043-84a7-35030f1cf9ba",
			Title:              "财联社1月28日互动平台精选",
			Content:            "今日大盘波动较大，沪指收跌0.97%，深成指跌0.53%，创业板指涨0.07%。旅游、教育、猪肉、三胎等板块涨幅居前。互动平台上，威海广泰表示，公司目前订单饱满，生产情况良好；爱旭股份表示，近期公司大尺寸电池订单充足，相关产线处于满产状态；玉马遮阳表示，公司产品的市场需求仍持续旺盛，在手订单3-5个月左右；中天科技表示，得益于电子铜箔市场需求增长，公司进一步投建产线；可孚医疗表示，公司新冠抗原家用自测检测试剂正在申请欧盟CE认证和美国FDA认证；铜冠铜箔表示，铜价上涨将导致铜箔价格相应增加，对公司营业收入的增加产生积极作用。",
			ContentGeneratedAt: time.Now(),
			SubSourceID:        "0e036627-5e7b-4a46-b46d-375502d0d69c",
			SubSource: model.SubSource{
				Name:      "快讯",
				Id:        "0e036627-5e7b-4a46-b46d-375502d0d69c",
				AvatarUrl: "https://newsfeed-logo.s3.us-west-1.amazonaws.com/cls.png",
			},
			ImageUrls:       []string{"https://d20uffqoe1h0vv.cloudfront.net/3f7fdb3b9673b3ab6a8d1f8c10adf4b0.png"},
			Tag:             "互动平台精选",
			SemanticHashing: "01101101011100010101101101010010010110000101101000111001010011100001100100010011000111011000011010111110000000111100000000101001",
		},
		{ // 1
			Id:                 "6ed7e272-d685-4c0d-aa6a-3026c23c3e95",
			Title:              "",
			Content:            "财联社1月28日电，欧元区1月工业景气指数13.9，预期15，前值14.9。",
			ContentGeneratedAt: time.Now(),
			SubSourceID:        "0e036627-5e7b-4a46-b46d-375502d0d69c",
			SubSource: model.SubSource{
				Name:      "快讯",
				Id:        "0e036627-5e7b-4a46-b46d-375502d0d69c",
				AvatarUrl: "https://newsfeed-logo.s3.us-west-1.amazonaws.com/cls.png",
			},
			ImageUrls:       []string{"https://d20uffqoe1h0vv.cloudfront.net/093f63165e533c77dd0c5b47f805669c.jpg"},
			Tag:             "环球市场情报",
			SemanticHashing: "10100110111010111000100110110111110010011101001000000101001000100011011111101110011001011101001001101101010101000000101000011011",
		},
		{ // 2
			Id:                 "294a136e-416f-4bf5-865e-9a26484e3c4f",
			Title:              "",
			Content:            "如果这样的风气不能被遏制，法律不能保护好人，和谐社会就是一句空话，敲诈勒索，寻衅滋事，这两条我看都符合。//@响马:操",
			ContentGeneratedAt: time.Now(),
			SubSourceID:        "c2bdedc6-671d-47f3-8505-ec3091808f29",
			SubSource: model.SubSource{
				Name:      "caoz",
				Id:        "c2bdedc6-671d-47f3-8505-ec3091808f29",
				AvatarUrl: "https://tvax2.sinaimg.cn/crop.0.0.664.664.180/591e78e3ly8gcfk75jah3j20ig0ihjsg.jpg?KID=imgbed,tva&Expires=1632388685&ssig=07eEdcDKlw",
			},
			ImageUrls: []string{},
			Tag:       "",
		},
		{ // 3
			Id:                 "087ceb96-7107-4244-aa04-0269c97cb9db",
			Title:              "",
			Content:            "Trump Responds, Criticizes Pence Over Comments Regarding Jan. 6 https://www.zerohedge.com/markets/trump-responds-criticizes-pence-over-comments-regarding-jan-6",
			ContentGeneratedAt: time.Now(),
			SubSourceID:        "df2eb2c5-4330-427a-8773-7f823014b4b3",
			SubSource: model.SubSource{
				Name:      "",
				Id:        "zerohedge",
				AvatarUrl: "https://d20uffqoe1h0vv.cloudfront.net/fff973f0bc718e9332549d45b277d49e.jpg",
			},
			ImageUrls: []string{},
			Tag:       "",
		},
		{ // 4
			Id:                 "d4f5d329-3200-4821-b0a7-6f4fd6610e56",
			Title:              "京东方有望为三星平价机型供应面板",
			Content:            "三星新款A13/A23智能手机平价机型将于5月上市，京东方正在为该机型开发面板。据业内人士透露，3月底三星已向京东方提出，由后者供应下一代旗舰智能手机面板，目前双方正在讨论技术验证及合同签署事项。 (BusinessKorea)",
			ContentGeneratedAt: time.Now(),
			SubSource: model.SubSource{
				Name:      "",
				Id:        "25fb05dc-102b-4521-b008-4623dc91a184",
				AvatarUrl: "https://newsfeed-logo.s3.us-west-1.amazonaws.com/kuailansi.png",
			},
			ImageUrls: []string{},
			Tag:       "",
		},
	}
	columns []model.Column = []model.Column{
		{ // 0
			Id:          "ColumnId-1",
			Name:        "Cunsumer Internet",
			CreatedAt:   time.Date(2022, 1, 25, 12, 30, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2022, 1, 25, 12, 30, 0, 0, time.UTC),
			CreatorID:   "CreatorID-1",
			Subscribers: users[0:3],
		},
		{ // 1
			Id:          "ColumnId-2",
			Name:        "半导体制造",
			CreatedAt:   time.Date(2022, 1, 25, 12, 30, 1, 0, time.UTC),
			UpdatedAt:   time.Date(2022, 1, 25, 12, 30, 1, 0, time.UTC),
			CreatorID:   "CreatorID-2",
			Subscribers: users[2:5],
		},
		{ // 2
			Id:          "ColumnId-3",
			Name:        "光伏",
			CreatedAt:   time.Date(2022, 1, 25, 12, 30, 2, 0, time.UTC),
			UpdatedAt:   time.Date(2022, 1, 25, 12, 30, 2, 0, time.UTC),
			CreatorID:   "CreatorID-3",
			Subscribers: users[5:6],
		},
		{ // 3
			Id:          "badcee9e-15ea-490c-8f08-be6b294fbdd6",
			Name:        "iOS测试用",
			CreatedAt:   time.Date(2022, 1, 25, 12, 30, 2, 0, time.UTC),
			UpdatedAt:   time.Date(2022, 1, 25, 12, 30, 2, 0, time.UTC),
			CreatorID:   "2ef1afbc-6ce3-4493-9a6b-e03e421d5066",
			Subscribers: users[:1],
		},
		{ // 4
			Id:          "81cddffa-c3d7-4df5-ab36-5e2ac14b8405",
			Name:        "twitter",
			CreatedAt:   time.Date(2022, 1, 25, 12, 30, 2, 0, time.UTC),
			UpdatedAt:   time.Date(2022, 1, 25, 12, 30, 2, 0, time.UTC),
			CreatorID:   "86e2de25-0279-4c7a-8d57-e4d4b91617e3",
			Subscribers: users[:1],
		},
		{ //5
			Id:          "fakeId",
			Name:        "快讯",
			CreatedAt:   time.Date(2022, 1, 25, 12, 30, 2, 0, time.UTC),
			UpdatedAt:   time.Date(2022, 1, 25, 12, 30, 2, 0, time.UTC),
			CreatorID:   "fakeId",
			Subscribers: users[:1],
		},
		{ // 6
			Id:          "dcd8c595-a18d-4134-ae16-3c970543a772",
			Name:        "kweb",
			CreatedAt:   time.Date(2022, 1, 25, 12, 30, 2, 0, time.UTC),
			UpdatedAt:   time.Date(2022, 1, 25, 12, 30, 2, 0, time.UTC),
			CreatorID:   "fakeId",
			Subscribers: users[:1],
		},
		{ // 7
			Id:          "e870fe51-6955-47bb-a0ac-61b14a010df2",
			Name:        "要闻",
			CreatedAt:   time.Date(2022, 1, 25, 12, 30, 2, 0, time.UTC),
			UpdatedAt:   time.Date(2022, 1, 25, 12, 30, 2, 0, time.UTC),
			CreatorID:   "fakeId",
			Subscribers: users[1:2],
		},
	}
	expectedNotificationOuputJobs []NotificationOutputJob = []NotificationOutputJob{
		{
			Title:       "【光伏】caoz",
			Subtitle:    "",
			Description: "如果这样的风气不能被遏制，法律不能保护好人，和谐社会就是一句空话，敲诈勒索，寻衅滋事，这两条我看都符合。//@响马:操",
			UserIds:     []string{"userId-6"},
			ColumnId:    "ColumnId-3",
		},
		{
			Title:       "【Cunsumer Internet】财联社1月28日互动平...",
			Subtitle:    "",
			Description: "今日大盘波动较大，沪指收跌0.97%，深成指跌0.53%，创业板指涨0.07%。旅游、教育、猪肉、三胎等板块涨幅居前。互动平台上，威海广泰表示，公司目前订单饱满，生产情况良好；爱旭股份表示，近期公司大尺寸电池订单充足，相关产线处于满产状态；...",
			UserIds:     []string{"userId-1", "userId-2"},
			ColumnId:    "ColumnId-1",
		},
		{
			Title:       "Cunsumer Intern...(1) 半导体制造(2)",
			Subtitle:    "",
			Description: "【caoz】如果这样的风气不能被遏制...\n【快讯】财联社1月28日互动平台精...\n【快讯】财联社1月28日电，欧元区1...",
			UserIds:     []string{"userId-3"},
			ColumnId:    "ColumnId-2", // could be some other columnId
		},
		{
			Title:       "半导体制造(2)",
			Subtitle:    "",
			Description: "【caoz】如果这样的风气不能被遏制...\n【快讯】财联社1月28日电，欧元区1...",
			UserIds:     []string{"userId-4", "userId-5"},
			ColumnId:    "ColumnId-2", // could be some other columnId
		},
	}
	notificationOutputJobs []NotificationOutputJob = []NotificationOutputJob{}
)

type mockNotificationConsumer struct {
	INotificationConsumer
}

func (m mockNotificationConsumer) PushNotification(job NotificationOutputJob) (*http.Response, error) {
	notificationOutputJobs = append(notificationOutputJobs, job)
	return nil, nil
}

func TestMain(m *testing.M) {
	dotenv.LoadDotEnvsInTests()
	os.Exit(m.Run())
}

func TestNotifier(t *testing.T) {
	t.Run("Test_notifier_should_stop", func(t *testing.T) {
		_, hook := test.NewNullLogger()
		mockNC := &mockNotificationConsumer{}
		notifier := NewNotifier(mockNC, TickerCycle, NotifierIntakeSize, NotifierOutputSize, PostDedupTTL)

		go notifier.Start()
		time.Sleep(NotifierProcessingWaitingTime)

		// check if ticker is ticking
		numOfLogs := 2 // +2: willProcessIntakeMsg, IntakeProcessingNoJobMsg
		time.Sleep(TickerCycle)
		require.Equal(t, numOfLogs, len(hook.Entries))

		notifier.Stop()
		numOfLogs += 1          // +1: Notifier stop request received...
		time.Sleep(TickerCycle) // notifier waits one more cycle then process the jobs
		time.Sleep(NotifierProcessingWaitingTime)
		numOfLogs += 3 // +3: Start processing intake jobs, no intake jobs, Notifier done
		require.Equal(t, numOfLogs, len(hook.Entries))

		// check if ticker stop ticking
		time.Sleep(2 * TickerCycle)
		require.Equal(t, numOfLogs, len(hook.Entries))
	})

	t.Run("Test_ticking_without_any_tasks", func(t *testing.T) {
		_, hook := test.NewNullLogger()
		mockNC := &mockNotificationConsumer{}
		notifier := NewNotifier(mockNC, TickerCycle, NotifierIntakeSize, NotifierOutputSize, PostDedupTTL)
		go notifier.Start()
		defer notifier.Stop()
		time.Sleep(NotifierProcessingWaitingTime)

		// test TestDurationCycles cycles
		for i := 1; i <= CycleCount; i++ {
			time.Sleep(TickerCycle)
			require.Equal(t, 2*i, len(hook.Entries))
			require.Equal(t, logrus.InfoLevel, hook.Entries[len(hook.Entries)-2].Level)
			require.Equal(t, willProcessIntakeMsg, hook.Entries[len(hook.Entries)-2].Message)
			require.Equal(t, logrus.InfoLevel, hook.LastEntry().Level)
			require.Equal(t, IntakeProcessingNoJobMsg, hook.LastEntry().Message)
		}
	})

	t.Run("Test_one_post_to_one_column", func(t *testing.T) {
		mockNC := &mockNotificationConsumer{}
		notifier := NewNotifier(mockNC, TickerCycle, NotifierIntakeSize, NotifierOutputSize, PostDedupTTL)
		notificationOutputJobs = []NotificationOutputJob{}
		require.Equal(t, 0, len(notificationOutputJobs))

		go notifier.Start()
		defer notifier.Stop()
		time.Sleep(NotifierProcessingWaitingTime)

		go notifier.AddIntakeJob(posts[0], columns[:1])
		time.Sleep(2 * TickerCycle)
		require.Equal(t, 1, len(notificationOutputJobs))
		require.Equal(t, "ColumnId-1", notificationOutputJobs[0].ColumnId)
		require.Equal(t, "【Cunsumer Internet】财联社1月28日互动平...", notificationOutputJobs[0].Title)
		require.Equal(t, "", notificationOutputJobs[0].Subtitle)
		require.Equal(t, "今日大盘波动较大，沪指收跌0.97%，深成指跌0.53%，创业板指涨0.07%。旅游、教育、猪肉、三胎等板块涨幅居前。互动平台上，威海广泰表示，公司目前订单饱满，生产情况良好；爱旭股份表示，近期公司大尺寸电池订单充足，相关产线处于满产状态；...", notificationOutputJobs[0].Description)
		require.Equal(t, 1, len(notificationOutputJobs[0].Images))
		require.Equal(t, "https://d20uffqoe1h0vv.cloudfront.net/3f7fdb3b9673b3ab6a8d1f8c10adf4b0.png", notificationOutputJobs[0].Images[0])
		require.Equal(t, 1, len(notificationOutputJobs[0].SubsourceAvatarUrls))
		require.Equal(t, "https://newsfeed-logo.s3.us-west-1.amazonaws.com/cls.png", notificationOutputJobs[0].SubsourceAvatarUrls[0])
		for _, user := range users {
			require.True(t, sort.SearchStrings(notificationOutputJobs[0].UserIds, user.Id) >= 0)
		}
	})

	t.Run("Test_one_post_to_one_column_subSourceNameSameAsColumnName", func(t *testing.T) {
		mockNC := &mockNotificationConsumer{}
		notifier := NewNotifier(mockNC, TickerCycle, NotifierIntakeSize, NotifierOutputSize, PostDedupTTL)
		notificationOutputJobs = []NotificationOutputJob{}
		require.Equal(t, 0, len(notificationOutputJobs))

		go notifier.Start()
		defer notifier.Stop()
		time.Sleep(NotifierProcessingWaitingTime)

		go notifier.AddIntakeJob(posts[0], columns[5:6])
		time.Sleep(2 * TickerCycle)
		require.Equal(t, 1, len(notificationOutputJobs))
		require.Equal(t, "fakeId", notificationOutputJobs[0].ColumnId)
		require.Equal(t, "【快讯】财联社1月28日互动平台精选", notificationOutputJobs[0].Title)
		require.Equal(t, "", notificationOutputJobs[0].Subtitle)
		require.Equal(t, "今日大盘波动较大，沪指收跌0.97%，深成指跌0.53%，创业板指涨0.07%。旅游、教育、猪肉、三胎等板块涨幅居前。互动平台上，威海广泰表示，公司目前订单饱满，生产情况良好；爱旭股份表示，近期公司大尺寸电池订单充足，相关产线处于满产状态；...", notificationOutputJobs[0].Description)
		require.Equal(t, 1, len(notificationOutputJobs[0].Images))
		require.Equal(t, "https://d20uffqoe1h0vv.cloudfront.net/3f7fdb3b9673b3ab6a8d1f8c10adf4b0.png", notificationOutputJobs[0].Images[0])
		require.Equal(t, 1, len(notificationOutputJobs[0].SubsourceAvatarUrls))
		require.Equal(t, "https://newsfeed-logo.s3.us-west-1.amazonaws.com/cls.png", notificationOutputJobs[0].SubsourceAvatarUrls[0])
		for _, user := range users {
			require.True(t, sort.SearchStrings(notificationOutputJobs[0].UserIds, user.Id) >= 0)
		}
	})

	t.Run("Test_multiple_posts_to_one_column", func(t *testing.T) {
		mockNC := &mockNotificationConsumer{}
		notifier := NewNotifier(mockNC, TickerCycle, NotifierIntakeSize, NotifierOutputSize, PostDedupTTL)
		notificationOutputJobs = []NotificationOutputJob{}
		require.Equal(t, 0, len(notificationOutputJobs))

		go notifier.Start()
		defer notifier.Stop()
		time.Sleep(NotifierProcessingWaitingTime)

		go notifier.AddIntakeJob(posts[0], columns[:1])
		go notifier.AddIntakeJob(posts[1], columns[:1])
		go notifier.AddIntakeJob(posts[2], columns[:1])

		time.Sleep(2 * TickerCycle)
		require.Equal(t, 1, len(notificationOutputJobs))
		require.Equal(t, "ColumnId-1", notificationOutputJobs[0].ColumnId)
		require.Equal(t, "Cunsumer Internet(3)", notificationOutputJobs[0].Title)
		require.Equal(t, "", notificationOutputJobs[0].Subtitle)
		require.True(t, len(notificationOutputJobs[0].Images) > 0)
		require.True(t, len(notificationOutputJobs[0].Images[0]) > 0)
		expectedDescriptionMap := map[string]bool{
			"【caoz】如果这样的风气不能被遏制...\n【快讯】财联社1月28日电，欧元区1...\n【快讯】财联社1月28日互动平台精...":             true,
			"【caoz】如果这样的风气不能被遏制...\n【快讯】财联社1月28日互动平台精...\n【快讯】财联社1月28日电，欧元区1...":             true,
			"【快讯】财联社1月28日互动平台精...\n【caoz】如果这样的风气不能被遏制...\n【快讯】财联社1月28日电，欧元区1...":             true,
			"【快讯】财联社1月28日互动平台精...\n【快讯】财联社1月28日电，欧元区1...\n【caoz】如果这样的风气不能被遏制...":             true,
			"【快讯】财联社1月28日电，欧元区1...\n【快讯】财联社1月28日互动平台精选,今日大盘波动较大，沪...\n【caoz】如果这样的风气不能被遏制...": true,
			"【快讯】财联社1月28日电，欧元区1...\n【caoz】如果这样的风气不能被遏制...\n【快讯】财联社1月28日互动平台精...":             true,
		}
		require.Equal(t, 2, len(notificationOutputJobs[0].SubsourceAvatarUrls))
		expectedAvatarUrls := map[string]bool{
			posts[0].SubSource.AvatarUrl: true, // same as posts[1]'s
			posts[2].SubSource.AvatarUrl: true,
		}
		for _, avatarUrl := range notificationOutputJobs[0].SubsourceAvatarUrls {
			require.True(t, expectedAvatarUrls[avatarUrl])
		}
		// require.Equal(t, "", notificationOutputJobs[0].Description) // for debugging
		require.True(t, expectedDescriptionMap[notificationOutputJobs[0].Description])
		for _, user := range users {
			require.True(t, sort.SearchStrings(notificationOutputJobs[0].UserIds, user.Id) >= 0)
		}
	})

	t.Run("Test_one_post_to_one_column_for_2_cycles", func(t *testing.T) {
		mockNC := &mockNotificationConsumer{}
		notifier := NewNotifier(mockNC, TickerCycle, NotifierIntakeSize, NotifierOutputSize, PostDedupTTL)
		notificationOutputJobs = []NotificationOutputJob{}
		require.Equal(t, 0, len(notificationOutputJobs))

		go notifier.Start()
		defer notifier.Stop()
		time.Sleep(NotifierProcessingWaitingTime)

		go notifier.AddIntakeJob(posts[0], columns[:1])
		time.Sleep(2 * TickerCycle)
		require.Equal(t, 1, len(notificationOutputJobs))
		require.Equal(t, "ColumnId-1", notificationOutputJobs[0].ColumnId)
		require.Equal(t, "【Cunsumer Internet】财联社1月28日互动平...", notificationOutputJobs[0].Title)
		require.Equal(t, "", notificationOutputJobs[0].Subtitle)
		require.Equal(t, "今日大盘波动较大，沪指收跌0.97%，深成指跌0.53%，创业板指涨0.07%。旅游、教育、猪肉、三胎等板块涨幅居前。互动平台上，威海广泰表示，公司目前订单饱满，生产情况良好；爱旭股份表示，近期公司大尺寸电池订单充足，相关产线处于满产状态；...", notificationOutputJobs[0].Description)
		require.True(t, len(notificationOutputJobs[0].Images) > 0)
		require.True(t, len(notificationOutputJobs[0].Images[0]) > 0)
		require.Equal(t, 1, len(notificationOutputJobs[0].SubsourceAvatarUrls))
		require.Equal(t, posts[0].SubSource.AvatarUrl, notificationOutputJobs[0].SubsourceAvatarUrls[0])

		for _, user := range users {
			require.True(t, sort.SearchStrings(notificationOutputJobs[0].UserIds, user.Id) >= 0)
		}

		go notifier.AddIntakeJob(posts[1], columns[:1])
		time.Sleep(2 * TickerCycle)
		require.Equal(t, 2, len(notificationOutputJobs))
		require.Equal(t, "ColumnId-1", notificationOutputJobs[1].ColumnId)
		require.Equal(t, "【Cunsumer Internet】快讯", notificationOutputJobs[1].Title)
		require.Equal(t, "", notificationOutputJobs[1].Subtitle)
		require.Equal(t, "财联社1月28日电，欧元区1月工业景气指数13.9，预期15，前值14.9。", notificationOutputJobs[1].Description)
		for _, user := range users {
			require.True(t, sort.SearchStrings(notificationOutputJobs[1].UserIds, user.Id) >= 0)
		}
	})

	// this test sometimes fails, need to investigate
	t.Run("Test_multiple_posts_to_multiple_columns", func(t *testing.T) {
		mockNC := &mockNotificationConsumer{}
		notifier := NewNotifier(mockNC, TickerCycle, NotifierIntakeSize, NotifierOutputSize, PostDedupTTL)
		notificationOutputJobs = []NotificationOutputJob{}
		require.Equal(t, 0, len(notificationOutputJobs))

		go notifier.Start()
		defer notifier.Stop()
		time.Sleep(NotifierProcessingWaitingTime)

		go notifier.AddIntakeJob(posts[0], columns[0:1])
		go notifier.AddIntakeJob(posts[1], columns[1:2])
		go notifier.AddIntakeJob(posts[2], columns[1:3])

		// posts[0] -> users[0:3] -> userId-1. userId-2 userId-3
		// posts[1] -> users[2:5] ->  userId-3 userId-4 userId-5
		// posts[2] -> users[2:5]+ users[5:6] -> userId-3 userId-4 userId-5 userId-6

		// userId-1, userId-2 -> posts[0] -> notificcation 1
		// userId-3 -> posts[0], posts[1], posts[2] -> notificcation 2
		// userId-4, userId-5 -> posts[1], posts[2] -> notificcation 3
		// userId-6 -> posts[2] -> -> notificcation 4

		time.Sleep(2 * TickerCycle)
		require.Equal(t, 4, len(notificationOutputJobs))
		// columnId check if skipped in containersAllNotifications
		// require.Equal(t, "", notificationOutputJobs[0].Description+notificationOutputJobs[1].Description+notificationOutputJobs[2].Description+notificationOutputJobs[3].Description) // for debugging purpose
		require.True(t, containersAllNotifications(expectedNotificationOuputJobs, notificationOutputJobs))
	})

	t.Run("Test_multiple_posts_to_same_columns", func(t *testing.T) {
		mockNC := &mockNotificationConsumer{}
		notifier := NewNotifier(mockNC, TickerCycle, NotifierIntakeSize, NotifierOutputSize, PostDedupTTL)
		notificationOutputJobs = []NotificationOutputJob{}
		require.Equal(t, 0, len(notificationOutputJobs))

		go notifier.Start()
		defer notifier.Stop()
		time.Sleep(NotifierProcessingWaitingTime)

		go notifier.AddIntakeJob(posts[0], columns[0:1])
		go notifier.AddIntakeJob(posts[1], columns[0:1])
		go notifier.AddIntakeJob(posts[2], columns[0:1])

		// posts[0],post[1],post[2] -> users[0:3] -> userId-1. userId-2 userId-3

		time.Sleep(2 * TickerCycle)
		require.Equal(t, 1, len(notificationOutputJobs))
		require.Equal(t, 2, len(notificationOutputJobs[0].Images))
		require.Equal(t, 2, len(notificationOutputJobs[0].SubsourceAvatarUrls))
		expectedAvatarUrls := map[string]bool{
			posts[0].SubSource.AvatarUrl: true, // same as posts[1]'s
			posts[2].SubSource.AvatarUrl: true,
		}
		for _, avatarUrl := range notificationOutputJobs[0].SubsourceAvatarUrls {
			require.True(t, expectedAvatarUrls[avatarUrl])
		}
	})

	t.Run("Test_multiple_same_post_with_different_columns", func(t *testing.T) {
		mockNC := &mockNotificationConsumer{}
		notifier := NewNotifier(mockNC, TickerCycle, NotifierIntakeSize, NotifierOutputSize, PostDedupTTL)
		notificationOutputJobs = []NotificationOutputJob{}
		require.Equal(t, 0, len(notificationOutputJobs))

		go notifier.Start()
		defer notifier.Stop()
		time.Sleep(NotifierProcessingWaitingTime)

		go notifier.AddIntakeJob(posts[3], columns[3:4])
		go notifier.AddIntakeJob(posts[3], columns[4:5])
		time.Sleep(2 * TickerCycle)
		require.Equal(t, 1, len(notificationOutputJobs))
		expectedTitleMap := map[string]bool{
			"【twitter】": true,
			"【iOS测试用】":  true,
		}
		require.Equal(t, true, expectedTitleMap[notificationOutputJobs[0].Title])
		require.Equal(t, "", notificationOutputJobs[0].Subtitle)
		expectedColumnIds := map[string]bool{}
		expectedColumnIds[columns[3].Id] = true
		expectedColumnIds[columns[4].Id] = true
		require.Equal(t, true, expectedColumnIds[notificationOutputJobs[0].ColumnId])
		require.Equal(t, "Trump Responds, Criticizes Pence Over Comments Regarding Jan. 6 https://www.zerohedge.com/markets/trump-responds-critici...", notificationOutputJobs[0].Description)
		require.Equal(t, 0, len(notificationOutputJobs[0].Images))
		require.Equal(t, 1, len(notificationOutputJobs[0].SubsourceAvatarUrls))
		require.Equal(t, posts[3].SubSource.AvatarUrl, notificationOutputJobs[0].SubsourceAvatarUrls[0])
	})

	t.Run("Test_add_intake_job_once_notifer_starts_to_process_intake_jobs", func(t *testing.T) {
		mockNC := &mockNotificationConsumer{}
		notifier := NewNotifier(mockNC, TickerCycle, NotifierIntakeSize, NotifierOutputSize, PostDedupTTL)
		notificationOutputJobs = []NotificationOutputJob{}
		require.Equal(t, 0, len(notificationOutputJobs))

		go notifier.Start()
		defer notifier.Stop()

		go notifier.AddIntakeJob(posts[0], columns[:1])
		time.Sleep(1 * TickerCycle)

		// uncomment following will trigger the case of adding intakeJob to closed channel
		// "panic: send on closed channel"
		// for i := 0; i < 1000; i++ {
		// 	go notifier.AddIntakeJob(posts[1], columns[:1])
		// }
	})

	t.Run("Test_similar_posts", func(t *testing.T) {
		mockNC := &mockNotificationConsumer{}
		notifier := NewNotifier(mockNC, TickerCycle, NotifierIntakeSize, NotifierOutputSize, PostDedupTTL)
		notificationOutputJobs = []NotificationOutputJob{}

		go notifier.Start()
		defer notifier.Stop()

		go notifier.AddIntakeJob(similarPosts[0], columns[0:1])

		time.Sleep(2 * TickerCycle)
		require.Equal(t, 1, len(notificationOutputJobs))

		// add similar post to same column
		go notifier.AddIntakeJob(similarPosts[1], columns[0:1])

		time.Sleep(2 * TickerCycle)
		require.Equal(t, 1, len(notificationOutputJobs))
		require.Equal(t, 0, len(notificationOutputJobs[0].Images))
		require.Equal(t, 0, len(notificationOutputJobs[0].SubsourceAvatarUrls))
	})

	t.Run("Test_nofitier_should_have_correct_columnname", func(t *testing.T) {
		mockNC := &mockNotificationConsumer{}
		notifier := NewNotifier(mockNC, TickerCycle, NotifierIntakeSize, NotifierOutputSize, PostDedupTTL)
		notificationOutputJobs = []NotificationOutputJob{}
		require.Equal(t, 0, len(notificationOutputJobs))

		go notifier.Start()
		defer notifier.Stop()
		time.Sleep(NotifierProcessingWaitingTime)

		go notifier.AddIntakeJob(posts[4], columns[6:8])

		time.Sleep(2 * TickerCycle)
		require.Equal(t, 2, len(notificationOutputJobs))

		for _, job := range notificationOutputJobs {
			if job.ColumnId == columns[7].Id {
				require.Equal(t, "【要闻】京东方有望为三星平价机型供应面板", job.Title)
				require.Equal(t, "", job.Subtitle)
				require.Equal(t, columns[7].Subscribers[0].Id, job.UserIds[0])
			}
			if job.ColumnId == columns[6].Id {
				require.Equal(t, "【kweb】京东方有望为三星平价机型供应面板", job.Title)
				require.Equal(t, "", job.Subtitle)
				require.Equal(t, columns[6].Subscribers[0].Id, job.UserIds[0])
			}
		}
	})
}

func containersAllNotifications(expected []NotificationOutputJob, actual []NotificationOutputJob) bool {
	if len(expected) != len(actual) {
		return false
	}
	expectedSerialized := []string{}
	for _, job := range expected {
		expectedSerialized = append(expectedSerialized, serializeNotificationOutputJob(job))
	}

	actualSerialized := []string{}
	for _, job := range actual {
		actualSerialized = append(actualSerialized, serializeNotificationOutputJob(job))
	}

	for _, serialized := range actualSerialized {
		if !contains(expectedSerialized, serialized) {
			return false
		}
	}

	for _, serialized := range expectedSerialized {
		if !contains(actualSerialized, serialized) {
			return false
		}
	}

	return true
}

func serializeNotificationOutputJob(job NotificationOutputJob) string {
	title := job.Title
	subtitle := job.Subtitle
	userIds := job.UserIds
	description := job.Description
	// due to multiple posts case columnId could be any of them, remove columnId check
	// columnId := job.ColumnId
	sort.Strings(userIds)
	usersStr := ""
	for _, userId := range userIds {
		usersStr += userId
	}
	return title + subtitle + description + usersStr // + columnId
}

// to do move to some centralized helper or util folder
func contains(s []string, str string) bool {
	for _, v := range s {
		vRuneMap := getRuneCountMap(v)
		strRuneMap := getRuneCountMap(str)
		if len(vRuneMap) == len(strRuneMap) {
			for r := range vRuneMap {
				if vRuneMap[r] != strRuneMap[r] {
					return false
				}
			}
			return true
		}
	}

	return false
}

// generate a key based on character and its appearance, position doesn't matter
func getRuneCountMap(s string) map[rune]int {
	m := map[rune]int{}
	sRune := []rune(s)
	for i := 0; i < len(sRune); i++ {
		m[sRune[i]] += 1
	}
	return m
}
