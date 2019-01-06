package main

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/nlopes/slack"
	"golang.org/x/net/html"
)

const bucketName string = "blockforge-infrastructure"
const slackChannel string = "DC6V5T82E"
const slackToken string = "xoxa-410442786752-414276217760-414753865764-e6e4ea550bd22c5c19a3c8eeef3fb2e4"
const ddwrtLocation string = "https://download1.dd-wrt.com/dd-wrtv2/downloads/betas/2018/"
const castXmrLocation string = "http://www.gandalph3000.com/download/"
const xmrStakLocation string = "https://api.github.com/repos/fireice-uk/xmr-stak"
const xmrRigNvidiaLocation string = "https://api.github.com/repos/xmrig/xmrig-nvidia"
const xmrRigAmdLocation string = "https://api.github.com/repos/xmrig/xmrig-amd"
const finminerEthLocation string = "https://api.github.com/repos/FinMiner/FinMiner"
const claymoreEthLocation string = "https://api.github.com/repos/nanopool/Claymore-Dual-Miner"
const claymoreZecLocation string = "https://api.github.com/repos/nanopool/ClaymoreZECMiner"
const excavatorZecLocation string = "https://api.github.com/repos/nanopool/excavator"
const ewbfZecLocation string = "https://api.github.com/repos/nanopool/ewbf-miner"
const sgminerZecLocation string = "https://api.github.com/repos/genesismining/sgminer-gm"
const nheqZecLocation string = "https://api.github.com/repos/nanopool/nheqminer"
const rhminerPascalLocation string = "https://api.github.com/repos/nanopool/rhminer"
const claymoreXmrLocation string = "https://api.github.com/repos/nanopool/Claymore-XMR-Miner"
const trexRavencoinLocation string = "https://api.github.com/repos/nanopool/trex"
const avermoreRavencoinLocation string = "https://api.github.com/repos/nanopool/avermore"
const nanominerLocation string = "https://api.github.com/repos/nanopool/nanominer"
const suprminerLocation string = "https://api.github.com/repos/ocminer/suprminer"
const wildrigLocation string = "https://api.github.com/repos/andru-kun/wildrig-multi"
const grinminerLocation string = "https://api.github.com/repos/mimblewimble/grin-miner"
const beamLocation string = "https://api.github.com/repos/BeamMW/beam"
const router string = "linksys-wrt3200acm"

func init() {
	log.SetFormatter(&log.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})
	log.SetLevel(log.DebugLevel)
	log.Debug("Initialization.")
}

func exitErrorf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	log.Fatal(msg)
	os.Exit(1)
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

type Assets struct {
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
	Assets           []Assets
}

func sendSlackMessage(channel string, message string) {
	api := slack.New(slackToken)
	channelID, timestamp, err := api.PostMessage(channel, slack.MsgOptionText(message, false))
	if err != nil {
		exitErrorf("Sending a message to Slack failed, %v", err)
	}

	// Not sure how to handle this in Golang yet.
	_ = timestamp

	log.WithFields(log.Fields{"channel_id": channelID})
	log.Info("Slack message sent successfully.")
}

func readFileServer(url string, dependency chan string) {
	response, err := http.Get(url)
	if err != nil {
		fmt.Println("The HTTP request failed with error %s\n", err)
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
						//dependency <- a.Val
						//break
					}
				}
			}
			if n.Type == html.TextNode {
				log.Info("We are in a text node!")
				// Figure out if it's a date
				trimStr := strings.TrimSpace(n.Data)
				log.Debug(trimStr)
				t, _ := time.Parse("2006-01-02 15:04", trimStr)
				//_ = t
				//if err != nil {
				//	log.Debug("Not a time string")
				//} else {
				//	log.Debug("It is a time string!")
				//}
				fileTime := t.Format(time.RFC1123)
				lastRunTime := getLastRunTime()
				//log.Debug(fileTime.After(lastRunTime))
			}
			for c := n.LastChild; c != nil; c = c.PrevSibling {
				f(c, dependency)
			}
		}
		go f(doc, dependency)
	}
}

func githubResourceUpdate(url string) {
	splitUrl := strings.Split(url, "/")
	repoName := splitUrl[len(splitUrl)-1]
	lastRunTime := getLastRunTime()
	fmt.Println(lastRunTime)
	c := &http.Client{}
	latestReleaseUrl := url + "/releases/latest"
	req, _ := http.NewRequest("GET", latestReleaseUrl, nil)
	req.Header.Set("If-Modified-Since", lastRunTime)
	res, err := c.Do(req)
	if err != nil {
		fmt.Printf("The HTTP request failed with error %s\n", err)
	}
	defer res.Body.Close()
	htmlData, err := ioutil.ReadAll(res.Body)
	serverType := res.Header.Get("Server")
	if serverType == "GitHub.com" {
		fmt.Println("Server is GitHub.")
		fmt.Println("API Calls Remaining: " + res.Header.Get("X-RateLimit-Remaining"))
	} else {
		panic("Unsupported server type. Only GitHub is supported.")
	}
	if err != nil {
		fmt.Printf("Failed with error %s\n", err)
		os.Exit(1)
	}

	var githubResponse GithubResponse
	json.Unmarshal([]byte(string(htmlData)), &githubResponse)
	if err != nil {
		fmt.Println("The HTTP request failed with error %s\n", err)
	} else {
		if res.StatusCode == 200 {
			sendSlackMessage(slackChannel, repoName+" has been updated!")
			sendSlackMessage(slackChannel, githubResponse.Html_url)
		} else if res.StatusCode == 304 {
			fmt.Println("No update for " + repoName)
		} else {
			fmt.Printf("The HTTP request failed with error %d: %s\n", res.StatusCode, http.StatusText(res.StatusCode))
		}
	}
}

func getLastRunTime() string {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		panic("unable to load SDK config, " + err.Error())
	}

	cfg.Region = endpoints.UsEast1RegionID

	svc := dynamodb.New(cfg)
	input := &dynamodb.QueryInput{
		TableName: aws.String("FinSense"),
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
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeProvisionedThroughputExceededException:
				fmt.Println(dynamodb.ErrCodeProvisionedThroughputExceededException, aerr.Error())
			case dynamodb.ErrCodeResourceNotFoundException:
				fmt.Println(dynamodb.ErrCodeResourceNotFoundException, aerr.Error())
			//case dynamodb.ErrCodeRequestLimitExceeded:
			//	fmt.Println(dynamodb.ErrCodeRequestLimitExceeded, aerr.Error())
			case dynamodb.ErrCodeInternalServerError:
				fmt.Println(dynamodb.ErrCodeInternalServerError, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Not an AWS error
			fmt.Println(err.Error())
		}
		panic("There was an error with DynamoDB")
	}
	return *result.Items[0]["Value"].S
}

func writeLastRunTime() {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		panic("unable to load SDK config, " + err.Error())
	}

	cfg.Region = endpoints.UsEast1RegionID

	t := time.Now().UTC()
	currentTimeUTC := t.Format(time.RFC1123)

	// Github only takes the GMT suffix.
	// Counts against rate limit if removed.
	currentTimeGMT := strings.Replace(currentTimeUTC, "UTC", "GMT", -1)

	svc := dynamodb.New(cfg)
	input := &dynamodb.PutItemInput{
		Item: map[string]dynamodb.AttributeValue{
			"Key": {
				S: aws.String("lastruntime"),
			},
			"Value": {
				S: aws.String(currentTimeGMT),
			},
		},
		TableName: aws.String("FinSense"),
	}

	req := svc.PutItemRequest(input)
	result, err := req.Send()
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeConditionalCheckFailedException:
				fmt.Println(dynamodb.ErrCodeConditionalCheckFailedException, aerr.Error())
			case dynamodb.ErrCodeProvisionedThroughputExceededException:
				fmt.Println(dynamodb.ErrCodeProvisionedThroughputExceededException, aerr.Error())
			case dynamodb.ErrCodeResourceNotFoundException:
				fmt.Println(dynamodb.ErrCodeResourceNotFoundException, aerr.Error())
			case dynamodb.ErrCodeItemCollectionSizeLimitExceededException:
				fmt.Println(dynamodb.ErrCodeItemCollectionSizeLimitExceededException, aerr.Error())
			//case dynamodb.ErrCodeTransactionConflictException:
			//	fmt.Println(dynamodb.ErrCodeTransactionConflictException, aerr.Error())
			//case dynamodb.ErrCodeRequestLimitExceededException:
			//	fmt.Println(dynamodb.ErrCodeRequestLimitExceeded, aerr.Error())
			case dynamodb.ErrCodeInternalServerError:
				fmt.Println(dynamodb.ErrCodeInternalServerError, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Not an AWS error
			fmt.Println(err.Error())
		}
		return
	}
	fmt.Println(result)
}

func main() {
	channel := make(chan string)
	//channel2 := make(chan string)
	//readFileServer(ddwrtLocation, channel, "2019")
	//dependency := <-channel
	//var release string = dependency[11 : len(dependency)-1]
	//var binFile string = "ddwrt-" + router + "-webflash.bin"
	//var ddwrtBin string = "ddwrt-" + release + ".bin"
	//if readFileServer(ddwrtLocation) {
	//	sendSlackMessage(slackChannel, "New version of DD-WRT available!")
	//}

	//readFileServer(castXmrLocation, channel2, "cast_xmr")
	//dependency = <-channel2
	//if softwareUpdate(castXmrLocation) {
	//	sendSlackMessage(slackChannel, "New version of Cast XMR available!")
	//}

	readFileServer(castXmrLocation, channel)

	githubResourceUpdate(xmrStakLocation)
	githubResourceUpdate(xmrRigNvidiaLocation)
	githubResourceUpdate(xmrRigAmdLocation)
	githubResourceUpdate(finminerEthLocation)
	githubResourceUpdate(claymoreEthLocation)
	githubResourceUpdate(claymoreZecLocation)
	githubResourceUpdate(excavatorZecLocation)
	githubResourceUpdate(ewbfZecLocation)
	githubResourceUpdate(sgminerZecLocation)
	githubResourceUpdate(nheqZecLocation)
	githubResourceUpdate(rhminerPascalLocation)
	githubResourceUpdate(claymoreXmrLocation)
	githubResourceUpdate(trexRavencoinLocation)
	githubResourceUpdate(avermoreRavencoinLocation)
	githubResourceUpdate(nanominerLocation)
	githubResourceUpdate(suprminerLocation)
	githubResourceUpdate(wildrigLocation)
	githubResourceUpdate(beamLocation)
	// TODO: Implement handler for repos with no releases.
	//githubResourceUpdate(grinminerLocation)

	//writeLastRunTime()
}
