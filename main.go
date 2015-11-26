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

func newConsulAgent(config *consul.Config) (*consul.Agent, error) {
	client, err := consul.NewClient(config)
	if err != nil {
		return nil, err
	}
	return client.Agent(), nil
}

type awsClient struct {
	ec2 *ec2.EC2
}

func newAWSClient(cfgs ...*aws.Config) (*awsClient, error) {
	session := session.New(cfgs...)
	return &awsClient{
		ec2: ec2.New(session),
	}, nil
}

func (awsClient *awsClient) getInstancesWithTag(key string, value string) ([]*ec2.Instance, error) {
	var instances []*ec2.Instance

	err := awsClient.ec2.DescribeInstancesPages(&ec2.DescribeInstancesInput{
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

func (opts *options) run() *checkers.Checker {
	consulAgent, err := newConsulAgent(consul.DefaultConfig())
	if err != nil {
		return checkers.Unknown(err.Error())
	}

	awsClient, err := newAWSClient()
	if err != nil {
		return checkers.Unknown(err.Error())
	}

	ec2Instances, err := awsClient.getInstancesWithTag(opts.EC2Tag, opts.Value)
	if err != nil {
		return checkers.Unknown(err.Error())
	}

	consulMembers, err := getConsulMembersWithTag(consulAgent, opts.ConsulTag, opts.Value)
	if err != nil {
		return checkers.Unknown(err.Error())
	}

	_, missingMembers := diff(ec2Instances, consulMembers)

	if len(missingMembers) != 0 {
		return checkers.Critical(fmt.Sprintf("Missing members: %v", missingMembers))
	}

	return checkers.Ok("OK")
}
