module github.com/rnr-capital/newsfeed-backend

go 1.16

require (
	github.com/99designs/gqlgen v0.17.40
	github.com/DataDog/datadog-go v4.8.2+incompatible
	github.com/DataDog/datadog-lambda-go v1.3.0
	github.com/Microsoft/go-winio v0.5.0 // indirect
	github.com/PuerkitoBio/goquery v1.7.1
	github.com/ThreeDotsLabs/watermill v1.1.1
	github.com/andybalholm/brotli v1.0.4 // indirect
	github.com/antchfx/htmlquery v1.2.3 // indirect
	github.com/antchfx/xmlquery v1.3.7
	github.com/araddon/dateparse v0.0.0-20210429162001-6b43995a97de
	github.com/aws/aws-lambda-go v1.27.0
	github.com/aws/aws-sdk-go v1.40.19
	github.com/aws/aws-sdk-go-v2/config v1.5.0
	github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider v1.4.1
	github.com/aws/aws-sdk-go-v2/service/lambda v1.9.0
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/drewlanenga/govector v0.0.0-20220726163947-b958ac08bc93
	github.com/dstotijn/go-notion v0.11.0
	github.com/gin-contrib/cors v1.3.1
	github.com/gin-gonic/gin v1.7.2
	github.com/go-playground/validator/v10 v10.8.0 // indirect
	github.com/go-redis/redis/v8 v8.11.4
	github.com/go-resty/resty/v2 v2.7.0
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/gocolly/colly v1.2.0
	github.com/google/go-cmp v0.5.9
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/uuid v1.3.0
	github.com/gorilla/websocket v1.5.0
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/jinzhu/copier v0.3.2
	github.com/joho/godotenv v1.3.0
	github.com/json-iterator/go v1.1.11 // indirect
	github.com/kennygrant/sanitize v1.2.4 // indirect
	github.com/lib/pq v1.10.2
	github.com/logdna/logdna-go v1.0.2
	github.com/mmcdole/gofeed v1.1.3
	github.com/n0madic/twitter-scraper v0.0.0-20230711213008-94503a2bc36c
	github.com/pgvector/pgvector-go v0.1.1
	github.com/philhofer/fwd v1.1.1 // indirect
	github.com/pkg/errors v0.9.1
	github.com/saintfish/chardet v0.0.0-20120816061221-3af4cd4741ca // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/slack-go/slack v0.9.5
	github.com/stretchr/testify v1.8.2
	github.com/temoto/robotstxt v1.1.2 // indirect
	github.com/ugorji/go v1.2.6 // indirect
	github.com/vektah/gqlparser/v2 v2.5.10
	golang.org/x/exp v0.0.0-20211029182501-9b944d235b9d // indirect
	golang.org/x/net v0.12.0
	golang.org/x/oauth2 v0.9.0
	gonum.org/v1/gonum v0.9.3
	google.golang.org/genproto v0.0.0-20211208223120-3a66f561d7aa // indirect
	google.golang.org/grpc v1.43.0
	google.golang.org/protobuf v1.30.0
	gopkg.in/DataDog/dd-trace-go.v1 v1.33.0
	gopkg.in/yaml.v2 v2.4.0
	gorm.io/datatypes v1.0.4
	gorm.io/driver/mysql v1.3.3 // indirect
	gorm.io/driver/postgres v1.3.4
	gorm.io/gorm v1.23.4
)

replace sourcegraph.com/sourcegraph/appdash => github.com/sourcegraph/appdash v0.0.0-20190731080439-ebfcffb1b5c0
