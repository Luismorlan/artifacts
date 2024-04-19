package notifier

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rnr-capital/newsfeed-backend/model"
	Util "github.com/rnr-capital/newsfeed-backend/utils"
	Logger "github.com/rnr-capital/newsfeed-backend/utils/log"
)

// receive intakeJob with a post and corresponding columns into queue
// process intakeJobs in queue every time period and aggregate them
// into output notification jobs with title, description and usersId/deviceIds
// 1. aggregate posts in a period to users
// 2. group users by exact posts into noficiations, each notification contains posts and users
// 3. process each notification and call external notification API

const (
	TitleMaxLen                  = 30
	SubTitleMaxLen               = 25
	DescriptionMaxLen            = 120
	OneLineLen                   = 25
	DescriptionContentItemMaxLen = 10
	ColumnnameMinRuneLen         = 5
	PostMaxLinesInDescription    = 4
	IntakeQueueBufferSize        = 200
	OutputQueueBufferSize        = 200
	// Hamming distance for 2 posts to be considered as semantically identical.
	// For 2 x 128 bit hashing, if with maximal entrophy, the chance of hamming
	// distance < 37 is C(128, 91)/(2^128) < 0.0001% which is pretty safe.
	POST_SIMILARITY_THRESHOLD = 37 // same as cient side
	willProcessIntakeMsg      = "Will process intake jobs"
	ProcessingIntakeMsg       = "Processing intake jobs"
	ProcessingOutputMsg       = "Processing output jobs"
	IntakeProcessingNoJobMsg  = "no intake jobs"
	OutputProcessingMsg       = "Start processing output jobs"
)

var (
	Log = Logger.LogV2
)

type INotificationConsumer interface {
	PushNotification(job NotificationOutputJob) (*http.Response, error)
}

type NotificationConsumer struct {
	INotificationConsumer
}

type Notifier struct {
	notificationConsumer INotificationConsumer
	intakeJobs           chan NotificationIntakeJob
	outputJobs           chan NotificationOutputJob
	deduplicator         NotifierDeduplicator
	ticker               time.Ticker
	tickerCycle          time.Duration
	ctx                  context.Context
	cancel               context.CancelFunc
}

type NotificationOutputJob struct {
	Title               string
	Subtitle            string
	Description         string
	UserIds             []string
	ColumnId            string
	Images              []string
	SubsourceAvatarUrls []string
}

type NotificationIntakeJob struct {
	post    model.Post
	columns []model.Column
}

type PostWithColumn struct {
	posdId   string
	columnId string
}

func NewNotifier(nc INotificationConsumer, d time.Duration, intakeSize int, outputSize int, postDedupTTL time.Duration) *Notifier {
	ctx, cancel := context.WithCancel(context.Background())
	return &Notifier{
		notificationConsumer: nc,
		intakeJobs:           make(chan NotificationIntakeJob, IntakeQueueBufferSize),
		outputJobs:           make(chan NotificationOutputJob, OutputQueueBufferSize),
		deduplicator:         *NewNotifierDeduplicator(postDedupTTL, POST_SIMILARITY_THRESHOLD),
		ticker:               *time.NewTicker(d),
		tickerCycle:          d,
		ctx:                  ctx,
		cancel:               cancel,
	}
}

// Start receiving intake jobs
func (n *Notifier) Start() {
	for {
		select {
		case <-n.ctx.Done():
			Log.Info("Notifier done")
			// need to move all intakeJobs to ouput jobs and after queue is clear, close them?
			// close(n.intakeJobs)
			// close(n.outputJobs)
			return
		case <-n.ticker.C:
			Log.Info(willProcessIntakeMsg)
			n.processIntakeJobs()
		}
	}
}

// Wait a one cycle to process existing jobs in queue and will stop
func (n *Notifier) Stop() {
	Log.Info("Notifier stop request received, will stop notifier after " + n.tickerCycle.String())
	time.Sleep(n.tickerCycle)
	n.cancel()
}

func (n *Notifier) AddIntakeJob(post model.Post, columns []model.Column) {
	intakeJob := NotificationIntakeJob{
		post:    post,
		columns: columns,
	}
	n.intakeJobs <- intakeJob
}

func (n *Notifier) reCreateIntakeChannel() {
	Log.Info("re-creating intake channel")
	n.intakeJobs = make(chan NotificationIntakeJob, IntakeQueueBufferSize)
}

func (n *Notifier) reCreateOutputChannel() {
	Log.Info("re-creating output channel")
	n.outputJobs = make(chan NotificationOutputJob, OutputQueueBufferSize)
}

// processIntakeJobs serializely process each job in intakeJob chan
// and generate output notification jobs
func (n *Notifier) processIntakeJobs() {
	if len(n.intakeJobs) == 0 {
		Log.Info(IntakeProcessingNoJobMsg)
		return
	}
	Log.Infof(ProcessingIntakeMsg, len(n.intakeJobs))
	close(n.intakeJobs)

	userIdtoUserMap := map[string]model.User{}
	postIdtoPostMap := map[string]model.Post{}
	columnIdtoColumnMap := map[string]model.Column{}
	userIdtoPostsWithColumnmap := map[string][]PostWithColumn{}

	postIdtoPostIdxMap := map[string]int{}
	columnIdtoColumnIdxMap := map[string]int{}
	// record user's array of post and column pair
	// key is contructed like: post1.id,column2.id;post3.id,column1.d;...
	postsWithColumnsKeyToUsersMap := map[string][]model.User{}
	postIdx := 0
	columnIdx := 0

	// constructing
	for intakeJob := range n.intakeJobs {
		// a map to track if a user already showed because one column associate multiple
		// columns which could have the same user
		userShowedMap := map[string]bool{}
		postId := intakeJob.post.Id
		if _, ok := postIdtoPostIdxMap[postId]; !ok { // use the first appearance
			postIdtoPostIdxMap[postId] = postIdx
		}
		postIdtoPostMap[postId] = intakeJob.post
		for _, column := range intakeJob.columns {
			columnId := column.Id
			if _, ok := columnIdtoColumnIdxMap[columnId]; !ok { // use the first appearance
				columnIdtoColumnIdxMap[columnId] = columnIdx
			}
			columnIdtoColumnMap[columnId] = column
			for _, user := range column.Subscribers {
				// check if user had similar post over a past period
				hadSimilar := n.deduplicator.UserHadSimilarPost(user.Id, intakeJob.post)
				if hadSimilar {
					continue
				}

				userId := user.Id
				userIdtoUserMap[userId] = *user
				if _, ok := userIdtoPostsWithColumnmap[userId]; !ok {
					userIdtoPostsWithColumnmap[userId] = []PostWithColumn{}
				}
				userExistedForPost := userShowedMap[userId]
				if userExistedForPost {
					// need to handle which column to save for the the user, here we do nothing which
					// means we are going to use the first column
				} else {
					// user was processed with current post, record post and column to this user
					userIdtoPostsWithColumnmap[userId] = append(userIdtoPostsWithColumnmap[userId], PostWithColumn{
						posdId:   postId,
						columnId: columnId,
					})
				}
			}
			columnIdx++
		}
		postIdx++
	}

	// calculate each user's postsWithColumn key and aggregate users with same notification
	for userId, postsWithColumn := range userIdtoPostsWithColumnmap {
		postsWithColumnKey := ""
		for _, postWithKey := range postsWithColumn {
			userPostId := postWithKey.posdId
			userPostColumnId := postWithKey.columnId
			userPostIdx := postIdtoPostIdxMap[userPostId]
			userPostColumnIdx := columnIdtoColumnIdxMap[userPostColumnId]
			postsWithColumnKey += fmt.Sprintf("%d,%d;", userPostIdx, userPostColumnIdx)
		}
		if _, ok := postsWithColumnsKeyToUsersMap[postsWithColumnKey]; !ok {
			postsWithColumnsKeyToUsersMap[postsWithColumnKey] = []model.User{}
		}
		postsWithColumnsKeyToUsersMap[postsWithColumnKey] = append(postsWithColumnsKeyToUsersMap[postsWithColumnKey], userIdtoUserMap[userId])
	}

	// generate notification
	for _, notificationUsers := range postsWithColumnsKeyToUsersMap {
		notificationPostsWithColumn := userIdtoPostsWithColumnmap[notificationUsers[0].Id]
		ouputJob, error := newNotificationOutputJob(notificationUsers, notificationPostsWithColumn, postIdtoPostMap, columnIdtoColumnMap)
		if error != nil {
			Log.Error("failed to generate NotificationOutputJob")
			continue
		}
		n.outputJobs <- ouputJob
	}
	n.reCreateIntakeChannel()
	n.processOutputJobs()
	n.deduplicator.CleanExpiredPosts()
}

func newNotificationOutputJob(users []model.User, postsWithColumn []PostWithColumn, postIdtoPostMap map[string]model.Post, columnIdtoColumnMap map[string]model.Column) (NotificationOutputJob, error) {
	// deduplicate posts with same id
	postsWithColumn = dedupPostsWithColumn(postsWithColumn)

	title := ""
	description := ""
	subTitle := ""
	columnId := ""
	userIds := []string{}
	PostsLen := len(postsWithColumn)
	UsersLen := len(users)
	if PostsLen == 0 || UsersLen == 0 {
		error := errors.New("invalid size of users")
		Log.Errorf(error)
		return NotificationOutputJob{}, error
	}

	if PostsLen == 1 {
		// == Single post case ==
		// title=[columnName]postTitle or [columnName]subsourceName: "", description = Post.content
		post := postIdtoPostMap[postsWithColumn[0].posdId]
		columnId = postsWithColumn[0].columnId
		column := columnIdtoColumnMap[columnId]
		title = fmt.Sprintf("【%s】%s", column.Name, post.Title)
		if len(post.Title) == 0 && len(post.SubSource.Name) > 0 && post.SubSource.Name != column.Name {
			title = title + post.SubSource.Name
		}
		description = post.Content
	} else {
		// == Multiple posts case ==
		// title=ColumnName1[:4](2), ColumnName2(4)
		// subtitle: ""
		// description=[SubsourceName1]Post1.title||content[:15]...[SubsourceName2]Post2.title||content[:15]...
		// for maximum 4(@PostMaxLinesInDescription) posts
		columnId = postsWithColumn[0].columnId
		title = fmtColumns(postsWithColumn, columnIdtoColumnMap, TitleMaxLen)
		subsourceNamesMap := map[string]bool{} // subsourcename -> true
		for idx, postWithColumn := range postsWithColumn {
			post := postIdtoPostMap[postWithColumn.posdId]
			subsourceNamesMap[post.SubSource.Name] = true
			if idx >= PostMaxLinesInDescription {
				continue
			}
			postDescription := ""
			if len(post.SubSource.Name) > 0 {
				postDescription += "【" + post.SubSource.Name + "】"
			}
			if len(post.Title) > 0 {
				postDescription += post.Title + ":"
			}
			postDescription += post.Content
			postDescription = removeNewLine(postDescription)
			postDescriptinOneline := Util.GetOneline(postDescription, true)

			description += postDescriptinOneline
			// add new line at the end if it's not the last line
			if idx != len(postsWithColumn)-1 {
				description += string(rune('\n'))
			}
		}
	}

	// normailization for title and description
	title, subTitle, description = normalizeNotification(title, subTitle, description)

	// get userIds
	for _, user := range users {
		userIds = append(userIds, user.Id)
	}

	// attachment images from post
	images := []string{}
	// subsource image urls
	subsourceAvatarUrls := []string{}
	// url set for dedup
	existingUrls := map[string]bool{}
	for _, postWithColumn := range postsWithColumn {
		// add posts' image urls
		postImageUrls := postIdtoPostMap[postWithColumn.posdId].ImageUrls
		for _, postImageUrl := range postImageUrls {
			if !existingUrls[postImageUrl] {
				images = append(images, postImageUrl)
				existingUrls[postImageUrl] = true
			}
		}

		// add posts' subsources avatar urls
		postSubsourceAvatarUrl := postIdtoPostMap[postWithColumn.posdId].SubSource.AvatarUrl
		if len(postSubsourceAvatarUrl) > 0 {
			if !existingUrls[postSubsourceAvatarUrl] {
				subsourceAvatarUrls = append(subsourceAvatarUrls, postSubsourceAvatarUrl)
				existingUrls[postSubsourceAvatarUrl] = true
			}
		}
	}

	return NotificationOutputJob{
		Title:               title,
		Subtitle:            subTitle,
		Description:         description,
		UserIds:             userIds,
		ColumnId:            columnId,
		Images:              images,
		SubsourceAvatarUrls: subsourceAvatarUrls,
	}, nil
}

// processOutputJobs processes all output jobs concurrently
// and call external notification API
func (n *Notifier) processOutputJobs() {
	Log.Infof(ProcessingOutputMsg, ": ", len(n.outputJobs))
	c := make(chan int, len(n.outputJobs))
	close(n.outputJobs)
	for outputJob := range n.outputJobs {
		outputJob := outputJob
		go func() {
			Log.Infof("call external notification API, Title: ", outputJob.Title, ", Subtitle: ", outputJob.Subtitle, ", Description: ", outputJob.Description, ", ColumnId: ", outputJob.ColumnId, ", UserIds:", outputJob.UserIds)
			n.notificationConsumer.PushNotification(outputJob)
			c <- 1
		}()
	}
	for i := 0; i < len(n.outputJobs); i++ {
		<-c
	}
	n.reCreateOutputChannel()
}

// return: ColumnName1[:4](2) ColumnName2[:4](4)
func fmtColumns(postsWithColumn []PostWithColumn, columnIdtoColumnMap map[string]model.Column, runeSize int) string {
	columnIdCountMap := map[string]int{}
	for _, postWithColumn := range postsWithColumn {
		if _, ok := columnIdCountMap[postWithColumn.columnId]; !ok {
			columnIdCountMap[postWithColumn.columnId] = 1
		} else {
			columnIdCountMap[postWithColumn.columnId] += 1
		}
	}

	res := ""
	targetColumnNameSize := runeSize / len(columnIdCountMap)
	if targetColumnNameSize < ColumnnameMinRuneLen {
		targetColumnNameSize = ColumnnameMinRuneLen
	}
	for columnId, count := range columnIdCountMap {
		columnName := columnIdtoColumnMap[columnId].Name
		columnNameRunes := []rune(columnName)
		if len(columnNameRunes) > targetColumnNameSize {
			columnNameRunes = columnNameRunes[:targetColumnNameSize]
			columnNameRunes = append(columnNameRunes, rune('.'))
			columnNameRunes = append(columnNameRunes, rune('.'))
			columnNameRunes = append(columnNameRunes, rune('.'))
			columnName = string(columnNameRunes)
		}
		res += fmt.Sprintf("%s(%v) ", columnName, count)
	}
	if len(res) > 0 {
		res = res[:len(res)-1] // remove last space
	}
	return res
}

func normalizeNotification(title string, subTitle string, description string) (string, string, string) {
	titleRunes := []rune(title)
	descriptionRunes := []rune(description)
	subTitleRunes := []rune(subTitle)
	if len(titleRunes) > TitleMaxLen {
		title = fmt.Sprintf("%s...", string(titleRunes[:TitleMaxLen]))
	}
	if len(descriptionRunes) > DescriptionMaxLen {
		description = fmt.Sprintf("%s...", string(descriptionRunes[:DescriptionMaxLen]))
	}
	if len(subTitleRunes) > SubTitleMaxLen {
		subTitle = fmt.Sprintf("%s...", string(subTitleRunes[:SubTitleMaxLen]))
	}

	return title, subTitle, description
}

func dedupPostsWithColumn(postsWithColumn []PostWithColumn) []PostWithColumn {
	postIdMap := map[string]bool{}
	res := []PostWithColumn{}
	for _, postWithColumn := range postsWithColumn {
		if _, ok := postIdMap[postWithColumn.posdId]; !ok {
			// only keep the first post for posts with same post id, in future we can have
			// have some selection logic based on other conditions e.g. Column created time.
			postIdMap[postWithColumn.posdId] = true
			res = append(res, postWithColumn)
		}
	}
	return res
}

func removeNewLine(s string) string {
	return strings.ReplaceAll(s, "\n", "") // "\r", "\t" ?
}
