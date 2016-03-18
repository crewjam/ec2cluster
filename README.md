
# EC2 Cluster Auto-discovery for GO

[![Build Status](https://travis-ci.org/crewjam/ec2cluster.svg?branch=master)](https://travis-ci.org/crewjam/ec2cluster)

[![](https://godoc.org/github.com/crewjam/ec2cluster?status.png)](http://godoc.org/github.com/crewjam/ec2cluster)

This package provides simple cluster auto-discovery for go using EC2 metadata. This should be used to seed a more reliable group membership algorithm, such as `github.com/hashicorp/memberlist`. Most such algorithms require an initial
seed list of nodes, which is what ec2cluster provides.

In this example we use all the EC2 instances tagged with the same value for 
`app` as the initial known members for a memberlist cluster:

    ec2TagName := "app"
    knownMembers, err := DiscoverClusterMembersByTag(&aws.Config{}, ec2TagName)
    if err != nil {
        panic(err)
    }
    cluster := memberlist.Create(memberlist.DefaultLANConfig())
    _, err := cluster.Join(knownMembers)
    if err != nil {
        panic(err)
    }
    defer cluster.Leave(time.Second)

When run as a command this program produces output that can be used to set environment variables. You can invoke it like:

    eval $(ec2cluster)

The output it produces can be passed to the shell, or parsed:

    AVAILABILITY_ZONE="us-west-2a"
    REGION="us-west-2"
    ADVERTISE_ADDRESS="10.0.10.124"
    TAG_AWS_CLOUDFORMATION_STACK_ID="arn:aws:cloudformation:us-west-2:012345678901:stack/myapp/d0a9d5eb-dfed-4455-ab4b-5e7214f337bb"
    TAG_APPLICATION="myapp"
    TAG_AWS_AUTOSCALING_GROUPNAME="example-Cluster1-Q0YWRWQJC5XL"
    TAG_CLUSTER="myapp-cluster1"
    TAG_AWS_CLOUDFORMATION_STACK_NAME="myapp"
    TAG_AWS_CLOUDFORMATION_LOGICAL_ID="Cluster1"
    TAG_NAME="myapp.example.com-master"
    ASG_DESIRED_CAPACITY=3
    ASG_MIN_SIZE=1
    ASG_MAX_SIZE=5
    CLUSTER="10.0.10.124 10.0.188.207 10.0.118.6"

# Monitoring ASG Lifecycle events

You can also use this tool to monitor autoscaling lifecycle events. To do this, configure your autoscaling group with a lifecycle hook that emits events to an SQS queue. Then invoke `ec2cluster --watch` which will produce one line of output per event, like:

    autoscaling:EC2_INSTANCE_TERMINATING    i-403e6d87

