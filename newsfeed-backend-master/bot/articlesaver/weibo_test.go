package articlesaver

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetWeiboArticle(t *testing.T) {

	// t.Run("returns error on request error", func(t *testing.T) {
	// 	// Arrange
	// 	url := "http://example.com"

	// 	// Act
	// 	_, err := GetWeiboArticle(url)

	// 	// Assert
	// 	assert.Error(t, err)
	// })

	// t.Run("returns error on non-200 status code", func(t *testing.T) {
	// 	// Arrange
	// 	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// 		w.WriteHeader(500)
	// 	}))
	// 	defer ts.Close()

	// 	url := ts.URL

	// 	// Act
	// 	_, err := GetWeiboArticle(url)

	// 	// Assert
	// 	assert.Error(t, err)
	// })

	t.Run("returns document on success", func(t *testing.T) {
		// // Arrange
		// ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 	w.WriteHeader(200)
		// 	w.Write([]byte("<html><body></body></html>"))
		// }))
		// defer ts.Close()

		// url := ts.URL

		// Act
		doc, err := GetWeiboArticle("https://weibo.com/ttarticle/p/show?id=2309404944527164310096")
		assert.NoError(t, err)
		assert.NotNil(t, doc)

		link, err := SaveWeiboDocToNotion(doc, "https://weibo.com/ttarticle/p/show?id=2309404944527164310096")
		assert.NoError(t, err)
		assert.NotNil(t, link)
		fmt.Println(link)
		// Check if the content is gzip encoded

		// Assert
	})

}
