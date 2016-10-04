package ec2cluster

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// DiscoverAdvertiseAddress returns the address that should be advertised for
// the current node based on the current EC2 instance's private IP address
func DiscoverAdvertiseAddress() (string, error) {
	return readMetadata("local-ipv4")
}

// DiscoverInstanceID returns an AWS instance ID or an empty string if the
// node is not running in EC2 or cannot reach the EC2 metadata service.
func DiscoverInstanceID() (string, error) {
	return readMetadata("instance-id")
}

// DiscoverAvailabilityZone returns an AWS availability zone or an empty string if the
// node is not running in EC2 or cannot reach the EC2 metadata service.
func DiscoverAvailabilityZone() (string, error) {
	return readMetadata("placement/availability-zone")
}

func readMetadata(suffix string) (string, error) {
	// a nice short timeout so we don't hang too much on non-AWS boxes
	client := *http.DefaultClient
	client.Timeout = 700 * time.Millisecond

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
