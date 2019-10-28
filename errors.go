package main

import (
	"context"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	log "github.com/sirupsen/logrus"
)

type Attributes struct {
	ApproximateReceiveCount          string
	SentTimestamp                    string
	SenderId                         string
	ApproximateFirstReceiveTimestamp string
}

type Record struct {
	MessageId         string
	ReceiptHandle     string
	Body              string
	Attributes        Attributes
	MessageAttributes struct{}
	Md5OfBody         string
	EventSource       string
	EventSourceARN    string
	AwsRegion         string
}

type SQSEvent struct {
	Records []Record
}

func init() {
	log.SetFormatter(&log.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})
	if os.Getenv("LOG_LEVEL") == "DEBUG" {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
}

func errMsg(ctx context.Context, event SQSEvent) (string, error) {
	log.Info(event)
	// send to slack
	// exit lambda runtime
	return "SUCCESS", nil
}

func main() {
	lambda.Start(errMsg)
}
