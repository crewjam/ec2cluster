package ec2cluster

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
        "github.com/aws/aws-sdk-go/aws/session"
	. "gopkg.in/check.v1"
)

func randomPort() uint16 {
	l, _ := net.Listen("tcp", ":0")
	defer l.Close()
	port := l.Addr().(*net.TCPAddr).Port
	return uint16(port)
}

type ClusterTest struct {
}

var _ = Suite(&ClusterTest{})

func (s *ClusterTest) TestNotInEC2(c *C) {
	t := *(http.DefaultTransport.(*http.Transport))
	t.Dial = func(network, addr string) (net.Conn, error) {
		return net.Dial(network, "0.0.0.0:9")
	}
	http.DefaultClient.Transport = &t
	defer func() { http.DefaultClient.Transport = nil }()

	addr, err := DiscoverAdvertiseAddress()
	c.Assert(err, ErrorMatches, ".*: connection refused$")
	c.Assert(addr, Equals, "")
}

func (s *ClusterTest) TestDiscovery(c *C) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requestCount == 0 {
			c.Assert(r.Host, Equals, "169.254.169.254")
			c.Assert(r.URL.String(), Equals, "/latest/meta-data/instance-id")
			fmt.Fprintf(w, "i-1a2b3c4d")
		}

		if requestCount == 1 {
			r.ParseForm()
			c.Assert(strings.HasPrefix(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256"), Equals, true)
			c.Assert(r.Form.Get("Action"), Equals, "DescribeInstances")
			c.Assert(r.Form.Get("InstanceId.1"), Equals, "i-1a2b3c4d")
			fmt.Fprintf(w, `<DescribeInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2014-10-01/">
			  <requestId>fdcdcab1-ae5c-489e-9c33-4637c5dda355</requestId>
				<reservationSet>
				  <item>
					<reservationId>r-1a2b3c4d</reservationId>
					<ownerId>123456789012</ownerId>
					<groupSet>
					  <item>
						<groupId>sg-1a2b3c4d</groupId>
						<groupName>my-security-group</groupName>
					  </item>
					</groupSet>
					<instancesSet>
					  <item>
						<instanceId>i-1a2b3c4d</instanceId>
						<imageId>ami-1a2b3c4d</imageId>
						<tagSet>
						  <item>
							<key>Name</key>
							<value>Windows Instance</value>
						  </item>
						</tagSet>
					  </item>
					</instancesSet>
				  </item>
				</reservationSet>
				</DescribeInstancesResponse>`)
		}
		if requestCount == 2 {
			r.ParseForm()
			c.Assert(strings.HasPrefix(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256"), Equals, true)
			c.Assert(r.Form.Get("Action"), Equals, "DescribeInstances")
			c.Assert(r.Form.Get("Filter.1.Name"), Equals, "tag:Name")
			c.Assert(r.Form.Get("Filter.1.Value.1"), Equals, "Windows Instance")
			fmt.Fprintf(w, `<DescribeInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2014-10-01/">
			  <requestId>fdcdcab1-ae5c-489e-9c33-4637c5dda355</requestId>
				<reservationSet>
				  <item>
					<instancesSet>
					  <item>
						<instanceId>i-1a2b3c4d</instanceId>
						<privateIpAddress>10.0.0.12</privateIpAddress>
					  </item>
					</instancesSet>
				  </item>
				  <item>
					<instancesSet>
					  <item>
						<instanceId>i-fefefefe</instanceId>
						<privateIpAddress>10.0.0.13</privateIpAddress>
					  </item>
					</instancesSet>
				  </item>
				</reservationSet>
			</DescribeInstancesResponse>
        	`)
		}
		if requestCount == 3 {
			c.Assert(r.Host, Equals, "169.254.169.254")
			c.Assert(r.URL.String(), Equals, "/latest/meta-data/local-ipv4")
			fmt.Fprintf(w, "10.0.0.99")
		}

		if requestCount >= 4 {
			w.WriteHeader(404)
			c.Assert(true, IsNil) // should not be reached
		}
		requestCount++
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	c.Assert(err, IsNil)

	t := *(http.DefaultTransport.(*http.Transport))
	t.Dial = func(network, addr string) (net.Conn, error) {
		return net.Dial(network, serverURL.Host)
	}
	http.DefaultClient.Transport = &t
	defer func() { http.DefaultClient.Transport = nil }()

	awsConfig := aws.NewConfig()
	awsConfig.WithDisableSSL(true)
	awsConfig.WithRegion("us-east-1")
	awsConfig.WithCredentials(
		credentials.NewStaticCredentials("AKIAJJJJJJJJJJJJJJJ", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", ""))

	awsSession := session.New(awsConfig)

	rv, err := DiscoverClusterMembersByTag(awsSession, "Name")
	c.Assert(err, IsNil)
	c.Assert(rv, DeepEquals, []string{"10.0.0.12", "10.0.0.13"})

	addr, err := DiscoverAdvertiseAddress()
	c.Assert(err, IsNil)
	c.Assert(addr, Equals, "10.0.0.99")
}
