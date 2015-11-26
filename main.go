package main

import (
	"fmt"

	"github.com/jessevdk/go-flags"
	"github.com/mackerelio/checkers"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"

	consul "github.com/hashicorp/consul/api"
	"github.com/hashicorp/serf/serf"
)

type options struct {
	EC2Tag    string `long:"ec2-tag" description:"tag name on EC2 instances" required:"true"`
	ConsulTag string `long:"consul-tag" description:"tag name on Consul agents" required:"true"`
	Value     string `long:"value" description:"expected tag value" required:"true"`
}

func main() {
	opts := &options{}
	_, err := flags.Parse(opts)
	if err != nil {
		panic(err)
	}

	checker := opts.run()
	checker.Exit()
}

func (opts *options) run() *checkers.Checker {
	missingInstances, _, err := check(opts)
	if err != nil {
		return checkers.Unknown(err.Error())
	}

	if len(missingInstances) != 0 {
		var ipAddrs []string
		for _, instance := range missingInstances {
			ipAddrs = append(ipAddrs, aws.StringValue(instance.PrivateIpAddress))
		}

		return checkers.Critical(fmt.Sprintf("Missing members: %v", ipAddrs))
	}

	return checkers.Ok("OK")
}

func check(opts *options) (missingInstances []*ec2.Instance, missingMembers []*consul.AgentMember, err error) {
	consulAgent, err := newConsulAgent(consul.DefaultConfig())
	if err != nil {
		return
	}

	ec2Client, err := newEC2Client()
	if err != nil {
		return
	}

	ec2Instances, err := getInstancesWithTag(ec2Client, opts.EC2Tag, opts.Value)
	if err != nil {
		return
	}

	consulMembers, err := getConsulMembersWithTag(consulAgent, opts.ConsulTag, opts.Value)
	if err != nil {
		return
	}

	missingInstances, missingMembers = diff(ec2Instances, consulMembers)
	return missingInstances, missingMembers, nil
}

func diff(instances []*ec2.Instance, members []*consul.AgentMember) (missingInstances []*ec2.Instance, missingMembers []*consul.AgentMember) {
	instanceMap := make(map[string]*ec2.Instance)
	memberMap := make(map[string]*consul.AgentMember)

	for _, instance := range instances {
		instanceMap[aws.StringValue(instance.PrivateIpAddress)] = instance
	}
	for _, member := range members {
		memberMap[member.Addr] = member
	}

	for addr, instance := range instanceMap {
		_, ok := memberMap[addr]
		if !ok {
			missingInstances = append(missingInstances, instance)
		}
	}

	for addr, member := range memberMap {
		_, ok := instanceMap[addr]
		if !ok {
			missingMembers = append(missingMembers, member)
		}
	}

	return
}

func newConsulAgent(config *consul.Config) (*consul.Agent, error) {
	client, err := consul.NewClient(config)
	if err != nil {
		return nil, err
	}
	return client.Agent(), nil
}

func newEC2Client(cfgs ...*aws.Config) (*ec2.EC2, error) {
	session := session.New(cfgs...)
	return ec2.New(session), nil
}

func getInstancesWithTag(ec2Client *ec2.EC2, key string, value string) ([]*ec2.Instance, error) {
	var instances []*ec2.Instance

	err := ec2Client.DescribeInstancesPages(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String(fmt.Sprintf("tag:%s", key)),
				Values: []*string{aws.String(value)},
			},
		},
	}, func(result *ec2.DescribeInstancesOutput, _ bool) bool {
		for _, reservation := range result.Reservations {
			instances = append(instances, reservation.Instances...)
		}
		return true // keep going
	})
	if err != nil {
		return nil, err
	}

	return instances, nil
}

func getConsulMembersWithTag(consulAgent *consul.Agent, key string, value string) ([]*consul.AgentMember, error) {
	allMembers, err := consulAgent.Members(false)
	if err != nil {
		return nil, err
	}

	var members []*consul.AgentMember
	for _, member := range allMembers {
		if serf.MemberStatus(member.Status) == serf.StatusAlive && member.Tags[key] == value {
			members = append(members, member)
		}
	}

	return members, nil
}
