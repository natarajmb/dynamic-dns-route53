// This is a program to update DNS record on Route53
//
// Requires config.yaml to be in same path as the executable
// assumption is that this program is scheduled to run under
// a unix cron like program to update Route53 record every time
// external IP changes.
//
// nataraj.basappa
//@version 1.0, 02/12/2015
//

package main

import (
	"os"
	"fmt"
	"time"
	"bytes"
	"strings"
	"net/http"
	"io/ioutil"
	"text/template"
	"github.com/zpatrick/go-config"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/route53"
)

const (
	CommentTemplate = "Updating Route53 record for {{.DNSRecordName}} at {{.Timestamp}}"
	ActionType = "UPSERT"
	RecordType = "A"
	TTL = 300
)

type UpdateComment struct {
	DNSRecordName string
	Timestamp     string
}

func main() {

	// Initialise and read configuration properties from config.yaml file
	yamlFile := config.NewYAMLFile("config.yaml")
	conf := config.NewConfig([]config.Provider{yamlFile})

	externalIp := getExternalIP(conf)

	// Read configuration properties from config.yaml file
	AWSAccessKeyId, _ := conf.String("aws_setting.aws_access_key_id")
	AWSSecretAccessKey, _ := conf.String("aws_setting.aws_secret_access_key")
	AWSHostedZoneId, _ := conf.String("aws_setting.aws_hosted_zone_id")
	AWSFQDN, _ := conf.String("aws_setting.aws_fqdn")

	// Connect to AWS Route53 service with configurations provided in config.yaml and obtain a session
	svc := route53.New(session.New(&aws.Config{Credentials:credentials.NewStaticCredentials(AWSAccessKeyId, AWSSecretAccessKey, "")}))

	awsIP := getCurrentAWSIP(svc, AWSHostedZoneId, AWSFQDN)

	if strings.Compare(awsIP, externalIp) != 0 {
		updateRoute53Record(svc, AWSHostedZoneId, AWSFQDN, externalIp)
	}

	os.Exit(0)
}

func getExternalIP(conf *config.Config) (externalIP string) {

	var currentIP []byte
	// Read ip_resolvers configuration properties from config.yaml file
	IPResolver, _ := conf.String("ip_resolvers.ip_resolver")
	IPResolverFallback, _ := conf.String("ip_resolvers.ip_resolver_fallback")

	// Check if you have to update Route53 with new IP
	resp1, err1 := http.Get(IPResolver)
	if err1 != nil {
		resp2, err2 := http.Get(IPResolverFallback)
		if err2 != nil {
			fmt.Println(err2.Error())
		}
		currentIP, _ = ioutil.ReadAll(resp2.Body)
	} else {
		currentIP, _ = ioutil.ReadAll(resp1.Body)
	}

	externalIP = strings.TrimSpace(string(currentIP))
	return externalIP
}

func getCurrentAWSIP(svc *route53.Route53, AWSHostedZoneId string, FQDN string) (awsIP string) {

	// Preparing input to retreive records from Route53
	CheckRecord := &route53.ListResourceRecordSetsInput{
		HostedZoneId:aws.String(AWSHostedZoneId),
		StartRecordType:aws.String(RecordType),
		StartRecordName:aws.String(FQDN),
	}

	// Get the record set from Route53 to check currently set IP
	resp, err := svc.ListResourceRecordSets(CheckRecord)
	if err != nil {
		fmt.Println(err.Error())
	}

	// Iterate over the record set to find the right record and value
	var setIP *string
	for _, record := range resp.ResourceRecordSets {
		if *record.Type == RecordType {
			setIP = record.ResourceRecords[0].Value
		}
	}

	if setIP != nil {
		awsIP = *setIP
	}
	return awsIP
}

func updateRoute53Record(svc *route53.Route53, AWSHostedZoneId string, FQDN string, externalIP string) {

	// Bit of templating to read and apply comments to be passed to Route53
	parsed_template, _ := template.New("comments").Parse(CommentTemplate)
	var comments bytes.Buffer
	parsed_template.Execute(&comments, UpdateComment{DNSRecordName:FQDN, Timestamp:time.Now().Format(time.RFC822)})
	AWSUpdateComment := comments.String()

	// Prepare the change set to be applied to Route53
	ChangeRecord := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Comment: aws.String(AWSUpdateComment),
			Changes: []*route53.Change{
				{
					Action: aws.String(ActionType),
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name: aws.String(FQDN),
						Type: aws.String(RecordType),
						TTL: aws.Int64(TTL),
						ResourceRecords: []*route53.ResourceRecord{
							{
								Value: aws.String(externalIP),
							},
						},
					},
				},
			},
		},
		HostedZoneId: aws.String(AWSHostedZoneId),
	}

	// Apply the change set prepared above to Route53, using the passed session
	// ignore response coming back as its not useful to what we are doing
	_, err := svc.ChangeResourceRecordSets(ChangeRecord)

	if err != nil {
		fmt.Println(err.Error())
	}

	return
}