package ec2cluster

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// DiscoverClusterMembersByTag returns a list of the private IP addresses of each
// EC2 instance having a tag that matches the tag on the currently running
// instance. The tag is specified by tagName.
//
// For example, if your instances are tagged with a tag named `app` whose
// value is unique per cluster, then you can invoke:
//
//     knownClusterMembers, err := DiscoverClusterMembersByTag(nil, "app")
//
func DiscoverClusterMembersByTag(config *aws.Config, tagName string) ([]string, error) {
	instanceID, err := discoverInstanceID()
	if err != nil {
		return nil, err
	}

	EC2 := ec2.New(config)
	resp, err := EC2.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{aws.String(instanceID)},
		MaxResults:  aws.Int64(1),
	})
	if err != nil {
		return nil, err
	}
	if len(resp.Reservations) != 1 || len(resp.Reservations[0].Instances) != 1 {
		return nil, fmt.Errorf("Cannot find instance %s", instanceID)
	}

	tagValue := ""
	for _, tag := range resp.Reservations[0].Instances[0].Tags {
		if *tag.Key == tagName {
			tagValue = *tag.Value
		}
	}
	if tagValue == "" {
		return nil, fmt.Errorf("Current instance (%s) does not have a tag %s.",
			instanceID, tagName)
	}

	// List all the instances having that tag
	rv := []string{}

	err = EC2.DescribeInstancesPages(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String(fmt.Sprintf("tag:%s", tagName)),
				Values: []*string{aws.String(tagValue)},
			},
		},
	}, func(resp *ec2.DescribeInstancesOutput, lastPage bool) (shouldContinue bool) {
		for _, reservation := range resp.Reservations {
			for _, instance := range reservation.Instances {
				rv = append(rv, *instance.PrivateIpAddress)
			}
		}
		return true
	})
	if err != nil {
		return nil, err
	}

	return rv, nil
}

// DiscoverAdvertiseAddress returns the address that should be advertised for
// the current node based on the current EC2 instance's private IP address
func DiscoverAdvertiseAddress() (string, error) {
	return readMetadata("local-ipv4")
}

func discoverInstanceID() (string, error) {
	return readMetadata("instance-id")
}

func readMetadata(suffix string) (string, error) {
	// a nice short timeout so we don't hang too much on non-AWS boxes
	client := http.DefaultClient
	client.Timeout = 200 * time.Millisecond

	resp, err := client.Get("http://169.254.169.254/latest/meta-data/" + suffix)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetching metadata: %s", resp.Status)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
