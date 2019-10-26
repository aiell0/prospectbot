package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/prospectbot/prospectbot"
)

func main() {
	lambda.Start(prospectbot.HandleError)
}
