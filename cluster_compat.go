package ec2cluster

import "github.com/aws/aws-sdk-go/aws/session"

// DiscoverClusterMembersByTag returns a list of the private IP addresses of each
// EC2 instance having a tag that matches the tag on the currently running
// instance. The tag is specified by tagName.
//
// For example, if your instances are tagged with a tag named `app` whose
// value is unique per cluster, then you can invoke:
//
//     knownClusterMembers, err := DiscoverClusterMembersByTag(nil, "app")
//
func DiscoverClusterMembersByTag(awsSession *session.Session, tagName string) ([]string, error) {
	instanceID, err := DiscoverInstanceID()
	if err != nil {
		return nil, err
	}

	c := Cluster{
		AwsSession: awsSession,
		InstanceID: instanceID,
		TagName:    tagName,
	}

	clusterMembers, err := c.Members()
	if err != nil {
		return nil, err
	}

	rv := []string{}
	for _, member := range clusterMembers {
		if member.PrivateIpAddress != nil {
			rv = append(rv, *member.PrivateIpAddress)
		}
	}
	return rv, nil
}
