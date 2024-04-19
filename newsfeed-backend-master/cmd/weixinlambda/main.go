package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/lambda"
)

type WeixinArticle struct {
	Title  string `json:"title"`
	Source string `json:"source"`
	Url    string `json:"url"`
}

type MyEvent struct {
	Articles []WeixinArticle `json:"articles"`
	BotId    string          `json:"robotId"`
}

type Request struct {
	Body string `json:"body"`
}

func HandleRequest(ctx context.Context, request json.RawMessage) (string, error) {
	var reqBody Request
	err := json.Unmarshal(request, &reqBody)
	if err != nil {
		fmt.Printf("Error while unmarshaling: %s\n", err)
		return "fail", err
	}

	var event MyEvent
	err = json.Unmarshal([]byte(reqBody.Body), &event)
	if err != nil {
		fmt.Printf("Error while unmarshaling: %s\n", err)
		return "fail", err
	}
	for _, article := range event.Articles {
		jsonData, _ := json.Marshal(article)
		fmt.Println("json data", string(jsonData))
		resp, err := http.Post("https://hooks.zapier.com/hooks/catch/14662062/3sygndd/", "application/json", bytes.NewBuffer(jsonData))
		fmt.Println("resp", resp, err)
	}
	return "ok", nil
}

func main() {
	lambda.Start(HandleRequest)
}

func ParseWeixinArticleFromJson(raw string) ([]WeixinArticle, error) {
	var event MyEvent
	err := json.Unmarshal([]byte(raw), &event)
	if err != nil {
		return nil, err
	}
	var res = []WeixinArticle{}
	res = append(res, event.Articles...)
	return res, nil
}
