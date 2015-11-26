package main

import (
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/mackerelio/checkers"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"

	consul "github.com/hashicorp/consul/api"
)

var VERSION string

type options struct {
	EC2TagKey        string `long:"ec2-tag" description:"tag name on EC2 instances" required:"true"`
	EC2TagValue      string `long:"ec2-value" description:"expected EC2 tag value" required:"true"`
	ConsulService    string `long:"consul-service" description:"name of Consul service" required:"true"`
	ConsulServiceTag string `long:"consul-tag" description:"tag name on Consul service"`
	ShowVersion      func() `long:"version" description:"Show version and exit"`
}

func main() {
	opts := &options{
		ShowVersion: func() {
			fmt.Printf("%s\n", VERSION)
			os.Exit(0)
		},
	}
	_, err := flags.Parse(opts)
	if err != nil {
		os.Exit(255)
	}

	checker := opts.run()
	checker.Exit()
}

func (opts *options) run() *checkers.Checker {
	missingInstances, missingMembers, err := check(opts)
	if err != nil {
		return checkers.Unknown(err.Error())
	}

	if len(missingInstances) != 0 {
		var ipAddrs []string
		for _, instance := range missingInstances {
			ipAddrs = append(ipAddrs, aws.StringValue(instance.PrivateIpAddress))
		}

		return checkers.Critical(fmt.Sprintf("%d instance(s) left from Consul cluster: %v", len(ipAddrs), ipAddrs))
	}

	if len(missingMembers) != 0 {
		var ipAddrs []string
		for _, member := range missingMembers {
			ipAddrs = append(ipAddrs, fmt.Sprintf("%s(%s)", member.Node, member.Address))
		}

		return checkers.Warning(fmt.Sprintf("%d instance(s) not properly tagged: %v", len(ipAddrs), ipAddrs))
	}

	return checkers.Ok("OK")
}

func check(opts *options) (missingInstances []*ec2.Instance, missingMembers []*consul.CatalogService, err error) {
	consulClient, err := opts.newConsulClient()
	if err != nil {
		return
	}

	ec2Client, err := opts.newEC2Client()
	if err != nil {
		return
	}

	ec2Instances, err := getInstancesWithTag(ec2Client, opts.EC2TagKey, opts.EC2TagValue)
	if err != nil {
		return
	}

	consulServices, err := getConsulServiceCatalog(consulClient, opts.ConsulService, opts.ConsulServiceTag)
	if err != nil {
		return
	}

	missingInstances, missingMembers = diff(ec2Instances, consulServices)
	return missingInstances, missingMembers, nil
}

func diff(instances []*ec2.Instance, members []*consul.CatalogService) (missingInstances []*ec2.Instance, missingMembers []*consul.CatalogService) {
	instanceMap := make(map[string]*ec2.Instance)
	memberMap := make(map[string]*consul.CatalogService)

	for _, instance := range instances {
		instanceMap[aws.StringValue(instance.PrivateIpAddress)] = instance
	}
	for _, member := range members {
		memberMap[member.Address] = member
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

func (opts *options) newConsulClient() (*consul.Client, error) {
	return consul.NewClient(consul.DefaultConfig())
}

func (opts *options) newEC2Client() (*ec2.EC2, error) {
	session := session.New()
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

func getConsulServiceCatalog(consulClient *consul.Client, name, tag string) ([]*consul.CatalogService, error) {
	catalog, _, err := consulClient.Catalog().Service(name, tag, nil)
	return catalog, err
}
