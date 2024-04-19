package notifier

import (
	"testing"
	"time"

	"github.com/pgvector/pgvector-go"
	"github.com/rnr-capital/newsfeed-backend/model"
	"github.com/stretchr/testify/require"
)

const (
	PostTTL                 = time.Second * 2
	PostSimilarityThreshold = 37
)

var (
	emb1                      = pgvector.NewVector([]float32{-0.21385951, -0.020870402, 0.053375296, 0.09230549, 0.0062988596, -0.011883218, 0.017824506, 0.062477097, 0.053160008, -0.054524288, 0.012885212, -0.056548327, 0.029230237, -0.005145558, -0.01425094, 0.0654019, 0.0006420099, 0.05903293, 0.006494866, 0.012895132, 0.04294406, 0.053717993, -0.041144352, 0.009911478, 0.03299959, -0.040267088, 0.03122021, -0.0078089326, -0.028805027, 0.02506149, -0.011974913, -0.05789814, -0.02042555, 0.019506209, 0.007936628, 0.110947944, -0.023690417, -0.07119492, 0.0058847107, -0.062044285, 0.019535935, -0.005885733, -0.017841052, -0.005167019, 0.0059522414, 0.012231415, 0.03151156, -0.018827565, -0.022419352, 0.023690322, 0.030699886, -0.011314588, -0.003176537, -0.023127161, 0.020406388, -0.0015781109, 0.0049422025, 0.027282275, 0.0013280667, 0.035343155, 0.014749111, 0.0042124903, 0.027489642, -0.021498341, 0.0014124477, 0.0136264665, -0.008606255, 0.04252986, -0.0061811386, 0.014591267, -0.041568667, -0.025240771, -0.002085906, 0.012288577, -0.041729372, 0.0069256052, -0.007705723, 0.007149281, -0.005028577, 0.016214475, 0.009357461, -0.011778682, 0.016736422, 0.0143109625, 0.019054774, 0.0197832, 0.010398859, 0.0016498897, -0.003941197, -0.030713508, -0.02982454, 0.039167713, -0.006404586, 0.00023291662, -0.009192385, 0.00040785133, 0.027513323, 0.021163044, -0.017019186, 0.0022130648})
	emb2                      = pgvector.NewVector([]float32{-0.21385951, -0.020870402, 0.053375296, 0.09230549, 0.0062988596, -0.011883218, 0.017824506, 0.062477097, 0.053160008, -0.054524288, 0.012885212, -0.056548327, 0.029230237, -0.005145558, -0.01425094, 0.0654019, 0.0006420099, 0.05903293, 0.006494866, 0.012895132, 0.04294406, 0.053717993, -0.041144352, 0.009911478, 0.03299959, -0.040267088, 0.03122021, -0.0078089326, -0.028805027, 0.02506149, -0.011974913, -0.05789814, -0.02042555, 0.019506209, 0.007936628, 0.110947944, -0.023690417, -0.07119492, 0.0058847107, -0.062044285, 0.019535935, -0.005885733, -0.017841052, -0.005167019, 0.0059522414, 0.012231415, 0.03151156, -0.018827565, -0.022419352, 0.023690322, 0.030699886, -0.011314588, -0.003176537, -0.023127161, 0.020406388, -0.0015781109, 0.0049422025, 0.027282275, 0.0013280667, 0.035343155, 0.014749111, 0.0042124903, 0.027489642, -0.021498341, 0.0014124477, 0.0136264665, -0.008606255, 0.04252986, -0.0061811386, 0.014591267, -0.041568667, -0.025240771, -0.002085906, 0.012288577, -0.041729372, 0.0069256052, -0.007705723, 0.007149281, -0.005028577, 0.016214475, 0.009357461, -0.011778682, 0.016736422, 0.0143109625, 0.019054774, 0.0197832, 0.010398859, 0.0016498897, -0.003941197, -0.030713508, -0.02982454, 0.039167713, -0.006404586, 0.00023291662, -0.009192385, 0.00040785133, 0.027513323, 0.021163044, -0.017019186, 0.0022130648})
	similarPosts []model.Post = []model.Post{
		{
			Id:                 "95dac8be-c1d5-4d69-8fdb-602c3053fcb5",
			Title:              "",
			Content:            "法国欧盟事务部长Beaune：我们需要对俄罗斯表现出坚定的态度，这样袭击才会停止，现在评估扎波罗热核电站的情况还为时过早，但我们可以看到袭击行动正在加剧。",
			ContentGeneratedAt: time.Now(),
			Embedding:          &emb1,
		},
		{
			Id:                 "23a300fb-1206-4735-9b1c-721d4aa63b57",
			Title:              "",
			Content:            "法国欧盟事务部长Beaune：俄罗斯最近的袭击非常令人担忧，情况很严重。",
			ContentGeneratedAt: time.Now(),
			Embedding:          &emb2,
		},
	}
)

func TestNotifierDeduplicator(t *testing.T) {

	t.Run("Test_initialization", func(t *testing.T) {
		notifierDeduplicator := NewNotifierDeduplicator(PostTTL, PostSimilarityThreshold)
		require.Equal(t, 0, len(notifierDeduplicator.UserIdToPosts))
		require.Equal(t, PostTTL, notifierDeduplicator.PostTTL)
		require.Equal(t, PostSimilarityThreshold, notifierDeduplicator.SimilarityThreshold)
	})

	t.Run("Test_UserHadSimilarPost", func(t *testing.T) {
		notifierDeduplicator := NewNotifierDeduplicator(PostTTL, PostSimilarityThreshold)
		require.False(t, notifierDeduplicator.UserHadSimilarPost(users[0].Id, similarPosts[0]))
		require.True(t, notifierDeduplicator.UserHadSimilarPost(users[0].Id, similarPosts[1]))
		require.False(t, notifierDeduplicator.UserHadSimilarPost(users[0].Id, posts[0]))
	})

	t.Run("Test_CleanExpiredPosts", func(t *testing.T) {
		localUserId1 := users[0].Id
		localUserId2 := users[1].Id
		notifierDeduplicator := NewNotifierDeduplicator(PostTTL, PostSimilarityThreshold)

		similarPosts[0].ContentGeneratedAt = time.Now()
		similarPosts[1].ContentGeneratedAt = time.Now()
		posts[0].ContentGeneratedAt = time.Now().Add(PostTTL / 2) // valid semanticHashing
		posts[2].ContentGeneratedAt = time.Now().Add(PostTTL / 2) // semanticHashing is null

		// localUserId1: similarPosts[0], similarPosts[1], posts[0], posts[2]
		// localUserId2: similarPosts[1], posts[2]
		notifierDeduplicator.UserHadSimilarPost(localUserId1, similarPosts[0])
		notifierDeduplicator.UserHadSimilarPost(localUserId1, similarPosts[1])
		notifierDeduplicator.UserHadSimilarPost(localUserId1, posts[0])
		notifierDeduplicator.UserHadSimilarPost(localUserId1, posts[2])
		notifierDeduplicator.UserHadSimilarPost(localUserId2, similarPosts[0])
		notifierDeduplicator.UserHadSimilarPost(localUserId2, posts[2])

		require.Equal(t, 2, len(notifierDeduplicator.UserIdToPosts))
		require.Equal(t, 3, len(notifierDeduplicator.UserIdToPosts[localUserId1]))
		require.Equal(t, 2, len(notifierDeduplicator.UserIdToPosts[localUserId2]))

		time.Sleep(PostTTL)
		// similarPosts[0] and similarPosts[1] should be expired
		notifierDeduplicator.CleanExpiredPosts()
		require.Equal(t, 2, len(notifierDeduplicator.UserIdToPosts))
		require.Equal(t, 2, len(notifierDeduplicator.UserIdToPosts[localUserId1]))
		require.Equal(t, 1, len(notifierDeduplicator.UserIdToPosts[localUserId2]))

		time.Sleep(PostTTL)
		// posts[0] and posts[2] should be expired
		notifierDeduplicator.CleanExpiredPosts()
		require.Equal(t, 2, len(notifierDeduplicator.UserIdToPosts))
		require.Equal(t, 0, len(notifierDeduplicator.UserIdToPosts[localUserId1]))
		require.Equal(t, 0, len(notifierDeduplicator.UserIdToPosts[localUserId2]))
	})
}
