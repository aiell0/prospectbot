package main

import (
	"github.com/aiell0/prospectbot"
	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	lambda.Start(prospectbot.CheckMiners)
}
