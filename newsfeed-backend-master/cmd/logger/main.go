package main

import (
	"flag"
	"log"
	"time"

	"github.com/rnr-capital/newsfeed-backend/deduplicator"
	"github.com/rnr-capital/newsfeed-backend/protocol"
	. "github.com/rnr-capital/newsfeed-backend/publisher"
	"github.com/rnr-capital/newsfeed-backend/utils"
	. "github.com/rnr-capital/newsfeed-backend/utils"
	"github.com/rnr-capital/newsfeed-backend/utils/dotenv"
	. "github.com/rnr-capital/newsfeed-backend/utils/flag"
	"google.golang.org/grpc"
)

const (
	crawlerPublisherQueueName    = "newsfeed_crawled_items_queue.fifo"
	devCrawlerPublisherQueueName = "crawler-publisher-queue"
	// Read batch size must be within [1, 10]
	sqsReadBatchSize                 = 10
	publishMaxBackOffSeconds float64 = 2.0
	initialBackOff           float64 = 0.1
)

var (
	serverAddr = flag.String("deduplicator_addr", "localhost:50051", "The server address in the format of host:port for deduplicator")
)

func getDeduplicatorClientAndConnection() (protocol.DeduplicatorClient, *grpc.ClientConn) {
	if !utils.IsProdEnv() {
		return deduplicator.FakeDeduplicatorClient{}, nil
	}

	opts := []grpc.DialOption{grpc.WithInsecure()}
	conn, err := grpc.Dial(*serverAddr, opts...)
	if err != nil {
		log.Fatalf("fail to dial deduplicator: %v", err)
	}
	client := protocol.NewDeduplicatorClient(conn)

	return client, conn
}

func getNewBackOff(backOff float64) float64 {
	if backOff == 0.0 {
		return initialBackOff
	} else if backOff*2 < publishMaxBackOffSeconds {
		return 2 * backOff
	}
	return publishMaxBackOffSeconds
}

func main() {
	ParseFlags()

	if err := dotenv.LoadDotEnvs(); err != nil {
		panic("fail to load env : " + err.Error())
	}

	db, err := GetDBConnection()
	if err != nil {
		panic("fail to connect database : " + err.Error())
	}
	PublisherDBSetup(db)

	client, conn := getDeduplicatorClientAndConnection()
	defer conn.Close()

	sqsName := crawlerPublisherQueueName
	if !utils.IsProdEnv() {
		sqsName = devCrawlerPublisherQueueName
	}
	reader, err := NewSQSMessageQueueReader(sqsName, 20)
	if err != nil {
		panic("fail initialize SQS message queue reader : " + err.Error())
	}

	// Main publish logic lives in processor
	processor := NewPublisherMessageProcessor(reader, db, client)

	// Exponentially backoff on
	backOff := 0.0
	log.Println("start processing messages")
	for {
		msgs, err := processor.Reader.ReceiveMessages(sqsReadBatchSize)
		if err != nil {
			log.Println("fail to receive messages from SQS : ", err.Error())
			backOff = getNewBackOff(backOff)
		} else {
			for _, msg := range msgs {
				decodedMsg, err := processor.DecodeCrawlerMessage(msg)
				if err != nil {
					log.Println("fail to decode message : ", err.Error())
					backOff = getNewBackOff(backOff)
				} else {
					log.Printf("processing message : %v", decodedMsg)
				}
			}
		}

		// Protective back off on read or process failure.
		time.Sleep(time.Duration(backOff) * time.Second)
	}
}
