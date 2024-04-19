package notifier

import (
	"fmt"
	"math"
	"time"

	"github.com/drewlanenga/govector"
	"github.com/pgvector/pgvector-go"
	"github.com/rnr-capital/newsfeed-backend/model"
)

const (
	SimilarityThreshold = 0.2
)

// PartialPostForDedup only keeps few fields from model.Post
// for deduplication purpose, NotifierDeduplicator will keep
// up to one day's posts.
type PartialPostForDedup struct {
	Id                 string
	ContentGeneratedAt time.Time
	Embedding          *pgvector.Vector
}

type NotifierDeduplicator struct {
	// store userId to partial post map
	UserIdToPosts       map[string][]PartialPostForDedup
	PostTTL             time.Duration
	SimilarityThreshold int
}

func NewNotifierDeduplicator(period time.Duration, similarityThreshold int) *NotifierDeduplicator {
	return &NotifierDeduplicator{
		UserIdToPosts:       map[string][]PartialPostForDedup{},
		PostTTL:             period,
		SimilarityThreshold: similarityThreshold,
	}
}

func (n *NotifierDeduplicator) CleanExpiredPosts() {
	currentTime := time.Now()
	expiredPostCount := 0
	totalPostCount := 0
	for useId, posts := range n.UserIdToPosts {
		lastExpiredIdx := -1
		for idx, post := range posts {
			// assume all posts have valid ContentGeneratedAt
			expired := post.ContentGeneratedAt.Add(n.PostTTL).Before(currentTime)
			if expired {
				lastExpiredIdx = idx
				expiredPostCount++
			} else {
				// assuming posts with larger idx are also not expired
				break
			}
		}
		if lastExpiredIdx > -1 {
			start := lastExpiredIdx + 1
			end := len(n.UserIdToPosts[useId])
			n.UserIdToPosts[useId] = n.UserIdToPosts[useId][start:end]
		}
		totalPostCount += len(n.UserIdToPosts[useId])
	}
	Log.Info(fmt.Sprintf("CleanExpiredPosts: posts count expired: %d, existing: %d", expiredPostCount, totalPostCount))
}

/*
 * UserHadSimilarPost checks if a given post has any similar post among user's
 * received posts, and save a light copy of new post for future checks.
 */
func (n *NotifierDeduplicator) UserHadSimilarPost(userId string, post model.Post) bool {
	userPosts, ok := n.UserIdToPosts[userId]
	if ok {
		for _, existingPost := range userPosts {
			if n.isEmbeddingSimilar(post.Embedding, existingPost.Embedding) {
				return true
			}
		}
		n.UserIdToPosts[userId] = append(n.UserIdToPosts[userId], n.getPartialPostForDedupFromPost(post))
		return false
	}
	// !ok case, initialize the PartialPostForDedup array
	n.UserIdToPosts[userId] = []PartialPostForDedup{
		n.getPartialPostForDedupFromPost(post),
	}
	return false
}

func (n *NotifierDeduplicator) isEmbeddingSimilar(e1 *pgvector.Vector, e2 *pgvector.Vector) bool {
	// If the hashing is invalid, or not of same length, they cannot be considered
	// as the semantically identical.
	if e1 == nil || e2 == nil {
		return false
	}
	v1, _ := govector.AsVector(e1.Slice())
	v2, _ := govector.AsVector(e2.Slice())
	diff, _ := v1.Subtract(v2)
	norm := govector.Norm(diff, 2)
	return math.Sqrt(norm) <= SimilarityThreshold
}

func (n *NotifierDeduplicator) getPartialPostForDedupFromPost(post model.Post) PartialPostForDedup {
	return PartialPostForDedup{
		Id:                 post.Id,
		ContentGeneratedAt: post.ContentGeneratedAt,
		Embedding:          post.Embedding,
	}
}
