package twitter

import (
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
	Logger "github.com/rnr-capital/newsfeed-backend/utils/log"
)

func HandleTwitterMessage(c *gin.Context) {
	jsonData, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, "fail to get request body"+err.Error())
		return
	}
	Logger.LogV2.Info("result is" + string(jsonData))
}
