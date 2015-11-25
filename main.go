package main

import (
	"fmt"

	"github.com/jessevdk/go-flags"
	"github.com/mackerelio/checkers"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elb"

	consul "github.com/hashicorp/consul/api"
	"github.com/hashicorp/serf/serf"
)

type options struct {
	LoadBalancerName string `long:"elb" description:"name of ELB" required:"true"`
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
	elb *elb.ELB
}

func newAWSClient(cfgs ...*aws.Config) (*awsClient, error) {
	session := session.New(cfgs...)
	return &awsClient{
		elb: elb.New(session),
		ec2: ec2.New(session),
	}, nil
}

func (awsClient *awsClient) listInServiceInstances(loadBalancerName string) ([]string, error) {
	result, err := awsClient.elb.DescribeInstanceHealth(&elb.DescribeInstanceHealthInput{
		LoadBalancerName: aws.String(loadBalancerName),
	})
	if err != nil {
		return nil, err
	}

	var instanceIds []string
	for _, instance := range result.InstanceStates {
		if aws.StringValue(instance.State) == "InService" {
			instanceIds = append(instanceIds, aws.StringValue(instance.InstanceId))
		}
	}

	return instanceIds, nil
}

func (awsClient *awsClient) getInstancesByIds(instanceIds []string) ([]*ec2.Instance, error) {
	var instances []*ec2.Instance

	err := awsClient.ec2.DescribeInstancesPages(&ec2.DescribeInstancesInput{
		InstanceIds: aws.StringSlice(instanceIds),
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

func findConsulMemberByIPAddress(members []*consul.AgentMember, ipAddress string) *consul.AgentMember {
	for _, member := range members {
		if member.Addr == ipAddress {
			return member
		}
	}

	return nil
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

	instanceIds, err := awsClient.listInServiceInstances(opts.LoadBalancerName)
	if err != nil {
		return checkers.Unknown(err.Error())
	}

	expectedInstances, err := awsClient.getInstancesByIds(instanceIds)
	if err != nil {
		return checkers.Unknown(err.Error())
	}

	members, err := consulAgent.Members(false)
	if err != nil {
		return checkers.Unknown(err.Error())
	}

	var missingInstances []*ec2.Instance
	for _, expectedInstance := range expectedInstances {
		member := findConsulMemberByIPAddress(members, aws.StringValue(expectedInstance.PrivateIpAddress))

		if member == nil || serf.MemberStatus(member.Status) != serf.StatusAlive {
			missingInstances = append(missingInstances, expectedInstance)
		}
	}

	if len(missingInstances) == 0 {
		return checkers.Ok("OK")
	} else {
		return checkers.Critical(fmt.Sprintf("Missing instances: %v", missingInstances))
	}
}
