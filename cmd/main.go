package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aiell0/internal"
)

func main() {
	lambda.Start(prospectbot.CheckMiners)
}
