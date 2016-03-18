package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/crewjam/awsregion"
	"github.com/crewjam/ec2cluster"
)

func main() {
	instanceID := flag.String("instance", "",
		"The instance ID of the cluster member. If not supplied, then the instance ID is determined from EC2 metadata")
	clusterTagName := flag.String("tag", "aws:autoscaling:groupName",
		"The instance tag that is common to all members of the cluster")
	doWatch := flag.Bool("watch", false,
		"Monitor autoscaling lifecycle events for the current autoscaling group and print them to stdout as they occur")
	flag.Parse()

	if *instanceID == "" {
		var err error
		*instanceID, err = ec2cluster.DiscoverInstanceID()
		if err != nil {
			log.Fatalf("ERROR: %s", err)
		}
	}

	s := &ec2cluster.Cluster{
		InstanceID: *instanceID,
		TagName:    *clusterTagName,
	}

	s.AwsSession = session.New()
	if region := os.Getenv("AWS_REGION"); region != "" {
		s.AwsSession.Config.WithRegion(region)
	}
	awsregion.GuessRegion(s.AwsSession.Config)

	if *doWatch {
		err := s.WatchLifecycleEvents(func(m *ec2cluster.LifecycleMessage) (bool, error) {
			fmt.Printf("%s\t%s\n", m.LifecycleTransition, m.EC2InstanceID)
			return true, nil
		})
		if err != nil {
			log.Fatalf("ERROR: %s", err)
		}
		return
	}

	instance, err := s.Instance()
	if err != nil {
		log.Fatalf("ERROR: failed to inspect instance: %s", err)
	}

	availabilityZone := *instance.Placement.AvailabilityZone
	fmt.Printf("AVAILABILITY_ZONE=\"%s\"\n", availabilityZone)
	fmt.Printf("REGION=\"%s\"\n", availabilityZone[:len(availabilityZone)-1])
	if instance.PrivateIpAddress != nil {
		fmt.Printf("ADVERTISE_ADDRESS=\"%s\"\n", *instance.PrivateIpAddress)
	}
	for _, tag := range instance.Tags {
		fmt.Printf("TAG_%s=\"%s\"\n", strings.ToUpper(
			regexp.MustCompile("[^A-Za-z0-9]").ReplaceAllString(*tag.Key, "_")),
			strings.Replace(*tag.Value, "\"", "\\\"", -1))
	}

	asg, err := s.AutoscalingGroup()
	if err != nil {
		log.Fatalf("ERROR: %s", err)
	}
	if asg != nil {
		fmt.Printf("ASG_DESIRED_CAPACITY=%d\n", asg.DesiredCapacity)
		fmt.Printf("ASG_MIN_SIZE=%d\n", asg.MinSize)
		fmt.Printf("ASG_MAX_SIZE=%d\n", asg.MaxSize)
	}

	clusterInstances, err := s.Members()
	if err != nil {
		log.Fatalf("ERROR: %s", err)
	}

	clusterVar := []string{}
	for _, instance := range clusterInstances {
		if instance.PrivateIpAddress != nil {
			clusterVar = append(clusterVar, *instance.PrivateIpAddress)
		}
	}
	fmt.Printf("CLUSTER=\"%s\"\n", strings.Join(clusterVar, " "))
}
