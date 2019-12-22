package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/nlopes/slack"
	"golang.org/x/net/html"
)

func init() {
	log.SetFormatter(&log.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})
	log.SetLevel(log.DebugLevel)
	log.Debug("Initialization.")
}

type Author struct {
	Login               string
	Id                  int
	Node_id             string
	Avatar_url          string
	Gravatar_id         int
	Url                 string
	Html_url            string
	Followers_url       string
	Following_url       string
	Gists_url           string
	Starred_url         string
	Subscriptions_url   string
	Organizations_url   string
	Repos_url           string
	Events_url          string
	Received_events_url string
	T_type              string
	Site_admin          bool
}

type Asset struct {
	Url                  string
	Browser_download_url string
	Id                   int
	Node_id              string
	Name                 string
	Label                string
	State                string
	Content_type         string
	Size                 int
	Download_count       int
	Created_at           string
	Updated_at           string
	Uploader             Uploader
}

type Uploader struct {
	Login               string
	Id                  int
	Node_id             string
	Avatar_url          string
	Gravatar_id         string
	Url                 string
	Html_url            string
	Followers_url       string
	Following_url       string
	Gists_url           string
	Starred_url         string
	Subscriptions_url   string
	Organizations_url   string
	Repos_url           string
	Events_url          string
	Received_events_url string
	T_type              string
	Site_admin          bool
}

type GithubResponse struct {
	Url              string
	Html_url         string
	Assets_url       string
	Upload_url       string
	Tarball_url      string
	Zipball_url      string
	Id               int
	Node_id          string
	Tag_name         string
	Target_commitish string
	Name             string
	Body             string
	Draft            bool
	Prerelease       bool
	Created_at       string
	Published_at     string
	Author           Author
	Assets           []Asset
}

func getSlackToken() string {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		exitErrorf("unable to load SDK config, %v", err)
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
		exitErrorf("KMS Error: %v", err)
	}
	return string(result.Plaintext)
}

func sendSlackMessage(channel string, message string) {
	slackToken := getSlackToken()
	api := slack.New(slackToken)
	channelID, timestamp, err := api.PostMessage(channel, slack.MsgOptionText(message, false))
	if err != nil {
		exitErrorf("Sending a message to Slack failed, %v", err)
	}
	log.WithFields(log.Fields{"channel_id": channelID, "timestamp": timestamp})
	log.Info("Slack message sent successfully.")
}

func readFileServer(url string, dependency chan string) {
	lastRunTime := getLastRunTime()
	response, err := http.Get(url)
	if err != nil {
		exitErrorf("The HTTP request failed with error %s\n", err)
	} else {
		data, _ := ioutil.ReadAll(response.Body)
		doc, err := html.Parse(strings.NewReader(string(data)))
		if err != nil {
			exitErrorf("Unable to parse HTML, %v", err)
		}
		var f func(*html.Node, chan string)
		f = func(n *html.Node, dependency chan string) {
			if n.Type == html.ElementNode && n.Data == "a" {
				for _, a := range n.Attr {
					if a.Key == "href" {
						log.Info(a.Val)
						dependency <- a.Val
						//break
					}
				}
			}
			// This only works for CastXMR. Will have to be made more robust.
			if n.Type == html.TextNode {
				log.Debug("text node")
				trimStr := strings.TrimSpace(n.Data)
				t, _ := time.Parse("2006-01-02 15:04", trimStr)
				fileTime := t.Format(time.RFC1123)
				lr_t, _ := time.Parse("Mon, 02 Jan 2006 15:04:05 MST", lastRunTime)
				file_t, _ := time.Parse("Mon, 02 Jan 2006 15:04:05 MST", fileTime)
				if lr_t.Before(file_t) {
					sendSlackMessage(os.Getenv("SLACK_CHANNEL"), "New version of software available at file server: "+url)
				}
			}
			for c := n.LastChild; c != nil; c = c.PrevSibling {
				f(c, dependency)
			}
		}
		go f(doc, dependency)
	}
}

func queryGithub(owner string, miner string) {
	lastRunTime := getLastRunTime()
	c := &http.Client{}
	latestReleaseUrl := "https://api.github.com/repos/" + owner + "/" + miner + "/releases/latest"
	req, _ := http.NewRequest("GET", latestReleaseUrl, nil)
	req.Header.Set("If-Modified-Since", lastRunTime)
	res, err := c.Do(req)
	if err != nil {
		fmt.Printf("The HTTP request failed with error %s\n", err)
	}
	defer res.Body.Close()
	htmlData, err := ioutil.ReadAll(res.Body)
	if err != nil {
		exitErrorf("Failed with error %s\n", err)
	}
	//serverType := res.Header.Get("Server")
	rateLimitRemaining := res.Header.Get("X-RateLimit-Remaining")
	i, err := strconv.Atoi(rateLimitRemaining)
	if err != nil {
		exitErrorf("Failed with error %s\n", err)
	}
	if i < 10 {
		sendSQSMessage("WARNING: API Rate: " + rateLimitRemaining)
	}

	var githubResponse GithubResponse
	json.Unmarshal([]byte(string(htmlData)), &githubResponse)
	if err != nil {
		fmt.Printf("The HTTP request failed with error %s\n", err)
	} else {
		if res.StatusCode == 200 {
			time_now := time.Now().UTC()
			time_published, _ := time.Parse("2006-01-02T15:04:05Z", githubResponse.Published_at)
			time_last_run, _ := time.Parse("Mon, 02 Jan 2006 15:04:05 MST", lastRunTime)
			if time_last_run.Before(time_published) && time_now.After(time_published) {
				sendSlackMessage(os.Getenv("SLACK_CHANNEL"), "New version of "+miner)
				sendSlackMessage(os.Getenv("SLACK_CHANNEL"), githubResponse.Html_url)
			}
			var assets []Asset = githubResponse.Assets
			for _, asset := range assets {
				time_asset_created, _ := time.Parse("2006-01-02T15:04:05Z", asset.Created_at)
				fmt.Println("Asset Creation Time: " + time_asset_created.String())
				if time_last_run.Before(time_asset_created) && time_now.After(time_asset_created) {
					sendSlackMessage(os.Getenv("SLACK_CHANNEL"), "The latest release of "+miner+" has been updated.")
					sendSlackMessage(os.Getenv("SLACK_CHANNEL"), asset.Name+":"+asset.Url)
				}
			}
		} else if res.StatusCode == 304 {
			fmt.Println("No update for " + miner)
		} else {
			sendSQSMessage("Uncaught HTTP Error: " + http.StatusText(res.StatusCode))
			//fmt.Printf("The HTTP request failed with error %d: %s\n", res.StatusCode, http.StatusText(res.StatusCode))
		}
	}
}

func getLastRunTime() string {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		panic("unable to load SDK config, " + err.Error())
	}

	cfg.Region = endpoints.UsEast1RegionID
	tableName := os.Getenv("SYSTEM_TABLE")

	svc := dynamodb.New(cfg)
	input := &dynamodb.QueryInput{
		TableName: aws.String(tableName),
		ExpressionAttributeNames: map[string]string{
			"#K": "Key",
		},
		ExpressionAttributeValues: map[string]dynamodb.AttributeValue{
			":v1": {
				S: aws.String("lastruntime"),
			},
		},
		KeyConditionExpression: aws.String("#K = :v1"),
	}

	req := svc.QueryRequest(input)
	result, err := req.Send()
	if err != nil {
		exitErrorf("DynamoDB Error: %v", err)
	}
	return *result.Items[0]["Value"].S
}

func readMinerTable() []map[string]dynamodb.AttributeValue {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		exitErrorf("Unable to load SDK config: %v", err)
	}

	cfg.Region = endpoints.UsEast1RegionID
	tableName := os.Getenv("MINER_TABLE")

	svc := dynamodb.New(cfg)
	input := &dynamodb.ScanInput{
		TableName: aws.String(tableName),
	}

	req := svc.ScanRequest(input)
	result, err := req.Send()
	if err != nil {
		exitErrorf("DynamoDB Error: %v", err)
	}
	return result.Items
}

func writeLastRunTime() {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		exitErrorf("Unable to load SDK config: %v", err)
	}

	cfg.Region = endpoints.UsEast1RegionID

	t := time.Now().UTC()
	currentTimeUTC := t.Format(time.RFC1123)

	// Github only takes the GMT suffix.
	// Counts against rate limit if removed.
	currentTimeGMT := strings.Replace(currentTimeUTC, "UTC", "GMT", -1)
	tableName := os.Getenv("SYSTEM_TABLE")

	svc := dynamodb.New(cfg)
	input := &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item: map[string]dynamodb.AttributeValue{
			"Key": {
				S: aws.String("lastruntime"),
			},
			"Value": {
				S: aws.String(currentTimeGMT),
			},
		},
	}

	req := svc.PutItemRequest(input)
	result, err := req.Send()
	if err != nil {
		exitErrorf("DynamoDB Error: %v", err)
	}
	log.Debug(result)
}

func exitErrorf(msg string, args ...interface{}) {
	s := fmt.Sprintf(msg+"\n", args...)
	sendSQSMessage(s)
	log.Fatal(s)
	os.Exit(1)
}

func sendSQSMessage(message string) {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		panic("unable to load SDK config, " + err.Error())
	}

	// Set the AWS Region that the service clients should use
	cfg.Region = endpoints.UsEast1RegionID
	svc := sqs.New(cfg)

	input := &sqs.SendMessageInput{
		DelaySeconds: aws.Int64(1),
		QueueUrl:     aws.String("https://sqs.us-east-1.amazonaws.com/385445628596/prospectbot-errors-dev"),
		MessageBody:  aws.String(message),
	}

	//TODO: Get queue url dynamically
	req := svc.SendMessageRequest(input)
	resp, err := req.Send()
	if err != nil {
		exitErrorf("failed to send message: %v\n", err)
	}
	log.Info("Error message successfully sent to SQS.")
	log.Debug(resp)
}

func checkMiners() (string, error) {
	miners := readMinerTable()
	for _, miner := range miners {
		queryGithub(*miner["GithubOwner"].S, *miner["Name"].S)
	}
	writeLastRunTime()
	return "Successful!", nil
}

func main() {
	lambda.Start(checkMiners)
}
