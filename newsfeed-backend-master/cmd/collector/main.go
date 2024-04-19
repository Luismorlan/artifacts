package main

import (
	ddlambda "github.com/DataDog/datadog-lambda-go"
	"github.com/aws/aws-lambda-go/lambda"
	collector_hander "github.com/rnr-capital/newsfeed-backend/collector/handler"
	"github.com/rnr-capital/newsfeed-backend/model"
	"github.com/rnr-capital/newsfeed-backend/protocol"
	"github.com/rnr-capital/newsfeed-backend/utils/dotenv"
	. "github.com/rnr-capital/newsfeed-backend/utils/flag"
	. "github.com/rnr-capital/newsfeed-backend/utils/log"
	"google.golang.org/protobuf/proto"
)

func init() {
	LogV2.Info("data collector initialized")
}

func cleanup() {
	LogV2.Info("data collector shutdown")
}

func HandleRequest(event model.DataCollectorRequest) (model.DataCollectorResponse, error) {
	res := model.DataCollectorResponse{}

	// parse job
	job := &protocol.PanopticJob{}
	if err := proto.Unmarshal(event.SerializedJob, job); err != nil {
		LogV2.Errorf("Failed to parse job with error:", err)
		return res, err
	}

	// handle
	var handler collector_hander.DataCollectJobHandler
	err := handler.Collect(job)
	if err != nil {
		LogV2.Errorf("Failed to execute job with error:", err)
		return res, err
	}
	// encode job
	bytes, err := proto.Marshal(job)
	if err != nil {
		return res, err
	}

	res.SerializedJob = bytes
	return res, nil
}

func main() {
	ParseFlags()

	defer cleanup()
	if err := dotenv.LoadDotEnvs(); err != nil {
		panic(err)
	}
	LogV2.Info("Starting lambda handler, waiting for requests...")

	lambda.Start(ddlambda.WrapFunction(HandleRequest, nil))
}
