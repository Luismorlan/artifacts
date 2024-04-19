package main

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/rnr-capital/newsfeed-backend/panoptic"
)

const (
	StaleTime = 60 * time.Minute
	PoolSize  = 60
)

func deleteLambdaFunctions(client *lambda.Client, ctx context.Context) int {
	res, err := client.ListFunctions(ctx, &lambda.ListFunctionsInput{})
	if err != nil {
		panic(err)
	}
	count := make(chan int, len(res.Functions))
	funcChan := make(chan types.FunctionConfiguration, PoolSize)
	wg := sync.WaitGroup{}

	for i := 0; i < PoolSize; i++ {
		wg.Add(1)
		go func(funcs chan types.FunctionConfiguration, count chan<- int) {
			defer wg.Done()
			for f := range funcs {
				now := time.Now()
				if !strings.HasPrefix(*f.FunctionName, "data_collector_") {
					count <- 0
					continue
				}
				lastModifiedTime, err := time.Parse("2006-01-02T15:04:05-0700", *f.LastModified)
				if err != nil {
					panic(err)
				}

				if now.Sub(lastModifiedTime) > StaleTime {
					_, err := client.DeleteFunction(ctx, &lambda.DeleteFunctionInput{
						FunctionName: f.FunctionName,
					})

					if err != nil {
						fmt.Println("error deleting function:", err)
						count <- 0
					} else {
						fmt.Println("function deleted, name:", *f.FunctionName, "Created at:", *f.LastModified)
						count <- 1
					}
				} else {
					count <- 0
				}
			}
		}(funcChan, count)
	}

	for _, f := range res.Functions {
		funcChan <- f
	}
	close(funcChan)
	wg.Wait()

	// sum up the count
	deleted := 0
	for i := 0; i < len(res.Functions); i++ {
		deleted += <-count
	}
	fmt.Printf("deleted %d functions:", deleted)
	return deleted
}

func main() {
	flag.Parse()

	var client *lambda.Client
	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(panoptic.AwsRegion),
	)
	if err != nil {
		panic(err)
	}
	client = lambda.NewFromConfig(cfg)

	for deleteLambdaFunctions(client, ctx) > 0 {
		fmt.Println("== still have Lambda function to clean up ==")
	}
}
