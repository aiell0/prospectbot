package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/blockforgecapital/internal"
)

func main() {
	lambda.Start(prospectbot.CheckMiners)
}
