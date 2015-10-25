
# EC2 Cluster Auto-discovery for GO

[![Build Status](https://travis-ci.org/crewjam/ec2cluster.svg?branch=master)](https://travis-ci.org/crewjam/ec2cluster)

[![](https://godoc.org/github.com/crewjam/ec2cluster?status.png)](http://godoc.org/github.com/crewjam/ec2cluster)

This package provides simple cluster auto-discovery for go using EC2 metadata. This should be used to seed a more reliable group membership algorithm, such as `github.com/hashicorp/memberlist`

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