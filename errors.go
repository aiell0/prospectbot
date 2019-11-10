package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/nlopes/slack"
	log "github.com/sirupsen/logrus"
)

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

func getSlackToken() string {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		panic("unable to load SDK config, " + err.Error())
	}

	// Set the AWS Region that the service clients should use
	cfg.Region = endpoints.UsEast1RegionID
	svc := kms.New(cfg)
	encryptionContext := make(map[string]string)
	encryptionContext["PARAMETER_ARN"] = "arn:aws:ssm:us-east-1:385445628596:parameter/slack/access-token"
	decoded, err := base64.StdEncoding.DecodeString(os.Getenv("SLACK_TOKEN"))
	if err != nil {
		fmt.Println("decode error:", err)
	}
	input := &kms.DecryptInput{
		CiphertextBlob:    []byte(string(decoded)),
		EncryptionContext: encryptionContext,
	}

	req := svc.DecryptRequest(input)
	result, err := req.Send()
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case kms.ErrCodeNotFoundException:
				fmt.Println(kms.ErrCodeNotFoundException, aerr.Error())
			case kms.ErrCodeDisabledException:
				fmt.Println(kms.ErrCodeDisabledException, aerr.Error())
			case kms.ErrCodeInvalidCiphertextException:
				fmt.Println(kms.ErrCodeInvalidCiphertextException, aerr.Error())
			case kms.ErrCodeKeyUnavailableException:
				fmt.Println(kms.ErrCodeKeyUnavailableException, aerr.Error())
			case kms.ErrCodeDependencyTimeoutException:
				fmt.Println(kms.ErrCodeDependencyTimeoutException, aerr.Error())
			case kms.ErrCodeInvalidGrantTokenException:
				fmt.Println(kms.ErrCodeInvalidGrantTokenException, aerr.Error())
			case kms.ErrCodeInternalException:
				fmt.Println(kms.ErrCodeInternalException, aerr.Error())
			case kms.ErrCodeInvalidStateException:
				fmt.Println(kms.ErrCodeInvalidStateException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
	}

	return string(result.Plaintext)
}

func errMsg(ctx context.Context, sqsEvent events.SQSEvent) (string, error) {
	for _, message := range sqsEvent.Records {
		fmt.Printf("The message %s for event source %s = %s \n", message.MessageId, message.EventSource, message.Body)
		sendSlackMessage(message.Body)
	}

	// exit lambda runtime
	return "SUCCESS", nil
}

func exitErrorf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	log.Fatal(msg)
	os.Exit(1)
}

func sendSlackMessage(message string) {
	slackToken := getSlackToken()
	log.Debug(slackToken)
	api := slack.New(slackToken)
	slackChannel := os.Getenv("SLACK_CHANNEL")
	channelID, timestamp, err := api.PostMessage(slackChannel, slack.MsgOptionText(message, false))
	if err != nil {
		exitErrorf("Sending a message to Slack failed, %v", err)
	}

	// Not sure how to handle this in Golang yet.
	_ = timestamp

	log.WithFields(log.Fields{"channel_id": channelID})
	log.Info("Slack message sent successfully.")
}

func main() {
	lambda.Start(errMsg)
}
